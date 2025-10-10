// API Client for CPU Simulation Dashboard

import type {
  HostHealth,
  CalculationRequest,
  CalculationResponse,
  CreateExperimentRequest,
  Experiment,
  ExperimentListResponse,
  ExperimentDataResponse,
  StopAndCollectResponse,
  ExperimentOperationResponse,
  ExperimentPhases,
  HostsResponse
} from './types';

export class DashboardAPIClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`API Error: ${response.status} - ${error}`);
    }

    return response.json();
  }

  // Global Experiment operations
  async getExperiments(limit?: number): Promise<ExperimentListResponse> {
    const params = new URLSearchParams();
    if (limit) {
      params.append('limit', limit.toString());
    }
    const query = params.toString() ? `?${params.toString()}` : '';
    return this.request<ExperimentListResponse>(`/experiments${query}`);
  }

  async createGlobalExperiment(data: CreateExperimentRequest): Promise<Experiment> {
    return this.request<Experiment>('/experiments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getGlobalExperiment(experimentId: string): Promise<Experiment> {
    return this.request<Experiment>(`/experiments/${encodeURIComponent(experimentId)}`);
  }

  async getExperimentData(experimentId: string, hostName?: string): Promise<ExperimentDataResponse> {
    const params = new URLSearchParams();
    if (hostName) {
      params.append('hostName', hostName);
    }
    const query = params.toString() ? `?${params.toString()}` : '';
    return this.request<ExperimentDataResponse>(`/experiments/${encodeURIComponent(experimentId)}/data${query}`);
  }

  // New experiment phase management
  async getExperimentPhases(experimentId: string): Promise<ExperimentPhases> {
    return this.request<ExperimentPhases>(`/experiments/${encodeURIComponent(experimentId)}/phases`);
  }

  async startCompleteExperiment(experimentId: string): Promise<ExperimentOperationResponse> {
    return this.request<ExperimentOperationResponse>(
      `/experiments/${encodeURIComponent(experimentId)}/start`,
      { method: 'POST' }
    );
  }

  async stopCompleteExperiment(experimentId: string): Promise<ExperimentOperationResponse> {
    return this.request<ExperimentOperationResponse>(
      `/experiments/${encodeURIComponent(experimentId)}/stop`,
      { method: 'POST' }
    );
  }

  // Legacy stop method (kept for backward compatibility)
  async stopGlobalExperiment(experimentId: string): Promise<StopAndCollectResponse> {
    return this.request<StopAndCollectResponse>(
      `/experiments/${encodeURIComponent(experimentId)}/stop`,
      { method: 'POST' }
    );
  }

  // Host operations
  async getHosts(): Promise<HostsResponse> {
    return this.request<HostsResponse>('/hosts');
  }

  async getHostHealth(name: string): Promise<HostHealth> {
    return this.request<HostHealth>(`/hosts/${encodeURIComponent(name)}/health`);
  }

  async testHostCalculation(
    name: string,
    data: CalculationRequest = { a: 12345678, b: 87654321 }
  ): Promise<CalculationResponse> {
    return this.request<CalculationResponse>(
      `/hosts/${encodeURIComponent(name)}/calculate`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      }
    );
  }
}

// Export singleton instance
export const apiClient = new DashboardAPIClient();