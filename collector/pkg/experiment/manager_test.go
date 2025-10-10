package experiment

import (
	"testing"
	"time"

	"cpusim/collector/api/generated"
	"cpusim/collector/pkg/metrics"
	"cpusim/collector/pkg/storage"
)

// setupTestManager creates a manager with a temporary storage directory
func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()

	// Create temporary directory for storage
	tempDir := t.TempDir()

	// Create storage
	fileStorage, err := storage.NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	// Create mock metrics collector
	metricsCollector := metrics.NewCollector("cpusim-server")

	// Create manager
	manager := NewManager(metricsCollector, fileStorage)

	return manager, tempDir
}

// TestStartExperiment_NewExperiment tests starting a new experiment
func TestStartExperiment_NewExperiment(t *testing.T) {
	manager, _ := setupTestManager(t)

	exp, err := manager.StartExperiment("test-exp-1", "Test experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	if exp.ID != "test-exp-1" {
		t.Errorf("expected experiment ID 'test-exp-1', got %s", exp.ID)
	}

	if exp.Status != generated.ExperimentStatusStatusRunning {
		t.Errorf("expected status 'running', got %s", exp.Status)
	}

	if !exp.IsActive {
		t.Error("expected experiment to be active")
	}

	// Cleanup
	manager.StopExperiment("test-exp-1")
}

// TestStartExperiment_Idempotent tests that starting an already running experiment is idempotent
func TestStartExperiment_Idempotent(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Start experiment first time
	exp1, err := manager.StartExperiment("test-exp-2", "Test experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Start same experiment again - should return existing experiment
	exp2, err := manager.StartExperiment("test-exp-2", "Different description", 200*time.Millisecond, 10*time.Second)
	if err != nil {
		t.Fatalf("expected idempotent success, got error: %v", err)
	}

	if exp1.ID != exp2.ID {
		t.Errorf("expected same experiment ID, got %s and %s", exp1.ID, exp2.ID)
	}

	// Should return the original experiment, not create a new one with new parameters
	if exp2.CollectionInterval != exp1.CollectionInterval {
		t.Error("expected idempotent call to return original experiment")
	}

	// Cleanup
	manager.StopExperiment("test-exp-2")
}

// TestStartExperiment_CannotRestartCompleted tests that completed experiments cannot be restarted
func TestStartExperiment_CannotRestartCompleted(t *testing.T) {
	manager, tempDir := setupTestManager(t)

	// Create a fake completed experiment in storage
	completedData := &storage.ExperimentData{
		ExperimentID:       "completed-exp",
		Description:        "Completed experiment",
		StartTime:          time.Now().Add(-1 * time.Hour),
		CollectionInterval: 1000,
		Metrics:            []storage.MetricDataPoint{},
	}
	endTime := time.Now()
	completedData.EndTime = &endTime
	completedData.Duration = 3600

	// Save to storage
	fileStorage, _ := storage.NewFileStorage(tempDir)
	if err := fileStorage.SaveExperimentData("completed-exp", completedData); err != nil {
		t.Fatalf("failed to save completed experiment: %v", err)
	}

	// Try to start experiment with same ID
	_, err := manager.StartExperiment("completed-exp", "New attempt", 100*time.Millisecond, 5*time.Second)
	if err == nil {
		t.Error("expected error when trying to restart completed experiment, got nil")
	}

	if err != nil && err.Error() != "experiment with ID completed-exp already completed, cannot restart" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestStopExperiment_Success tests successfully stopping an active experiment
func TestStopExperiment_Success(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Start experiment
	_, err := manager.StartExperiment("test-exp-3", "Test experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait a bit to collect some data
	time.Sleep(250 * time.Millisecond)

	// Stop experiment
	stoppedExp, err := manager.StopExperiment("test-exp-3")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	if stoppedExp.Status != generated.ExperimentStatusStatusStopped {
		t.Errorf("expected status 'stopped', got %s", stoppedExp.Status)
	}

	if stoppedExp.IsActive {
		t.Error("expected experiment to be inactive")
	}

	if stoppedExp.EndTime == nil {
		t.Error("expected EndTime to be set")
	}

	// Verify experiment was saved to storage
	_, err = manager.GetExperimentData("test-exp-3")
	if err != nil {
		t.Errorf("expected experiment data to be saved to storage: %v", err)
	}
}

// TestStopExperiment_Idempotent tests that stopping an already stopped experiment is idempotent
func TestStopExperiment_Idempotent(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Start and stop experiment
	_, err := manager.StartExperiment("test-exp-4", "Test experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	_, err = manager.StopExperiment("test-exp-4")
	if err != nil {
		t.Fatalf("failed to stop experiment first time: %v", err)
	}

	// Try to stop again - should succeed idempotently
	stoppedExp, err := manager.StopExperiment("test-exp-4")
	if err != nil {
		t.Fatalf("expected idempotent success, got error: %v", err)
	}

	if stoppedExp.Status != generated.ExperimentStatusStatusStopped {
		t.Errorf("expected status 'stopped', got %s", stoppedExp.Status)
	}

	if stoppedExp.IsActive {
		t.Error("expected experiment to be inactive")
	}
}

// TestStopExperiment_NotFound tests stopping a non-existent experiment
func TestStopExperiment_NotFound(t *testing.T) {
	manager, _ := setupTestManager(t)

	_, err := manager.StopExperiment("non-existent-exp")
	if err == nil {
		t.Error("expected error when stopping non-existent experiment, got nil")
	}

	if err != nil && err.Error() != "experiment with ID non-existent-exp not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestGetExperimentData_FromMemory tests getting data for a running experiment
func TestGetExperimentData_FromMemory(t *testing.T) {
	manager, _ := setupTestManager(t)

	_, err := manager.StartExperiment("test-exp-5", "Test experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait to collect some data
	time.Sleep(250 * time.Millisecond)

	data, err := manager.GetExperimentData("test-exp-5")
	if err != nil {
		t.Fatalf("failed to get experiment data: %v", err)
	}

	if data.ExperimentID != "test-exp-5" {
		t.Errorf("expected experiment ID 'test-exp-5', got %s", data.ExperimentID)
	}

	// Note: In test environments, metrics collection may fail, so we don't strictly check data points
	// Just verify the structure is correct
	if data.Metrics == nil {
		t.Error("expected Metrics array to be initialized")
	}

	// Cleanup
	manager.StopExperiment("test-exp-5")
}

// TestGetExperimentData_FromStorage tests getting data for a stopped experiment
func TestGetExperimentData_FromStorage(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Start, wait, and stop experiment
	_, err := manager.StartExperiment("test-exp-6", "Test experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	time.Sleep(250 * time.Millisecond)

	_, err = manager.StopExperiment("test-exp-6")
	if err != nil {
		t.Fatalf("failed to stop experiment: %v", err)
	}

	// Now get data - should come from storage
	data, err := manager.GetExperimentData("test-exp-6")
	if err != nil {
		t.Fatalf("failed to get experiment data from storage: %v", err)
	}

	if data.ExperimentID != "test-exp-6" {
		t.Errorf("expected experiment ID 'test-exp-6', got %s", data.ExperimentID)
	}

	if data.EndTime == nil {
		t.Error("expected EndTime to be set for stopped experiment")
	}
}

// TestExperimentTimeout tests that experiments timeout correctly
func TestExperimentTimeout(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Start experiment with very short timeout
	_, err := manager.StartExperiment("test-exp-timeout", "Timeout test", 50*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to start experiment: %v", err)
	}

	// Wait for timeout plus buffer
	time.Sleep(350 * time.Millisecond)

	// Check experiment status
	exp, err := manager.GetExperiment("test-exp-timeout")
	if err != nil {
		t.Fatalf("failed to get experiment: %v", err)
	}

	if exp.Status != generated.ExperimentStatusStatusTimeout {
		t.Errorf("expected status 'timeout', got %s", exp.Status)
	}

	if exp.IsActive {
		t.Error("expected experiment to be inactive after timeout")
	}
}

// TestSingleExperimentPerHost tests that only one experiment can run at a time per host
func TestSingleExperimentPerHost(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Start first experiment
	exp1, err := manager.StartExperiment("test-exp-first", "First experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start first experiment: %v", err)
	}

	if exp1.ID != "test-exp-first" {
		t.Errorf("expected experiment ID 'test-exp-first', got %s", exp1.ID)
	}

	// Try to start second experiment while first is running - should fail
	_, err = manager.StartExperiment("test-exp-second", "Second experiment", 100*time.Millisecond, 5*time.Second)
	if err == nil {
		t.Error("expected error when starting second experiment while first is running, got nil")
	}

	expectedErrMsg := "another experiment test-exp-first is already running on this host, please stop it first"
	if err != nil && err.Error() != expectedErrMsg {
		t.Errorf("unexpected error message: got %v, want %s", err, expectedErrMsg)
	}

	// Stop first experiment
	_, err = manager.StopExperiment("test-exp-first")
	if err != nil {
		t.Fatalf("failed to stop first experiment: %v", err)
	}

	// Now second experiment should succeed
	exp2, err := manager.StartExperiment("test-exp-second", "Second experiment", 100*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to start second experiment after stopping first: %v", err)
	}

	if exp2.ID != "test-exp-second" {
		t.Errorf("expected experiment ID 'test-exp-second', got %s", exp2.ID)
	}

	// Cleanup
	manager.StopExperiment("test-exp-second")
}