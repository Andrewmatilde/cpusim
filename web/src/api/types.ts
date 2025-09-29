// Generated types from OpenAPI specification

export interface Host {
  name?: string;
  ip?: string;
  cpuServiceUrl?: string;
  collectorServiceUrl?: string;
}

export interface HostHealth {
  name?: string;
  ip?: string;
  cpuServiceHealthy?: boolean;
  collectorServiceHealthy?: boolean;
  timestamp?: string;
}

export interface CalculationRequest {
  a?: number;
  b?: number;
}

export interface CalculationResponse {
  gcd?: string;
  process_time?: string;
}

export interface CreateExperimentRequest {
  experimentId: string;
  description?: string;
  timeout?: number;
  collectionInterval?: number;
  participatingHosts: Array<{
    name: string;
    ip: string;
  }>;
}

export interface Experiment {
  experimentId: string;
  description?: string;
  createdAt: string;
  timeout?: number;
  collectionInterval?: number;
  participatingHosts: Array<{
    name: string;
    ip: string;
  }>;
}

export interface ExperimentListResponse {
  experiments: Experiment[];
  total: number;
  hasMore?: boolean;
}

export interface ExperimentDataResponse {
  experimentId: string;
  experiment?: Experiment;
  hosts?: Array<{
    name: string;
    ip: string;
    data?: CollectorExperimentData;
  }>;
}

export interface CollectorExperimentData {
  experimentId?: string;
  description?: string;
  startTime?: string;
  endTime?: string;
  duration?: number;
  collectionInterval?: number;
  metrics?: MetricDataPoint[];
}

export interface MetricDataPoint {
  timestamp: string;
  systemMetrics: {
    cpuUsagePercent: number;
    memoryUsageBytes: number;
    memoryUsagePercent: number;
    calculatorServiceHealthy: boolean;
    networkIOBytes: {
      bytesReceived: number;
      bytesSent: number;
      packetsReceived: number;
      packetsSent: number;
    };
  };
}

export interface StopAndCollectResponse {
  experimentId: string;
  status: 'success' | 'partial' | 'failed';
  timestamp: string;
  message?: string;
  hostsCollected?: Array<{
    name?: string;
    ip?: string;
  }>;
  hostsFailed?: Array<{
    name?: string;
    ip?: string;
    error?: string;
  }>;
}

export interface HostsResponse {
  hosts: Host[];
}