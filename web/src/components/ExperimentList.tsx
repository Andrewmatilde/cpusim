import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { apiClient } from '@/api/client';
import type { Experiment, ExperimentDataResponse, StopAndCollectResponse } from '@/api/types';
import { RefreshCw, Square, Download, Eye, Users, Clock } from 'lucide-react';
import { toast } from 'sonner';

interface ExperimentListProps {
  refreshTrigger: number;
}

export function ExperimentList({ refreshTrigger }: ExperimentListProps) {
  const [experiments, setExperiments] = useState<Experiment[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedExperiment, setSelectedExperiment] = useState<Experiment | null>(null);
  const [experimentData, setExperimentData] = useState<ExperimentDataResponse | null>(null);
  const [dataLoading, setDataLoading] = useState(false);

  const fetchExperiments = async () => {
    try {
      setLoading(true);
      const response = await apiClient.getExperiments(20);
      setExperiments(response.experiments || []);
    } catch (error) {
      toast.error('Failed to fetch experiments');
      console.error('Fetch experiments error:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchExperiments();
  }, [refreshTrigger]);

  const handleStopExperiment = async (experimentId: string) => {
    try {
      const response = await apiClient.stopGlobalExperiment(experimentId);

      if (response.status === 'success') {
        toast.success(response.message || 'Experiment stopped and data collected successfully');
      } else if (response.status === 'partial') {
        toast.warning(response.message || 'Experiment stopped with some failures');
      } else {
        toast.error(response.message || 'Failed to stop experiment');
      }

      // Refresh the experiments list
      fetchExperiments();
    } catch (error) {
      toast.error('Failed to stop experiment');
      console.error('Stop experiment error:', error);
    }
  };

  const handleViewData = async (experiment: Experiment) => {
    try {
      setSelectedExperiment(experiment);
      setDataLoading(true);
      const data = await apiClient.getExperimentData(experiment.experimentId);
      setExperimentData(data);
    } catch (error) {
      toast.error('Failed to load experiment data');
      console.error('Load experiment data error:', error);
    } finally {
      setDataLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getStatusBadge = (experiment: Experiment) => {
    const createdTime = new Date(experiment.createdAt).getTime();
    const now = Date.now();
    const timeoutMs = (experiment.timeout || 300) * 1000;

    if (now - createdTime > timeoutMs) {
      return <Badge variant="secondary">Completed</Badge>;
    } else {
      return <Badge variant="default">Running</Badge>;
    }
  };

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Global Experiments ({experiments.length})
          </CardTitle>
          <Button onClick={fetchExperiments} variant="outline" size="sm" disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
        </CardHeader>
        <CardContent>
          {experiments.length === 0 ? (
            <Alert>
              <AlertDescription>
                No experiments found. Create a new experiment to get started.
              </AlertDescription>
            </Alert>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Experiment ID</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Hosts</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Timeout</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {experiments.map((experiment) => {
                  const isRunning = (() => {
                    const createdTime = new Date(experiment.createdAt).getTime();
                    const now = Date.now();
                    const timeoutMs = (experiment.timeout || 300) * 1000;
                    return now - createdTime <= timeoutMs;
                  })();

                  return (
                    <TableRow key={experiment.experimentId}>
                      <TableCell className="font-medium">{experiment.experimentId}</TableCell>
                      <TableCell>{getStatusBadge(experiment)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Users className="h-3 w-3" />
                          {experiment.participatingHosts.length}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(experiment.createdAt)}
                      </TableCell>
                      <TableCell>{experiment.timeout || 300}s</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            onClick={() => handleViewData(experiment)}
                            variant="outline"
                            size="sm"
                          >
                            <Eye className="h-3 w-3 mr-1" />
                            Data
                          </Button>
                          {isRunning && (
                            <Button
                              onClick={() => handleStopExperiment(experiment.experimentId)}
                              variant="destructive"
                              size="sm"
                            >
                              <Square className="h-3 w-3 mr-1" />
                              Stop
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Experiment Data Dialog */}
      <Dialog open={!!selectedExperiment} onOpenChange={() => setSelectedExperiment(null)}>
        <DialogContent className="sm:max-w-[80vw] md:max-w-4xl lg:max-w-5xl max-h-[75vh] flex flex-col top-[45%] -translate-y-[45%]">
          <DialogHeader className="flex-shrink-0">
            <DialogTitle className="flex items-center gap-2">
              <Download className="h-5 w-5" />
              {selectedExperiment?.experimentId} - Data
            </DialogTitle>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto">
            {dataLoading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="h-6 w-6 animate-spin mr-2" />
                Loading experiment data...
              </div>
            ) : experimentData ? (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <strong>Experiment ID:</strong> {experimentData.experimentId}
                  </div>
                  <div>
                    <strong>Participating Hosts:</strong> {experimentData.hosts?.length || 0}
                  </div>
                </div>

                {experimentData.hosts && experimentData.hosts.length > 0 && (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Host</TableHead>
                        <TableHead>IP</TableHead>
                        <TableHead>Data Available</TableHead>
                        <TableHead>Duration</TableHead>
                        <TableHead>Data Points</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {experimentData.hosts.map((host) => (
                        <TableRow key={host.name}>
                          <TableCell className="font-medium">{host.name}</TableCell>
                          <TableCell>{host.ip}</TableCell>
                          <TableCell>
                            <Badge variant={host.data ? "success" : "secondary"}>
                              {host.data ? "Available" : "No Data"}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            {host.data?.duration ? `${host.data.duration}s` : '-'}
                          </TableCell>
                          <TableCell>
                            {host.data?.metrics?.length || 0}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </div>
            ) : (
              <Alert>
                <AlertDescription>No data available for this experiment</AlertDescription>
              </Alert>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}