// Re-export generated types from OpenAPI specification
export type {
  Host,
  HostConfig,
  HostHealth,
  CalculationRequest,
  CalculationResponse,
  RequestConfig,
  PhaseStatus,
  ExperimentPhases,
  CreateExperimentRequest,
  Experiment,
  ExperimentListResponse,
  CollectorExperimentData,
  ExperimentDataResponse,
  MetricDataPoint,
  SystemMetrics,
  NetworkIO,
  StopAndCollectResponse,
  ExperimentOperationResponse,
  RequestExperimentStats,
  GetHosts200Response,
} from './generated/models';

// Legacy type aliases for backward compatibility
export type { RequestExperimentStats as RequesterData } from './generated/models';
export type { GetHosts200Response as HostsResponse } from './generated/models';