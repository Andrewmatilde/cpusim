package exp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"time"
)

type Data interface {
	json.Marshaler
	json.Unmarshaler
}

type CollectFunc[T Data] func(context.Context, gin.Params) (T, error)

type Experiment[T Data] struct {
	ctx context.Context

	logger zerolog.Logger

	CollectData CollectFunc[T]

	fs FileStorage[T]

	cancel context.CancelFunc
	done   chan struct{}
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

func (s *Experiment[T]) Start(id string, timeout time.Duration, params gin.Params) error {
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
	s.done = make(chan struct{})

	go func() {
		defer close(s.done)
		data, err := s.CollectData(ctx, params)
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

	// Wait for the goroutine to finish (data collection and save)
	select {
	case <-s.done:
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("stop timeout: experiment did not finish within 15 seconds")
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
