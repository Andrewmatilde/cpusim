package exp

import (
	"encoding/json"
	"os"
	"path/filepath"
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
