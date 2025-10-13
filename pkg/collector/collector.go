package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Collector handles system metrics collection
type Collector struct {
	config           Config
	lastNetStats     []net.IOCountersStat
	lastCPUStats     []cpu.TimesStat
	lastCPUTime      time.Time
}

// NewCollector creates a new metrics collector
func NewCollector(config Config) *Collector {
	return &Collector{
		config: config,
	}
}

// Run collects metrics for the duration specified in context
func (c *Collector) Run(ctx context.Context) (*MetricsData, error) {
	data := &MetricsData{
		Config:    c.config,
		StartTime: time.Now(),
		Metrics:   make([]MetricDataPoint, 0),
	}

	// Collection interval
	interval := time.Duration(c.config.CollectionInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect metrics immediately at start
	if metric, err := c.collectSinglePoint(ctx); err == nil {
		data.Metrics = append(data.Metrics, *metric)
	}

	// Continue collecting until context is done
	for {
		select {
		case <-ctx.Done():
			data.EndTime = time.Now()
			data.Duration = data.EndTime.Sub(data.StartTime).Seconds()
			data.DataPointsCollected = len(data.Metrics)
			return data, nil

		case <-ticker.C:
			metric, err := c.collectSinglePoint(ctx)
			if err != nil {
				fmt.Printf("Error collecting metrics: %v\n", err)
				continue
			}
			data.Metrics = append(data.Metrics, *metric)
		}
	}
}

// collectSinglePoint collects a single metric data point
func (c *Collector) collectSinglePoint(ctx context.Context) (*MetricDataPoint, error) {
	metric := &MetricDataPoint{
		Timestamp: time.Now(),
	}

	// Collect CPU usage (best effort, don't fail on error)
	cpuPercent, err := c.getCPUUsage(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to get CPU usage: %v\n", err)
	} else {
		metric.CPUUsagePercent = cpuPercent
	}

	// Collect memory usage
	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to get memory usage: %v\n", err)
	} else {
		metric.MemoryUsageBytes = int64(memInfo.Used)
		metric.MemoryUsagePercent = memInfo.UsedPercent
	}

	// Collect network I/O (best effort)
	networkIO, err := c.getNetworkIO(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to get network I/O: %v\n", err)
	} else {
		metric.NetworkIOBytes = *networkIO
	}

	// Check calculator service health by process
	healthy := c.checkCalculatorProcessHealth(ctx)
	metric.CalculatorServiceHealthy = healthy

	return metric, nil
}

// getNetworkIO calculates network I/O rates per second
func (c *Collector) getNetworkIO(ctx context.Context) (*NetworkIO, error) {
	currentStats, err := net.IOCountersWithContext(ctx, false)
	if err != nil {
		return nil, err
	}

	if len(currentStats) == 0 {
		return &NetworkIO{}, nil
	}

	// If this is the first call, initialize lastNetStats and return zero values
	if len(c.lastNetStats) == 0 {
		c.lastNetStats = currentStats
		return &NetworkIO{
			BytesReceived:   0,
			BytesSent:       0,
			PacketsReceived: 0,
			PacketsSent:     0,
		}, nil
	}

	// Calculate the rate based on difference from last measurement
	lastStat := c.lastNetStats[0]
	networkIO := &NetworkIO{
		BytesReceived:   int64(currentStats[0].BytesRecv - lastStat.BytesRecv),
		BytesSent:       int64(currentStats[0].BytesSent - lastStat.BytesSent),
		PacketsReceived: int64(currentStats[0].PacketsRecv - lastStat.PacketsRecv),
		PacketsSent:     int64(currentStats[0].PacketsSent - lastStat.PacketsSent),
	}

	// Store current stats for next calculation
	c.lastNetStats = currentStats

	return networkIO, nil
}

// checkCalculatorProcessHealth checks if the calculator process is running
func (c *Collector) checkCalculatorProcessHealth(ctx context.Context) bool {
	if c.config.CalculatorProcess == "" {
		return false
	}

	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return false
	}

	for _, proc := range processes {
		name, err := proc.NameWithContext(ctx)
		if err != nil {
			continue
		}

		// Check if process name contains the calculator process name
		if strings.Contains(name, c.config.CalculatorProcess) {
			// Additional check to ensure the process is running
			status, err := proc.StatusWithContext(ctx)
			if err != nil {
				continue
			}

			// Consider process healthy if it's running, sleeping, or idle
			// status is returned as []string, check the first element
			if len(status) > 0 && (status[0] == "R" || status[0] == "S" || status[0] == "I") {
				return true
			}
		}
	}

	return false
}

// getCPUUsage calculates CPU usage percentage based on time differences
func (c *Collector) getCPUUsage(ctx context.Context) (float64, error) {
	currentStats, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		return 0, err
	}

	currentTime := time.Now()

	// If we don't have previous stats, return 0 and store current stats
	if len(c.lastCPUStats) == 0 {
		c.lastCPUStats = currentStats
		c.lastCPUTime = currentTime
		return 0, nil
	}

	if len(currentStats) == 0 || len(c.lastCPUStats) == 0 {
		return 0, nil
	}

	// Calculate time difference
	timeDelta := currentTime.Sub(c.lastCPUTime).Seconds()
	if timeDelta <= 0 {
		return 0, nil
	}

	// Get current and last CPU stats (using first CPU core for overall system usage)
	current := currentStats[0]
	last := c.lastCPUStats[0]

	// Calculate total time differences
	totalCurrent := current.User + current.System + current.Nice + current.Iowait + current.Irq + current.Softirq + current.Steal + current.Idle
	totalLast := last.User + last.System + last.Nice + last.Iowait + last.Irq + last.Softirq + last.Steal + last.Idle

	totalDelta := totalCurrent - totalLast
	if totalDelta <= 0 {
		return 0, nil
	}

	// Calculate idle time difference
	idleDelta := current.Idle - last.Idle

	// Calculate CPU usage percentage
	cpuUsage := (1.0 - (idleDelta / totalDelta)) * 100.0

	// Store current stats for next calculation
	c.lastCPUStats = currentStats
	c.lastCPUTime = currentTime

	// Ensure the value is within valid range
	if cpuUsage < 0 {
		cpuUsage = 0
	} else if cpuUsage > 100 {
		cpuUsage = 100
	}

	return cpuUsage, nil
}
