// API Client for CPU Simulation Dashboard

import type {
  Host,
  HostHealth,
  CalculationRequest,
  CalculationResponse,
  ExperimentRequest,
  ExperimentResponse,
  ExperimentStatus,
  ExperimentListResponse,
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

  // Host operations
  async getHosts(): Promise<HostsResponse> {
    // The backend returns an array directly, not an object with hosts property
    const hosts = await this.request<Host[]>('/hosts');
    return { hosts };
  }

  async getHostHealth(name: string): Promise<HostHealth> {
    return this.request<HostHealth>(`/hosts/${encodeURIComponent(name)}/health`);
  }

  async testHostCalculation(
    name: string,
    data: CalculationRequest
  ): Promise<CalculationResponse> {
    return this.request<CalculationResponse>(
      `/hosts/${encodeURIComponent(name)}/calculate`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      }
    );
  }

  // Experiment operations
  async getHostExperiments(name: string): Promise<ExperimentListResponse> {
    return this.request<ExperimentListResponse>(
      `/hosts/${encodeURIComponent(name)}/experiments`
    );
  }

  async startHostExperiment(
    name: string,
    data: ExperimentRequest
  ): Promise<ExperimentResponse> {
    return this.request<ExperimentResponse>(
      `/hosts/${encodeURIComponent(name)}/experiments`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      }
    );
  }

  async getHostExperimentStatus(
    name: string,
    experimentId: string
  ): Promise<ExperimentStatus> {
    return this.request<ExperimentStatus>(
      `/hosts/${encodeURIComponent(name)}/experiments/${encodeURIComponent(experimentId)}/status`
    );
  }

  async stopHostExperiment(
    name: string,
    experimentId: string
  ): Promise<ExperimentResponse> {
    return this.request<ExperimentResponse>(
      `/hosts/${encodeURIComponent(name)}/experiments/${encodeURIComponent(experimentId)}/stop`,
      {
        method: 'POST',
      }
    );
  }
}

// Export singleton instance
export const apiClient = new DashboardAPIClient();