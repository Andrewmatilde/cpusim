// Re-export generated types from Dashboard OpenAPI specification (v0.7.0)
export type {
  // Dashboard configuration
  ServiceConfig,
  TargetHost,
  ClientHost,

  // Experiment management
  StartExperimentRequest,
  ExperimentResponse,
  ExperimentData,
  ExperimentListResponse,
  ExperimentInfo,
  CollectorResult,
  RequesterResult,
  ExperimentError,

  // Status and health
  StatusResponse,
  HealthResponse,
  ErrorResponse,
  HostsStatusResponse,
  TargetHostStatus,
  ClientHostStatus,
} from './generated/models';

// Type aliases for convenience
export type DashboardStatus = 'Pending' | 'Running';
export type ExperimentStatus = 'running' | 'completed' | 'failed' | 'partial';
