package requester

import (
	"context"
	"fmt"
	"time"

	"cpusim/pkg/exp"
	"github.com/rs/zerolog"
)

// Service manages request experiments using the exp framework
type Service struct {
	exp.Manager[*RequestData]

	fs     exp.FileStorage[*RequestData]
	logger zerolog.Logger
	config Config
}

// NewService creates a new requester service
func NewService(storagePath string, config Config, logger zerolog.Logger) (*Service, error) {
	fs, err := exp.NewFileStorage[*RequestData](storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	s := &Service{
		fs:     *fs,
		logger: logger,
		config: config,
	}

	// Create collector function with the service config
	collectFunc := func(ctx context.Context) (*RequestData, error) {
		s.logger.Info().
			Str("target", fmt.Sprintf("%s:%d", s.config.TargetIP, s.config.TargetPort)).
			Int("qps", s.config.QPS).
			Msg("Starting request experiment")

		collector := NewCollector(s.config)
		data, err := collector.Run(ctx)
		if err != nil {
			return nil, err
		}

		s.logger.Info().
			Int64("total_requests", data.TotalRequests).
			Int64("successful", data.Successful).
			Int64("failed", data.Failed).
			Float64("avg_response_time", data.Stats.AvgResponseTime).
			Msg("Request experiment completed")

		return data, nil
	}

	// Create and embed the manager
	s.Manager = *exp.NewManager[*RequestData](*fs, collectFunc, logger)

	return s, nil
}

// StartExperiment starts a new request sending experiment
func (s *Service) StartExperiment(id string, timeout time.Duration) error {
	return s.Manager.Start(id, timeout)
}

// StopExperiment stops the current running experiment
func (s *Service) StopExperiment() error {
	return s.Manager.Stop()
}

// GetExperiment retrieves experiment data by ID
func (s *Service) GetExperiment(id string) (*RequestData, error) {
	return s.fs.Load(id)
}
