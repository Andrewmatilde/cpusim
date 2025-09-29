// Generated types from OpenAPI specification

export interface Host {
  name: string;
  ip: string;
  cpuServiceUrl: string;
  collectorServiceUrl: string;
}

export interface HostHealth {
  name: string;
  ip: string;
  cpuServiceHealthy: boolean;
  collectorServiceHealthy: boolean;
  collectorHealth?: {
    status: string;
    timestamp: string;
  };
}

export interface CalculationRequest {
  a: number;
  b: number;
}

export interface CalculationResponse {
  gcd: string;
  process_time: string;
}

export interface ExperimentRequest {
  experimentId: string;
  description: string;
  timeout: number;
  collectionInterval: number;
}

export interface ExperimentResponse {
  experimentId: string;
  message: string;
  status: string;
  timestamp: string;
}

export interface NetworkIO {
  bytesReceived: number;
  bytesSent: number;
  packetsReceived: number;
  packetsSent: number;
}

export interface ExperimentStatus {
  experimentId: string;
  status: string;
  isActive: boolean;
  dataPointsCollected: number;
  startTime?: string;
  endTime?: string;
  duration?: number;
  lastMetrics?: {
    cpuUsagePercent: number;
    memoryUsageBytes: number;
    memoryUsagePercent: number;
    calculatorServiceHealthy: boolean;
    networkIOBytes?: NetworkIO;
  };
}

export interface ExperimentListResponse {
  experiments: ExperimentStatus[];
  total: number;
  hasMore: boolean;
}

export interface HostsResponse {
  hosts: Host[];
}