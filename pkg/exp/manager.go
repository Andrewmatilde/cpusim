package exp

import (
	"fmt"
	"github.com/rs/zerolog"
	"time"
)

type Manager[T Data] struct {
	logger zerolog.Logger

	collector CollectFunc[T]

	fs FileStorage[T]

	currentExperiment *Experiment[T]
}

func NewManager[T Data](fs FileStorage[T], collector CollectFunc[T], logger zerolog.Logger) *Manager[T] {
	return &Manager[T]{
		logger:    logger,
		collector: collector,
		fs:        fs,
	}
}

func (f *Manager[T]) Start(id string, timeout time.Duration) error {
	if f.currentExperiment != nil && !f.currentExperiment.IsDone() {
		return fmt.Errorf("experiment already started")
	}

	exp := NewExperiment(f.fs, f.logger)
	exp.SetDataCollector(f.collector)

	err := exp.Start(id, timeout)
	if err != nil {
		return err
	}
	f.currentExperiment = exp
	return nil
}

func (f *Manager[T]) Stop() error {
	if f.currentExperiment == nil {
		return fmt.Errorf("experiment already stopped")
	}
	f.currentExperiment.Stop()
	return nil
}

func (f *Manager[T]) GetExperiment(id string) (T, error) {
	return f.fs.Load(id)
}

const Pending = "Pending"
const Running = "Running"

func (f *Manager[T]) GetStatus() string {
	if f.currentExperiment == nil {
		return Pending
	}
	if f.currentExperiment.IsDone() {
		return Pending
	} else {
		return Running
	}
}
