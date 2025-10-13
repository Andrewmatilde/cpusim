package collector

import (
	"context"
	"fmt"
	"time"

	"cpusim/pkg/exp"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Service manages metrics collection experiments using the exp framework
type Service struct {
	exp.Manager[*MetricsData]

	fs     exp.FileStorage[*MetricsData]
	logger zerolog.Logger
	config Config
}

// NewService creates a new collector service
func NewService(storagePath string, config Config, logger zerolog.Logger) (*Service, error) {
	fs, err := exp.NewFileStorage[*MetricsData](storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	s := &Service{
		fs:     *fs,
		logger: logger,
		config: config,
	}

	// Create collector function with the service config
	collectFunc := func(ctx context.Context, params gin.Params) (*MetricsData, error) {
		s.logger.Info().
			Int("collection_interval", s.config.CollectionInterval).
			Str("calculator_process", s.config.CalculatorProcess).
			Msg("Starting metrics collection experiment")

		collector := NewCollector(s.config)
		data, err := collector.Run(ctx)
		if err != nil {
			return nil, err
		}

		s.logger.Info().
			Int("data_points", data.DataPointsCollected).
			Float64("duration", data.Duration).
			Msg("Metrics collection experiment completed")

		return data, nil
	}

	// Create and embed the manager
	s.Manager = *exp.NewManager[*MetricsData](*fs, collectFunc, logger)

	return s, nil
}

// StartExperiment starts a new metrics collection experiment
func (s *Service) StartExperiment(id string, timeout time.Duration) error {
	return s.Manager.Start(id, timeout, gin.Params{})
}

// StopExperiment stops the current running experiment
func (s *Service) StopExperiment() error {
	return s.Manager.Stop()
}

// GetExperiment retrieves experiment data by ID
func (s *Service) GetExperiment(id string) (*MetricsData, error) {
	return s.fs.Load(id)
}
