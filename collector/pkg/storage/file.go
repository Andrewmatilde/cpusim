package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExperimentData represents the data structure for storing experiment results
type ExperimentData struct {
	ExperimentID       string            `json:"experimentId"`
	Description        string            `json:"description,omitempty"`
	StartTime          time.Time         `json:"startTime"`
	EndTime            *time.Time        `json:"endTime,omitempty"`
	Duration           int               `json:"duration,omitempty"` // Duration in seconds
	CollectionInterval int               `json:"collectionInterval"` // Interval in milliseconds
	Metrics            []MetricDataPoint `json:"metrics"`
}

// MetricDataPoint represents a single metrics collection point
type MetricDataPoint struct {
	Timestamp     time.Time     `json:"timestamp"`
	SystemMetrics SystemMetrics `json:"systemMetrics"`
}

// SystemMetrics represents the system metrics at a point in time
type SystemMetrics struct {
	CPUUsagePercent          float64   `json:"cpuUsagePercent"`
	MemoryUsageBytes         int64     `json:"memoryUsageBytes"`
	MemoryUsagePercent       float64   `json:"memoryUsagePercent"`
	NetworkIOBytes           NetworkIO `json:"networkIOBytes"`
	CalculatorServiceHealthy bool      `json:"calculatorServiceHealthy"`
}

// NetworkIO represents network I/O statistics
type NetworkIO struct {
	BytesReceived   int64 `json:"bytesReceived"`
	BytesSent       int64 `json:"bytesSent"`
	PacketsReceived int64 `json:"packetsReceived"`
	PacketsSent     int64 `json:"packetsSent"`
}

// FileStorage handles file-based storage of experiment data
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file storage instance
func NewFileStorage(basePath string) (*FileStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStorage{
		basePath: basePath,
	}, nil
}

// SaveExperimentData saves experiment data to a JSON file
func (fs *FileStorage) SaveExperimentData(experimentID string, data *ExperimentData) error {
	filename := fmt.Sprintf("%s.json", experimentID)
	filepath := filepath.Join(fs.basePath, filename)

	// Convert to JSON with pretty formatting
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal experiment data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write experiment data to file: %w", err)
	}

	return nil
}

// LoadExperimentData loads experiment data from a JSON file
func (fs *FileStorage) LoadExperimentData(experimentID string) (*ExperimentData, error) {
	filename := fmt.Sprintf("%s.json", experimentID)
	filepath := filepath.Join(fs.basePath, filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("experiment data file not found: %s", experimentID)
	}

	// Read file
	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read experiment data file: %w", err)
	}

	// Parse JSON
	var data ExperimentData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse experiment data: %w", err)
	}

	return &data, nil
}

// ExperimentExists checks if an experiment data file exists
func (fs *FileStorage) ExperimentExists(experimentID string) bool {
	filename := fmt.Sprintf("%s.json", experimentID)
	filepath := filepath.Join(fs.basePath, filename)

	_, err := os.Stat(filepath)
	return err == nil
}

// ListExperiments returns a list of all experiment IDs with their basic info
func (fs *FileStorage) ListExperiments() ([]ExperimentInfo, error) {
	files, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var experiments []ExperimentInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Extract experiment ID from filename
		experimentID := file.Name()[:len(file.Name())-5] // Remove .json extension

		// Get file info
		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		experiments = append(experiments, ExperimentInfo{
			ExperimentID: experimentID,
			CreatedAt:    fileInfo.ModTime(),
			Size:         fileInfo.Size(),
		})
	}

	return experiments, nil
}

// DeleteExperimentData deletes an experiment data file
func (fs *FileStorage) DeleteExperimentData(experimentID string) error {
	filename := fmt.Sprintf("%s.json", experimentID)
	filepath := filepath.Join(fs.basePath, filename)

	if err := os.Remove(filepath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("experiment data file not found: %s", experimentID)
		}
		return fmt.Errorf("failed to delete experiment data file: %w", err)
	}

	return nil
}

// ExperimentInfo represents basic information about a stored experiment
type ExperimentInfo struct {
	ExperimentID string    `json:"experimentId"`
	CreatedAt    time.Time `json:"createdAt"`
	Size         int64     `json:"size"`
}

// GetStoragePath returns the full path to the storage directory
func (fs *FileStorage) GetStoragePath() string {
	return fs.basePath
}

// GetExperimentFilePath returns the full path to an experiment's data file
func (fs *FileStorage) GetExperimentFilePath(experimentID string) string {
	filename := fmt.Sprintf("%s.json", experimentID)
	return filepath.Join(fs.basePath, filename)
}