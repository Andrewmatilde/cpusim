package exp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileStorage[T Data] struct {
	basePath string
}

func NewFileStorage[T Data](basePath string) (*FileStorage[T], error) {
	err := os.MkdirAll(basePath, 0755)
	if err != nil {
		return nil, err
	}

	return &FileStorage[T]{basePath: basePath}, nil
}

func (fs *FileStorage[T]) Save(id string, data T) error {
	f, err := os.Create(filepath.Join(fs.basePath, id+".json"))
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	return encoder.Encode(data)
}

func (fs *FileStorage[T]) Load(id string) (T, error) {
	var zero T
	f, err := os.Open(filepath.Join(fs.basePath, id+".json"))
	if err != nil {
		return zero, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&zero)
	if err != nil {
		return zero, err
	}
	return zero, nil
}

// ExperimentInfo contains metadata about a stored experiment
type ExperimentInfo struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
	FileSizeKB int64     `json:"fileSizeKB"`
}

// List returns a list of all experiments stored in the file system
func (fs *FileStorage[T]) List() ([]ExperimentInfo, error) {
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, err
	}

	var experiments []ExperimentInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only list .json files
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Get file info for timestamps and size
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Extract experiment ID (remove .json extension)
		id := strings.TrimSuffix(entry.Name(), ".json")

		experiments = append(experiments, ExperimentInfo{
			ID:         id,
			CreatedAt:  info.ModTime(), // Use ModTime as creation time
			ModifiedAt: info.ModTime(),
			FileSizeKB: info.Size() / 1024,
		})
	}

	return experiments, nil
}
