package experiment

import (
	"fmt"
	"testing"
	"time"

	"cpusim/requester/api/generated"
	"cpusim/requester/pkg/storage"
)

// setupTestManager creates a manager with a temporary storage directory
func setupTestManager(t *testing.T) *Manager {
	t.Helper()

	// Create temporary directory for storage
	tempDir := t.TempDir()

	// Create storage
	fileStorage, err := storage.NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	// Create manager with test storage
	manager := &Manager{
		currentExperiment: nil,
		storage:           fileStorage,
	}

	return manager
}

// createTestRequest creates a test experiment request
func createTestRequest(experimentID string) generated.StartRequestExperimentRequest {
	return generated.StartRequestExperimentRequest{
		ExperimentId: experimentID,
		TargetIP:     "localhost",
		TargetPort:   8080,
		Qps:          10,
		Timeout:      30, // 30 seconds timeout
		Description:  "Test experiment",
	}
}

// TestStartExperiment_NewExperiment tests starting a new experiment
func TestStartExperiment_NewExperiment(t *testing.T) {
	manager := setupTestManager(t)

	request := createTestRequest("test-exp-1")
	exp, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	if exp.ExperimentId != "test-exp-1" {
		t.Errorf("expected experiment ID 'test-exp-1', got %s", exp.ExperimentId)
	}

	if exp.Status != generated.RequestExperimentStatusRunning {
		t.Errorf("expected status 'running', got %s", exp.Status)
	}

	// Cleanup
	manager.StopExperiment("test-exp-1")
}

// TestStartExperiment_Idempotent tests that starting an already running experiment is idempotent
func TestStartExperiment_Idempotent(t *testing.T) {
	manager := setupTestManager(t)

	// Start experiment first time
	request1 := createTestRequest("test-exp-2")
	exp1, err := manager.StartExperiment(request1)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Start same experiment again with different parameters - should return existing experiment
	request2 := createTestRequest("test-exp-2")
	request2.Qps = 100 // Different QPS
	exp2, err := manager.StartExperiment(request2)
	if err != nil {
		t.Fatalf("expected idempotent success, got error: %v", err)
	}

	if exp1.ExperimentId != exp2.ExperimentId {
		t.Errorf("expected same experiment ID, got %s and %s", exp1.ExperimentId, exp2.ExperimentId)
	}

	// Should return the original experiment, not create a new one with new parameters
	if exp2.Qps != exp1.Qps {
		t.Error("expected idempotent call to return original experiment")
	}

	// Cleanup
	manager.StopExperiment("test-exp-2")
}

// TestStartExperiment_CannotRestartCompleted tests that completed experiments cannot be restarted
func TestStartExperiment_CannotRestartCompleted(t *testing.T) {
	manager := setupTestManager(t)

	// Start and immediately stop an experiment
	request := createTestRequest("completed-exp")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Stop immediately (won't have sent many requests but that's ok for test)
	time.Sleep(100 * time.Millisecond)
	_, err = manager.StopExperiment("completed-exp")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	// Try to start experiment with same ID again
	_, err = manager.StartExperiment(request)
	if err == nil {
		t.Error("expected error when trying to restart completed experiment, got nil")
	}

	expectedError := "experiment with ID completed-exp already completed, cannot restart"
	if err != nil && err.Error() != expectedError {
		t.Errorf("unexpected error message: got %v, want %s", err, expectedError)
	}
}

// TestStopExperiment_Success tests successfully stopping an active experiment
func TestStopExperiment_Success(t *testing.T) {
	manager := setupTestManager(t)

	// Start experiment
	request := createTestRequest("test-exp-3")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait a bit to send some requests
	time.Sleep(150 * time.Millisecond)

	// Stop experiment
	result, err := manager.StopExperiment("test-exp-3")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	if result.ExperimentId != "test-exp-3" {
		t.Errorf("expected experiment ID 'test-exp-3', got %s", result.ExperimentId)
	}

	if result.EndTime.IsZero() {
		t.Error("expected EndTime to be set")
	}

	// Verify experiment was saved to storage
	_, err = manager.storage.LoadExperiment("test-exp-3")
	if err != nil {
		t.Errorf("expected experiment to be saved to storage: %v", err)
	}
}

// TestStopExperiment_Idempotent tests that stopping an already stopped experiment is idempotent
func TestStopExperiment_Idempotent(t *testing.T) {
	manager := setupTestManager(t)

	// Start and stop experiment
	request := createTestRequest("test-exp-4")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	_, err = manager.StopExperiment("test-exp-4")
	if err != nil {
		t.Fatalf("failed to stop experiment first time: %v", err)
	}

	// Try to stop again - should succeed idempotently
	result, err := manager.StopExperiment("test-exp-4")
	if err != nil {
		t.Fatalf("expected idempotent success, got error: %v", err)
	}

	if result.ExperimentId != "test-exp-4" {
		t.Errorf("expected experiment ID 'test-exp-4', got %s", result.ExperimentId)
	}

	if result.EndTime.IsZero() {
		t.Error("expected EndTime to be set")
	}
}

// TestStopExperiment_NotFound tests stopping a non-existent experiment
func TestStopExperiment_NotFound(t *testing.T) {
	manager := setupTestManager(t)

	_, err := manager.StopExperiment("non-existent-exp")
	if err == nil {
		t.Error("expected error when stopping non-existent experiment, got nil")
	}

	expectedError := "experiment with ID non-existent-exp not found"
	if err != nil && err.Error() != expectedError {
		t.Errorf("unexpected error message: got %v, want %s", err, expectedError)
	}
}

// TestGetExperiment_FromMemory tests getting a running experiment
func TestGetExperiment_FromMemory(t *testing.T) {
	manager := setupTestManager(t)

	request := createTestRequest("test-exp-5")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait a bit
	time.Sleep(150 * time.Millisecond)

	exp, err := manager.GetExperiment("test-exp-5")
	if err != nil {
		t.Fatalf("failed to get experiment: %v", err)
	}

	if exp.ExperimentId != "test-exp-5" {
		t.Errorf("expected experiment ID 'test-exp-5', got %s", exp.ExperimentId)
	}

	if exp.Status != generated.RequestExperimentStatusRunning {
		t.Errorf("expected status 'running', got %s", exp.Status)
	}

	// Cleanup
	manager.StopExperiment("test-exp-5")
}

// TestGetExperiment_FromStorage tests getting a stopped experiment
func TestGetExperiment_FromStorage(t *testing.T) {
	manager := setupTestManager(t)

	// Start, wait, and stop experiment
	request := createTestRequest("test-exp-6")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	_, err = manager.StopExperiment("test-exp-6")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	// Now get experiment - should come from storage
	exp, err := manager.GetExperiment("test-exp-6")
	if err != nil {
		t.Fatalf("failed to get experiment from storage: %v", err)
	}

	if exp.ExperimentId != "test-exp-6" {
		t.Errorf("expected experiment ID 'test-exp-6', got %s", exp.ExperimentId)
	}

	if exp.Status != generated.RequestExperimentStatusStopped {
		t.Errorf("expected status 'stopped', got %s", exp.Status)
	}
}

// TestGetExperimentStats_FromMemory tests getting stats for a running experiment
func TestGetExperimentStats_FromMemory(t *testing.T) {
	manager := setupTestManager(t)

	request := createTestRequest("test-exp-7")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait to accumulate some stats
	time.Sleep(200 * time.Millisecond)

	// Stop the experiment first to avoid nil pointer in endTime
	_, err = manager.StopExperiment("test-exp-7")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	// Now get stats
	stats, err := manager.GetExperimentStats("test-exp-7")
	if err != nil {
		t.Fatalf("failed to get experiment stats: %v", err)
	}

	if stats.ExperimentId != "test-exp-7" {
		t.Errorf("expected experiment ID 'test-exp-7', got %s", stats.ExperimentId)
	}
}

// TestGetExperimentStats_FromStorage tests getting stats for a stopped experiment
func TestGetExperimentStats_FromStorage(t *testing.T) {
	manager := setupTestManager(t)

	// Start, wait, and stop experiment
	request := createTestRequest("test-exp-8")
	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	_, err = manager.StopExperiment("test-exp-8")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	// Get stats from storage
	stats, err := manager.GetExperimentStats("test-exp-8")
	if err != nil {
		t.Fatalf("failed to get experiment stats from storage: %v", err)
	}

	if stats.ExperimentId != "test-exp-8" {
		t.Errorf("expected experiment ID 'test-exp-8', got %s", stats.ExperimentId)
	}
}

// TestListExperiments tests listing experiments from memory and storage
func TestListExperiments(t *testing.T) {
	manager := setupTestManager(t)

	// Start and stop experiments sequentially (only one at a time per host)
	for i := 0; i < 3; i++ {
		request := createTestRequest(fmt.Sprintf("list-test-exp-%d", i))
		_, err := manager.StartExperiment(request)
		if err != nil {
			t.Fatalf("failed to start experiment %d: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)

		// Stop all except the last one
		if i < 2 {
			_, err = manager.StopExperiment(fmt.Sprintf("list-test-exp-%d", i))
			if err != nil {
				t.Fatalf("failed to stop experiment %d: %v", i, err)
			}
		}
	}

	// List all experiments - should have 1 running + 2 stopped
	experiments := manager.ListExperiments(nil)

	if len(experiments) != 3 {
		t.Errorf("expected 3 experiments, got %d", len(experiments))
	}

	// Verify one is running
	runningCount := 0
	for _, exp := range experiments {
		if exp.Status == generated.RequestExperimentStatusRunning {
			runningCount++
		}
	}

	if runningCount != 1 {
		t.Errorf("expected 1 running experiment, got %d", runningCount)
	}

	// Cleanup
	manager.StopExperiment("list-test-exp-2")
}

// TestSingleExperimentPerHost tests that only one experiment can run at a time per host
func TestSingleExperimentPerHost(t *testing.T) {
	manager := setupTestManager(t)

	// Start first experiment
	request1 := createTestRequest("test-exp-first")
	exp1, err := manager.StartExperiment(request1)
	if err != nil {
		t.Fatalf("failed to start first experiment: %v", err)
	}

	if exp1.ExperimentId != "test-exp-first" {
		t.Errorf("expected experiment ID 'test-exp-first', got %s", exp1.ExperimentId)
	}

	// Try to start second experiment while first is running - should fail
	request2 := createTestRequest("test-exp-second")
	_, err = manager.StartExperiment(request2)
	if err == nil {
		t.Error("expected error when starting second experiment while first is running, got nil")
	}

	expectedErrMsg := "another experiment test-exp-first is already running on this host, please stop it first"
	if err != nil && err.Error() != expectedErrMsg {
		t.Errorf("unexpected error message: got %v, want %s", err, expectedErrMsg)
	}

	// Stop first experiment
	time.Sleep(100 * time.Millisecond)
	_, err = manager.StopExperiment("test-exp-first")
	if err != nil {
		t.Fatalf("failed to stop first experiment: %v", err)
	}

	// Now second experiment should succeed
	exp2, err := manager.StartExperiment(request2)
	if err != nil {
		t.Fatalf("failed to start second experiment after stopping first: %v", err)
	}

	if exp2.ExperimentId != "test-exp-second" {
		t.Errorf("expected experiment ID 'test-exp-second', got %s", exp2.ExperimentId)
	}

	// Cleanup
	time.Sleep(100 * time.Millisecond)
	manager.StopExperiment("test-exp-second")
}

// TestStopAllExperiments tests stopping all running experiments
func TestStopAllExperiments(t *testing.T) {
	manager := setupTestManager(t)

	// Start and stop experiments to create history, then start one final experiment
	for i := 0; i < 3; i++ {
		request := createTestRequest(fmt.Sprintf("stop-all-test-%d", i))
		_, err := manager.StartExperiment(request)
		if err != nil {
			t.Fatalf("failed to start experiment %d: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)

		// Stop all except the last one
		if i < 2 {
			_, err = manager.StopExperiment(fmt.Sprintf("stop-all-test-%d", i))
			if err != nil {
				t.Fatalf("failed to stop experiment %d: %v", i, err)
			}
		}
	}

	// Stop all running experiments (should be only one)
	manager.StopAllExperiments()

	// Verify no experiments are running
	experiments := manager.ListExperiments(nil)
	for _, exp := range experiments {
		if exp.Status == generated.RequestExperimentStatusRunning {
			t.Errorf("expected no running experiments, found %s still running", exp.ExperimentId)
		}
	}
}

// TestExperimentTimeout tests that experiments timeout correctly and save data
func TestExperimentTimeout(t *testing.T) {
	manager := setupTestManager(t)

	// Create experiment with very short timeout (1 second)
	request := generated.StartRequestExperimentRequest{
		ExperimentId: "timeout-test",
		TargetIP:     "localhost",
		TargetPort:   8080,
		Qps:          10,
		Timeout:      1, // 1 second timeout
		Description:  "Timeout test",
	}

	_, err := manager.StartExperiment(request)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait for timeout plus buffer
	time.Sleep(1500 * time.Millisecond)

	// Verify experiment was saved to storage with completed status
	data, err := manager.storage.LoadExperiment("timeout-test")
	if err != nil {
		t.Fatalf("expected experiment to be saved after timeout: %v", err)
	}

	if data.Experiment.Status != generated.RequestExperimentStatusCompleted {
		t.Errorf("expected status 'completed', got %s", data.Experiment.Status)
	}

	if data.Experiment.EndTime.IsZero() {
		t.Error("expected EndTime to be set after timeout")
	}
}