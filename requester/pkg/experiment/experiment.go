package experiment

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cpusim/requester/api/generated"
)

// ExperimentStatus represents the status of an experiment
type ExperimentStatus string

const (
	StatusRunning   ExperimentStatus = "running"
	StatusStopped   ExperimentStatus = "stopped"
	StatusCompleted ExperimentStatus = "completed"
	StatusError     ExperimentStatus = "error"
)

// Experiment represents a request sending experiment
type Experiment struct {
	config     generated.StartRequestExperimentRequest
	status     ExperimentStatus
	startTime  time.Time
	endTime    *time.Time
	stats      *RequestStats
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	stopChan   chan struct{}
	mu         sync.RWMutex
}

// NewExperiment creates a new experiment
func NewExperiment(config generated.StartRequestExperimentRequest) *Experiment {
	ctx, cancel := context.WithCancel(context.Background())

	// Configure HTTP transport for high concurrency
	transport := &http.Transport{
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 100,              // 每个host最大空闲连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时
		DisableKeepAlives:   false,            // 启用连接复用
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second, // 单个请求超时
	}

	return &Experiment{
		config:     config,
		status:     StatusRunning,
		stats:      NewRequestStats(),
		httpClient: httpClient,
		ctx:        ctx,
		cancel:     cancel,
		stopChan:   make(chan struct{}),
	}
}

// Start starts the experiment
func (e *Experiment) Start() {
	e.mu.Lock()
	e.startTime = time.Now()
	e.mu.Unlock()

	// Set up timeout if specified
	var timeoutTimer *time.Timer
	if e.config.Timeout > 0 {
		timeoutTimer = time.NewTimer(time.Duration(e.config.Timeout) * time.Second)
		defer timeoutTimer.Stop()
	}

	// Calculate QPS interval
	qps := e.config.Qps
	if qps <= 0 {
		qps = 1  // Fallback to 1 QPS if invalid
	}

	interval := time.Second / time.Duration(qps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	targetURL := fmt.Sprintf("http://%s:%d/calculate", e.config.TargetIP, e.config.TargetPort)

	for {
		select {
		case <-ticker.C:
			// Send request in goroutine to maintain QPS timing
			go e.sendRequest(targetURL)

		case <-e.stopChan:
			e.setStatus(StatusStopped)
			return

		case <-timeoutTimer.C:
			e.setStatus(StatusCompleted)
			return

		case <-e.ctx.Done():
			e.setStatus(StatusStopped)
			return
		}
	}
}

// sendRequest sends a single HTTP request
func (e *Experiment) sendRequest(targetURL string) {
	startTime := time.Now()

	// Create request with empty JSON body
	req, err := http.NewRequestWithContext(e.ctx, "POST", targetURL, bytes.NewBufferString("{}"))
	if err != nil {
		e.stats.RecordRequest(0, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := e.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		e.stats.RecordRequest(duration, err)
		return
	}

	resp.Body.Close()

	// Record successful request
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		e.stats.RecordRequest(duration, nil)
	} else {
		e.stats.RecordRequest(duration, fmt.Errorf("HTTP %d", resp.StatusCode))
	}
}

// Stop stops the experiment
func (e *Experiment) Stop() (*generated.StopExperimentResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == StatusStopped || e.status == StatusCompleted {
		return nil, fmt.Errorf("experiment already stopped")
	}

	// Signal stop
	close(e.stopChan)
	e.cancel()

	// Set end time
	now := time.Now()
	e.endTime = &now
	e.status = StatusStopped

	// Calculate duration
	duration := int(e.endTime.Sub(e.startTime).Seconds())

	return &generated.StopExperimentResult{
		ExperimentId: &e.config.ExperimentId,
		StopStatus:   stringPtr("stopped"),
		EndTime:      e.endTime,
		Duration:     &duration,
		FinalStats:   e.stats.ToRequestExperimentStats(e.config.ExperimentId, string(e.status), e.startTime, e.endTime),
	}, nil
}

// GetStats returns the current statistics
func (e *Experiment) GetStats() *generated.RequestExperimentStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.stats.ToRequestExperimentStats(e.config.ExperimentId, string(e.status), e.startTime, e.endTime)
}

// ToRequestExperiment converts to API response format
func (e *Experiment) ToRequestExperiment() *generated.RequestExperiment {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var duration *int
	if e.endTime != nil {
		d := int(e.endTime.Sub(e.startTime).Seconds())
		duration = &d
	}

	var endTimePtr *time.Time
	if e.endTime != nil {
		endTimePtr = e.endTime
	}

	var status generated.RequestExperimentStatus
	switch e.status {
	case StatusRunning:
		status = generated.RequestExperimentStatusRunning
	case StatusStopped:
		status = generated.RequestExperimentStatusStopped
	case StatusCompleted:
		status = generated.RequestExperimentStatusCompleted
	case StatusError:
		status = generated.RequestExperimentStatusError
	}

	return &generated.RequestExperiment{
		ExperimentId: &e.config.ExperimentId,
		TargetIP:     &e.config.TargetIP,
		TargetPort:   &e.config.TargetPort,
		Timeout:      &e.config.Timeout,
		Qps:          &e.config.Qps,
		Description:  e.config.Description,
		Status:       &status,
		StartTime:    &e.startTime,
		EndTime:      endTimePtr,
		Duration:     duration,
		CreatedAt:    &e.startTime,
	}
}

// IsCompleted returns true if the experiment is completed
func (e *Experiment) IsCompleted() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status == StatusCompleted || e.status == StatusStopped
}

// GetEndTime returns the end time if available
func (e *Experiment) GetEndTime() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.endTime != nil {
		return *e.endTime
	}
	return time.Time{}
}

// setStatus sets the experiment status
func (e *Experiment) setStatus(status ExperimentStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.status = status
	if status == StatusCompleted || status == StatusStopped || status == StatusError {
		if e.endTime == nil {
			now := time.Now()
			e.endTime = &now
		}
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}