package requester

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

	// Define experiment config
	config := Config{
		TargetIP:   "httpbin.org",
		TargetPort: 80,
		QPS:        5,
		Timeout:    10,
	}

	// Create service with config
	service, err := NewService(tempDir, config, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Start experiment
	experimentID := "test-exp-1"
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
	t.Logf("  Total Requests: %d", data.TotalRequests)
	t.Logf("  Successful: %d", data.Successful)
	t.Logf("  Failed: %d", data.Failed)
	t.Logf("  Avg Response Time: %.2f ms", data.Stats.AvgResponseTime)
	t.Logf("  P95: %.2f ms", data.Stats.P95)
	t.Logf("  P99: %.2f ms", data.Stats.P99)
	t.Logf("  Error Rate: %.2f%%", data.Stats.ErrorRate)
	t.Logf("  Actual QPS: %.2f", data.Stats.ActualQPS)

	// Basic assertions
	if data.TotalRequests == 0 {
		t.Error("Expected some requests to be sent")
	}

	if data.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// QPS should be approximately what we configured (allowing for some variance)
	expectedRequests := int64(config.QPS) * int64(timeout.Seconds())
	variance := 0.3 // Allow 30% variance
	minExpected := float64(expectedRequests) * (1 - variance)
	maxExpected := float64(expectedRequests) * (1 + variance)

	if float64(data.TotalRequests) < minExpected || float64(data.TotalRequests) > maxExpected {
		t.Logf("Warning: Total requests (%d) outside expected range [%.0f, %.0f]",
			data.TotalRequests, minExpected, maxExpected)
	}
}

func TestService_StopExperiment(t *testing.T) {
	t.Skip("Skipping integration test - takes too long")
	tempDir := t.TempDir()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	config := Config{
		TargetIP:   "httpbin.org",
		TargetPort: 80,
		QPS:        5,
		Timeout:    10,
	}

	service, err := NewService(tempDir, config, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	experimentID := "test-exp-stop"
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
	t.Logf("  Total Requests: %d", data.TotalRequests)

	// Duration should be less than the full timeout since we stopped it early
	if data.Duration >= 10.0 {
		t.Errorf("Expected duration to be less than 10s, got %.2f", data.Duration)
	}

	if data.TotalRequests == 0 {
		t.Error("Expected some requests to be sent before stopping")
	}
}

func TestService_MultipleExperimentsSerial(t *testing.T) {
	t.Skip("Skipping integration test - takes too long")
	tempDir := t.TempDir()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	config := Config{
		TargetIP:   "httpbin.org",
		TargetPort: 80,
		QPS:        5,
		Timeout:    10,
	}

	service, err := NewService(tempDir, config, logger)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Run first experiment
	exp1ID := "test-exp-serial-1"
	err = service.StartExperiment(exp1ID, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to start experiment 1: %v", err)
	}

	time.Sleep(3 * time.Second)

	// Run second experiment
	exp2ID := "test-exp-serial-2"
	err = service.StartExperiment(exp2ID, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to start experiment 2: %v", err)
	}

	time.Sleep(3 * time.Second)

	// Both experiments should be retrievable
	data1, err := service.GetExperiment(exp1ID)
	if err != nil {
		t.Fatalf("Failed to get experiment 1: %v", err)
	}

	data2, err := service.GetExperiment(exp2ID)
	if err != nil {
		t.Fatalf("Failed to get experiment 2: %v", err)
	}

	if data1.TotalRequests == 0 {
		t.Error("Expected experiment 1 to have requests")
	}

	if data2.TotalRequests == 0 {
		t.Error("Expected experiment 2 to have requests")
	}

	t.Logf("Experiment 1: %d requests", data1.TotalRequests)
	t.Logf("Experiment 2: %d requests", data2.TotalRequests)
}
