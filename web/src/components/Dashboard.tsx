import { useState } from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useHosts } from '@/hooks/useHosts';
import { ExperimentManager } from './ExperimentManager';
import { CalculationTest } from './CalculationTest';
import type { Host } from '@/api/types';
import { RefreshCw, Server, Activity, AlertCircle, Wifi, WifiOff, PlayCircle, FlaskConical } from 'lucide-react';
import { Toaster } from '@/components/ui/sonner';

export function Dashboard() {
  const { hosts, loading, error, healthStatus, refetch } = useHosts();
  const [selectedHost, setSelectedHost] = useState<Host | null>(null);
  const [dialogTab, setDialogTab] = useState<'experiments' | 'calculation'>('experiments');

  const handleViewDetails = (host: Host) => {
    setSelectedHost(host);
    setDialogTab('experiments');
  };

  const handleRunTest = (host: Host) => {
    setSelectedHost(host);
    setDialogTab('calculation');
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
      <main className="container mx-auto px-4 py-8">
        {loading && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-48" />
            ))}
          </div>
        )}

        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {!loading && !error && hosts.length === 0 && (
          <Alert>
            <AlertDescription>
              No hosts found. Make sure your dashboard backend is running on port 9090.
            </AlertDescription>
          </Alert>
        )}

        {!loading && !error && hosts.length > 0 && (
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
                    const health = healthStatus[host.name];
                    const isHealthy = health?.cpuServiceHealthy && health?.collectorServiceHealthy;

                    return (
                      <TableRow key={host.name}>
                        <TableCell className="font-medium">{host.name}</TableCell>
                        <TableCell>{host.ip}</TableCell>
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
                          {health?.collectorHealth?.timestamp
                            ? new Date(health.collectorHealth.timestamp).toLocaleTimeString()
                            : "-"}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              onClick={() => handleViewDetails(host)}
                              variant="outline"
                              size="sm"
                            >
                              <Activity className="h-4 w-4 mr-1" />
                              Details
                            </Button>
                            <Button
                              onClick={() => handleRunTest(host)}
                              size="sm"
                              disabled={!isHealthy}
                            >
                              <FlaskConical className="h-4 w-4 mr-1" />
                              Test
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}
      </main>

      {/* Host Details Dialog */}
      <Dialog open={!!selectedHost} onOpenChange={() => setSelectedHost(null)}>
        <DialogContent className="sm:max-w-[80vw] md:max-w-3xl lg:max-w-4xl xl:max-w-5xl max-h-[75vh] flex flex-col top-[45%] -translate-y-[45%]">
          <DialogHeader className="flex-shrink-0">
            <DialogTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              {selectedHost?.name}
            </DialogTitle>
          </DialogHeader>

          {selectedHost && (
            <Tabs value={dialogTab} onValueChange={(v) => setDialogTab(v as 'experiments' | 'calculation')} className="flex-1 flex flex-col overflow-hidden">
              <TabsList className="grid w-full grid-cols-2 flex-shrink-0">
                <TabsTrigger value="experiments">Experiments</TabsTrigger>
                <TabsTrigger value="calculation">Calculation Test</TabsTrigger>
              </TabsList>
              <TabsContent value="experiments" className="flex-1 overflow-y-auto mt-4">
                <ExperimentManager host={selectedHost} />
              </TabsContent>
              <TabsContent value="calculation" className="flex-1 overflow-y-auto mt-4">
                <CalculationTest host={selectedHost} />
              </TabsContent>
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}