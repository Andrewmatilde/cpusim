import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '@/api/client';
import type { Host, HostHealth } from '@/api/types';

export function useHosts() {
  const [hosts, setHosts] = useState<Host[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [healthStatus, setHealthStatus] = useState<Record<string, HostHealth>>({});

  const fetchHosts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiClient.getHosts();
      setHosts(response.hosts || []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch hosts';
      // Check if it's a connection error
      if (errorMessage.includes('500') || errorMessage.includes('fetch')) {
        setError('Cannot connect to the backend API. Please ensure the dashboard backend is running on http://localhost:9090');
      } else {
        setError(errorMessage);
      }
      setHosts([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchHostHealth = useCallback(async (hostName: string) => {
    try {
      const health = await apiClient.getHostHealth(hostName);
      setHealthStatus(prev => ({ ...prev, [hostName]: health }));
      return health;
    } catch (err) {
      console.error(`Failed to fetch health for ${hostName}:`, err);
      return null;
    }
  }, []);

  const refreshAllHealth = useCallback(async () => {
    const healthPromises = hosts.map(host => fetchHostHealth(host.name));
    await Promise.all(healthPromises);
  }, [hosts, fetchHostHealth]);

  useEffect(() => {
    fetchHosts();
  }, [fetchHosts]);

  useEffect(() => {
    if (hosts.length > 0) {
      refreshAllHealth();
      const interval = setInterval(refreshAllHealth, 10000); // Refresh every 10s
      return () => clearInterval(interval);
    }
  }, [hosts, refreshAllHealth]);

  return {
    hosts,
    loading,
    error,
    healthStatus,
    refetch: fetchHosts,
    refreshHealth: refreshAllHealth,
  };
}