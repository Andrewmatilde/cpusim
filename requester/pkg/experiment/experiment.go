package experiment

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cpusim/requester/api/generated"
	"cpusim/requester/pkg/storage"
)

// Experiment represents a request sending experiment
type Experiment struct {
	config     generated.StartRequestExperimentRequest
	status     generated.RequestExperimentStatus
	startTime  time.Time
	endTime    *time.Time
	stats      *RequestStats
	httpClient *http.Client
	storage    *storage.FileStorage
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{} // Signals that Start() has finished
	mu         sync.RWMutex
}

// NewExperiment creates a new experiment
func NewExperiment(config generated.StartRequestExperimentRequest, storage *storage.FileStorage) *Experiment {
	// Create context with timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc
	if config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

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
		status:     generated.RequestExperimentStatusRunning,
		stats:      NewRequestStats(),
		httpClient: httpClient,
		storage:    storage,
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
}

// Start starts the experiment
func (e *Experiment) Start() {
	e.mu.Lock()
	e.startTime = time.Now()
	e.mu.Unlock()

	// Close done channel when finished
	defer close(e.done)

	// Calculate QPS interval
	qps := e.config.Qps
	if qps <= 0 {
		qps = 1 // Fallback to 1 QPS if invalid
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

		case <-e.ctx.Done():
			// Immediately set end time and status when loop exits
			e.mu.Lock()
			now := time.Now()
			e.endTime = &now
			if e.ctx.Err() == context.DeadlineExceeded {
				e.status = generated.RequestExperimentStatusCompleted // Timeout
			} else {
				e.status = generated.RequestExperimentStatusStopped // Manual stop
			}
			e.mu.Unlock()

			// Save experiment data to storage
			experimentData := e.ToRequestExperiment()
			stats := e.GetStats()
			if err := e.storage.SaveExperiment(experimentData, stats); err != nil {
				fmt.Printf("Warning: failed to save experiment data: %v\n", err)
			}

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
	e.mu.RLock()
	status := e.status
	e.mu.RUnlock()

	// Check if already stopped (idempotent)
	if status == generated.RequestExperimentStatusStopped || status == generated.RequestExperimentStatusCompleted {
		return e.getStopResult(), nil
	}

	// Trigger stop by cancelling context
	e.cancel()

	// Wait for Start() goroutine to finish
	<-e.done

	return e.getStopResult(), nil
}

// getStopResult builds the stop result from current state
func (e *Experiment) getStopResult() *generated.StopExperimentResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var duration int
	if e.endTime != nil {
		duration = int(e.endTime.Sub(e.startTime).Seconds())
	}

	var endTime time.Time
	if e.endTime != nil {
		endTime = *e.endTime
	}

	return &generated.StopExperimentResult{
		ExperimentId: e.config.ExperimentId,
		EndTime:      endTime,
		Duration:     duration,
		FinalStats:   *e.stats.ToRequestExperimentStats(e.config.ExperimentId, string(e.status), e.startTime, e.endTime),
	}
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

	var duration int
	if e.endTime != nil {
		duration = int(e.endTime.Sub(e.startTime).Seconds())
	}

	var endTime time.Time
	if e.endTime != nil {
		endTime = *e.endTime
	}

	return &generated.RequestExperiment{
		ExperimentId: e.config.ExperimentId,
		TargetIP:     e.config.TargetIP,
		TargetPort:   e.config.TargetPort,
		Timeout:      e.config.Timeout,
		Qps:          e.config.Qps,
		Description:  e.config.Description,
		Status:       e.status, // Directly use status since it's already the correct type
		StartTime:    e.startTime,
		EndTime:      endTime,
		Duration:     duration,
		CreatedAt:    e.startTime,
	}
}

// IsCompleted returns true if the experiment is completed
func (e *Experiment) IsCompleted() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status == generated.RequestExperimentStatusCompleted || e.status == generated.RequestExperimentStatusStopped
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
