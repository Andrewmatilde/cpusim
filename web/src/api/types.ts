// Re-export generated types from Dashboard OpenAPI specification (v0.6.0)
export type {
  // Dashboard configuration
  ServiceConfig,
  TargetHost,
  ClientHost,

  // Experiment management
  StartExperimentRequest,
  ExperimentResponse,
  ExperimentData,
  CollectorResult,
  RequesterResult,
  ExperimentError,

  // Status and health
  StatusResponse,
  HealthResponse,
  ErrorResponse,
} from './generated/models';

// Type aliases for convenience
export type DashboardStatus = 'Pending' | 'Running';
export type ExperimentStatus = 'running' | 'completed' | 'failed' | 'partial';
