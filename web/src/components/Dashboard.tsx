import { useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useHosts } from '@/hooks/useHosts';
import { ExperimentForm } from './ExperimentForm';
import { ExperimentList } from './ExperimentList';
import type { Host } from '@/api/types';
import { RefreshCw, Server, AlertCircle, Wifi, WifiOff, FlaskConical } from 'lucide-react';
import { Toaster } from '@/components/ui/sonner';
import { apiClient } from '@/api/client';
import { toast } from 'sonner';

export function Dashboard() {
  const { hosts, loading, error, healthStatus, refetch } = useHosts();
  const [experimentRefreshTrigger, setExperimentRefreshTrigger] = useState(0);

  const handleRunTest = async (host: Host) => {
    try {
      const result = await apiClient.testHostCalculation(host.name || '');
      toast.success(`Test completed! GCD: ${result.gcd}, Time: ${result.processTime}`);
    } catch (error) {
      toast.error('Test failed');
      console.error('Test error:', error);
    }
  };

  const handleExperimentCreated = () => {
    setExperimentRefreshTrigger(prev => prev + 1);
  };

  return (
    <div className="min-h-screen bg-background">
      <Toaster />

      {/* Header */}
      <header className="border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Server className="h-6 w-6" />
              <h1 className="text-2xl font-bold">CPU Simulation Dashboard</h1>
            </div>
            <Button onClick={refetch} variant="outline" size="sm">
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8 space-y-8">
        {loading && (
          <div className="space-y-4">
            <Skeleton className="h-64" />
            <Skeleton className="h-48" />
          </div>
        )}

        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {!loading && !error && (
          <>
            {/* Global Experiment Management */}
            <div className="grid lg:grid-cols-2 gap-8">
              <ExperimentForm
                hosts={hosts || []}
                onExperimentCreated={handleExperimentCreated}
              />
              <div>
                <ExperimentList refreshTrigger={experimentRefreshTrigger} />
              </div>
            </div>

            {/* Host Management */}
            {hosts.length === 0 ? (
              <Alert>
                <AlertDescription>
                  No hosts found. Make sure your dashboard backend is running on port 9090.
                </AlertDescription>
              </Alert>
            ) : (
              <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                  <CardTitle className="flex items-center gap-2">
                    <Server className="h-5 w-5" />
                    Managed Hosts ({hosts.length})
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>IP Address</TableHead>
                        <TableHead>CPU Service</TableHead>
                        <TableHead>Collector Service</TableHead>
                        <TableHead>Overall Status</TableHead>
                        <TableHead>Last Updated</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {hosts.map((host) => {
                        const health = healthStatus[host.name || ''];
                        const isHealthy = health?.cpuServiceHealthy && health?.collectorServiceHealthy;

                        return (
                          <TableRow key={host.name}>
                            <TableCell className="font-medium">{host.name}</TableCell>
                            <TableCell>{host.externalIP}</TableCell>
                            <TableCell>
                              <Badge variant={health?.cpuServiceHealthy ? "success" : "destructive"}>
                                {health?.cpuServiceHealthy ? "Healthy" : "Unhealthy"}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant={health?.collectorServiceHealthy ? "success" : "destructive"}>
                                {health?.collectorServiceHealthy ? "Healthy" : "Unhealthy"}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant={isHealthy ? "default" : health ? "destructive" : "secondary"}>
                                {isHealthy ? (
                                  <>
                                    <Wifi className="h-3 w-3 mr-1" />
                                    Online
                                  </>
                                ) : health ? (
                                  <>
                                    <WifiOff className="h-3 w-3 mr-1" />
                                    Offline
                                  </>
                                ) : (
                                  "Unknown"
                                )}
                              </Badge>
                            </TableCell>
                            <TableCell className="text-sm text-muted-foreground">
                              {health?.timestamp
                                ? new Date(health.timestamp).toLocaleTimeString()
                                : "-"}
                            </TableCell>
                            <TableCell className="text-right">
                              <Button
                                onClick={() => handleRunTest(host)}
                                size="sm"
                                disabled={!health?.cpuServiceHealthy}
                              >
                                <FlaskConical className="h-4 w-4 mr-1" />
                                Test
                              </Button>
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            )}
          </>
        )}
      </main>
    </div>
  );
}