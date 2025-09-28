package metrics

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

// SystemMetrics represents the current system metrics
type SystemMetrics struct {
	CPUUsagePercent     float64   `json:"cpuUsagePercent"`
	MemoryUsageBytes    int64     `json:"memoryUsageBytes"`
	MemoryUsagePercent  float64   `json:"memoryUsagePercent"`
	NetworkIOBytes      NetworkIO `json:"networkIOBytes"`
	CalculatorServiceHealthy bool `json:"calculatorServiceHealthy"`
	Timestamp           time.Time `json:"timestamp"`
}

// NetworkIO represents network I/O statistics
type NetworkIO struct {
	BytesReceived    int64 `json:"bytesReceived"`
	BytesSent        int64 `json:"bytesSent"`
	PacketsReceived  int64 `json:"packetsReceived"`
	PacketsSent      int64 `json:"packetsSent"`
}

// Collector handles system metrics collection
type Collector struct {
	calculatorProcessName string
	lastNetStats         []net.IOCountersStat
	lastCPUStats         []cpu.TimesStat
	lastCPUTime          time.Time
}

// NewCollector creates a new metrics collector
func NewCollector(calculatorProcessName string) *Collector {
	return &Collector{
		calculatorProcessName: calculatorProcessName,
	}
}

// GetCurrentMetrics collects and returns current system metrics
func (c *Collector) GetCurrentMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// Collect CPU usage
	cpuPercent, err := c.getCPUUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}
	metrics.CPUUsagePercent = cpuPercent

	// Collect memory usage
	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage: %w", err)
	}
	metrics.MemoryUsageBytes = int64(memInfo.Used)
	metrics.MemoryUsagePercent = memInfo.UsedPercent

	// Collect network I/O
	networkIO, err := c.getNetworkIO(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get network I/O: %w", err)
	}
	metrics.NetworkIOBytes = *networkIO

	// Check calculator service health by process
	healthy := c.checkCalculatorProcessHealth(ctx)
	metrics.CalculatorServiceHealthy = healthy

	return metrics, nil
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

	networkIO := &NetworkIO{
		BytesReceived:   int64(currentStats[0].BytesRecv),
		BytesSent:       int64(currentStats[0].BytesSent),
		PacketsReceived: int64(currentStats[0].PacketsRecv),
		PacketsSent:     int64(currentStats[0].PacketsSent),
	}

	// If we have previous stats, calculate the rate
	if len(c.lastNetStats) > 0 {
		lastStat := c.lastNetStats[0]
		networkIO.BytesReceived = int64(currentStats[0].BytesRecv - lastStat.BytesRecv)
		networkIO.BytesSent = int64(currentStats[0].BytesSent - lastStat.BytesSent)
		networkIO.PacketsReceived = int64(currentStats[0].PacketsRecv - lastStat.PacketsRecv)
		networkIO.PacketsSent = int64(currentStats[0].PacketsSent - lastStat.PacketsSent)
	}

	// Store current stats for next calculation
	c.lastNetStats = currentStats

	return networkIO, nil
}

// checkCalculatorProcessHealth checks if the calculator process is running
func (c *Collector) checkCalculatorProcessHealth(ctx context.Context) bool {
	if c.calculatorProcessName == "" {
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
		if strings.Contains(name, c.calculatorProcessName) {
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