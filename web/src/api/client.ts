// API Client for CPU Simulation Dashboard v0.6.0

import type {
  ServiceConfig,
  StatusResponse,
  HealthResponse,
  StartExperimentRequest,
  ExperimentResponse,
  ExperimentData,
  ExperimentListResponse,
  ErrorResponse,
  HostsStatusResponse,
} from './types';

export class DashboardAPIClient {
  private baseUrl: string;

  constructor(baseUrl: string = '') {
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
      let errorMessage: string;
      try {
        const errorData: ErrorResponse = await response.json();
        errorMessage = errorData.message || `HTTP ${response.status}`;
      } catch {
        errorMessage = await response.text() || `HTTP ${response.status}`;
      }
      throw new Error(`API Error: ${errorMessage}`);
    }

    return response.json();
  }

  // Configuration
  async getConfig(): Promise<ServiceConfig> {
    return this.request<ServiceConfig>('/config');
  }

  // Status
  async getStatus(): Promise<StatusResponse> {
    return this.request<StatusResponse>('/status');
  }

  // Health
  async getHealth(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health');
  }

  // Experiment Management
  async listExperiments(): Promise<ExperimentListResponse> {
    return this.request<ExperimentListResponse>('/experiments');
  }

  async startExperiment(data: StartExperimentRequest): Promise<ExperimentResponse> {
    return this.request<ExperimentResponse>('/experiments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async stopExperiment(experimentId: string): Promise<ExperimentResponse> {
    return this.request<ExperimentResponse>(
      `/experiments/${encodeURIComponent(experimentId)}/stop`,
      { method: 'POST' }
    );
  }

  async getExperimentData(experimentId: string): Promise<ExperimentData> {
    return this.request<ExperimentData>(
      `/experiments/${encodeURIComponent(experimentId)}`
    );
  }

  // Hosts Status
  async getHostsStatus(): Promise<HostsStatusResponse> {
    return this.request<HostsStatusResponse>('/hosts/status');
  }

  // Experiment Groups
  async listExperimentGroups(): Promise<import('./generated').ExperimentGroupListResponse> {
    return this.request('/experiment-groups');
  }

  async startExperimentGroup(data: import('./generated').StartExperimentGroupRequest): Promise<import('./generated').ExperimentGroupResponse> {
    return this.request('/experiment-groups', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getExperimentGroupWithDetails(groupId: string): Promise<import('./generated').ExperimentGroupDetail> {
    return this.request(`/experiment-groups/${encodeURIComponent(groupId)}`);
  }
}

// Export singleton instance
export const apiClient = new DashboardAPIClient();
