package collector

import (
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestService_BasicFlow(t *testing.T) {
	// Create temporary directory for test data
	tempDir := t.TempDir()

	// Create logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Define collector config
	config := Config{
		CollectionInterval: 1, // 1 second
		CalculatorProcess:  "cpusim-server",
	}

	// Create service with config
	service, err := NewService(tempDir, config, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Start experiment
	experimentID := "test-collector-exp-1"
	timeout := 3 * time.Second

	t.Logf("Starting experiment: %s", experimentID)
	err = service.StartExperiment(experimentID, timeout)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}

	// Wait for experiment to complete
	t.Logf("Waiting for experiment to complete...")
	time.Sleep(timeout + 1*time.Second)

	// Retrieve experiment data
	t.Logf("Retrieving experiment data...")
	data, err := service.GetExperiment(experimentID)
	if err != nil {
		t.Fatalf("Failed to get experiment: %v", err)
	}

	// Verify results
	if data == nil {
		t.Fatal("Expected non-nil data")
	}

	t.Logf("Experiment Results:")
	t.Logf("  Duration: %.2f seconds", data.Duration)
	t.Logf("  Data Points Collected: %d", data.DataPointsCollected)

	if len(data.Metrics) > 0 {
		lastMetric := data.Metrics[len(data.Metrics)-1]
		t.Logf("  Last CPU Usage: %.2f%%", lastMetric.CPUUsagePercent)
		t.Logf("  Last Memory Usage: %.2f%%", lastMetric.MemoryUsagePercent)
		t.Logf("  Calculator Service Healthy: %v", lastMetric.CalculatorServiceHealthy)
	}

	// Basic assertions
	if data.DataPointsCollected == 0 {
		t.Error("Expected some data points to be collected")
	}

	if data.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// Should have collected at least 2-3 data points in 3 seconds with 1 second interval
	if data.DataPointsCollected < 2 {
		t.Errorf("Expected at least 2 data points, got %d", data.DataPointsCollected)
	}
}

func TestService_StopExperiment(t *testing.T) {
	t.Skip("Skipping integration test - takes too long")

	tempDir := t.TempDir()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	config := Config{
		CollectionInterval: 1,
		CalculatorProcess:  "cpusim-server",
	}

	service, err := NewService(tempDir, config, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	experimentID := "test-collector-stop"
	timeout := 10 * time.Second

	// Start experiment
	err = service.StartExperiment(experimentID, timeout)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}

	// Let it run for a bit
	time.Sleep(2 * time.Second)

	// Stop experiment
	t.Logf("Stopping experiment...")
	err = service.StopExperiment()
	if err != nil {
		t.Fatalf("Failed to stop experiment: %v", err)
	}

	// Wait a bit for cleanup
	time.Sleep(500 * time.Millisecond)

	// Retrieve experiment data
	data, err := service.GetExperiment(experimentID)
	if err != nil {
		t.Fatalf("Failed to get experiment: %v", err)
	}

	t.Logf("Stopped experiment results:")
	t.Logf("  Duration: %.2f seconds", data.Duration)
	t.Logf("  Data Points Collected: %d", data.DataPointsCollected)

	// Duration should be less than the full timeout since we stopped it early
	if data.Duration >= 10.0 {
		t.Errorf("Expected duration to be less than 10s, got %.2f", data.Duration)
	}

	if data.DataPointsCollected == 0 {
		t.Error("Expected some data points to be collected before stopping")
	}
}
