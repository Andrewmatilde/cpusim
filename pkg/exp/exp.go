package exp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"time"
)

type Data interface {
	json.Marshaler
	json.Unmarshaler
}

type CollectFunc[T Data] func(context.Context) (T, error)

type Experiment[T Data] struct {
	ctx context.Context

	logger zerolog.Logger

	CollectData CollectFunc[T]

	fs FileStorage[T]

	cancel context.CancelFunc
}

func NewExperiment[T Data](fs FileStorage[T], logger zerolog.Logger) *Experiment[T] {
	return &Experiment[T]{
		ctx:    context.Background(),
		fs:     fs,
		logger: logger,
	}
}

func (s *Experiment[T]) SetDataCollector(f CollectFunc[T]) {
	s.CollectData = f
}

func (s *Experiment[T]) Start(id string, timeout time.Duration) error {
	if id == "" {
		return fmt.Errorf("id must not be empty")
	}

	if s.CollectData == nil {
		return fmt.Errorf("no collect data found")
	}

	// Create new context with timeout and update s.ctx
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s.ctx = ctx
	s.cancel = cancel

	go func() {
		data, err := s.CollectData(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to collect data")
			return
		}
		err = s.fs.Save(id, data)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to save data")
		}
	}()

	return nil
}

func (s *Experiment[T]) Stop() error {
	if s.cancel == nil {
		return fmt.Errorf("experiment not started")
	}

	s.cancel()

	// Wait for context to be done with a timeout
	select {
	case <-s.ctx.Done():
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("stop timeout: experiment did not finish within 5 seconds")
	}
}

func (s *Experiment[T]) IsDone() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}
