package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"cpusim/pkg/exp"
)

// GroupStorage handles file-based storage for experiment groups
type GroupStorage struct {
	basePath string
}

// NewGroupStorage creates a new group storage instance
func NewGroupStorage(basePath string) (*GroupStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &GroupStorage{
		basePath: basePath,
	}, nil
}

// Save saves an experiment group to disk
func (s *GroupStorage) Save(groupID string, group *ExperimentGroup) error {
	filePath := filepath.Join(s.basePath, groupID+".json")

	data, err := json.MarshalIndent(group, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write group file: %w", err)
	}

	return nil
}

// Load loads an experiment group from disk
func (s *GroupStorage) Load(groupID string) (*ExperimentGroup, error) {
	filePath := filepath.Join(s.basePath, groupID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("group not found: %s", groupID)
		}
		return nil, fmt.Errorf("failed to read group file: %w", err)
	}

	var group ExperimentGroup
	if err := json.Unmarshal(data, &group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group: %w", err)
	}

	return &group, nil
}

// List returns a list of all experiment groups, sorted by start time (newest first)
func (s *GroupStorage) List() ([]exp.ExperimentInfo, error) {
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var groups []exp.ExperimentInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		groupID := file.Name()[:len(file.Name())-5] // Remove .json extension
		group, err := s.Load(groupID)
		if err != nil {
			continue // Skip files that can't be loaded
		}

		groups = append(groups, exp.ExperimentInfo{
			ID:         group.GroupID,
			CreatedAt:  group.StartTime,
			ModifiedAt: group.EndTime,
		})
	}

	// Sort by created time, newest first
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].CreatedAt.After(groups[j].CreatedAt)
	})

	return groups, nil
}

// Delete removes an experiment group from disk
func (s *GroupStorage) Delete(groupID string) error {
	filePath := filepath.Join(s.basePath, groupID+".json")

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("group not found: %s", groupID)
		}
		return fmt.Errorf("failed to delete group file: %w", err)
	}

	return nil
}

// Exists checks if an experiment group exists
func (s *GroupStorage) Exists(groupID string) bool {
	filePath := filepath.Join(s.basePath, groupID+".json")
	_, err := os.Stat(filePath)
	return err == nil
}

// Update updates specific fields of an experiment group
func (s *GroupStorage) Update(groupID string, updateFunc func(*ExperimentGroup) error) error {
	group, err := s.Load(groupID)
	if err != nil {
		return err
	}

	if err := updateFunc(group); err != nil {
		return err
	}

	return s.Save(groupID, group)
}
