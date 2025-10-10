package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cpusim/requester/api/generated"
)

// FileStorage handles persisting experiment data to filesystem
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

// ExperimentData represents the complete experiment data for storage
type ExperimentData struct {
	Experiment *generated.RequestExperiment      `json:"experiment"`
	Stats      *generated.RequestExperimentStats `json:"stats"`
	SavedAt    time.Time                         `json:"savedAt"`
}

// SaveExperiment saves experiment and its stats to filesystem
func (fs *FileStorage) SaveExperiment(experiment *generated.RequestExperiment, stats *generated.RequestExperimentStats) error {
	if experiment.ExperimentId == "" {
		return fmt.Errorf("experiment ID cannot be empty")
	}

	data := &ExperimentData{
		Experiment: experiment,
		Stats:      stats,
		SavedAt:    time.Now(),
	}

	filename := fmt.Sprintf("%s.json", experiment.ExperimentId)
	filepath := filepath.Join(fs.basePath, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode experiment data: %w", err)
	}

	return nil
}

// LoadExperiment loads experiment data from filesystem
func (fs *FileStorage) LoadExperiment(experimentId string) (*ExperimentData, error) {
	filename := fmt.Sprintf("%s.json", experimentId)
	filepath := filepath.Join(fs.basePath, filename)

	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("experiment not found")
		}
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	var data ExperimentData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode experiment data: %w", err)
	}

	return &data, nil
}

// ListExperiments returns a list of all stored experiments
func (fs *FileStorage) ListExperiments() ([]*generated.RequestExperiment, error) {
	files, err := filepath.Glob(filepath.Join(fs.basePath, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list experiment files: %w", err)
	}

	var experiments []*generated.RequestExperiment
	for _, file := range files {
		data, err := fs.loadExperimentFile(file)
		if err != nil {
			// Log error but continue processing other files
			fmt.Printf("Warning: failed to load experiment file %s: %v\n", file, err)
			continue
		}
		experiments = append(experiments, data.Experiment)
	}

	return experiments, nil
}

// ExperimentExists checks if an experiment exists in storage
func (fs *FileStorage) ExperimentExists(experimentId string) bool {
	filename := fmt.Sprintf("%s.json", experimentId)
	filepath := filepath.Join(fs.basePath, filename)

	_, err := os.Stat(filepath)
	return err == nil
}

// DeleteExperiment removes an experiment from storage
func (fs *FileStorage) DeleteExperiment(experimentId string) error {
	filename := fmt.Sprintf("%s.json", experimentId)
	filepath := filepath.Join(fs.basePath, filename)

	err := os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete experiment file: %w", err)
	}

	return nil
}

// GetStoragePath returns the base storage path
func (fs *FileStorage) GetStoragePath() string {
	return fs.basePath
}

// loadExperimentFile loads experiment data from a specific file
func (fs *FileStorage) loadExperimentFile(filepath string) (*ExperimentData, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data ExperimentData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// CleanupOldExperiments removes experiments older than the specified duration
func (fs *FileStorage) CleanupOldExperiments(olderThan time.Duration) error {
	files, err := filepath.Glob(filepath.Join(fs.basePath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list experiment files: %w", err)
	}

	cutoff := time.Now().Add(-olderThan)
	deletedCount := 0

	for _, file := range files {
		data, err := fs.loadExperimentFile(file)
		if err != nil {
			continue
		}

		if data.SavedAt.Before(cutoff) {
			if err := os.Remove(file); err == nil {
				deletedCount++
			}
		}
	}

	fmt.Printf("Cleaned up %d old experiment files\n", deletedCount)
	return nil
}