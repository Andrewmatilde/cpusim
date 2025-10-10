import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import type { ChartConfig } from '@/components/ui/chart';
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from 'recharts';
import { apiClient } from '@/api/client';
import type { Experiment, ExperimentDataResponse, MetricDataPoint } from '@/api/types';
import { RefreshCw, Square, Download, Eye, Clock, TrendingUp, Play, Server, Laptop, Loader2 } from 'lucide-react';
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
  const [operatingExperiment, setOperatingExperiment] = useState<string | null>(null);

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

  const handleStartExperiment = async (experimentId: string) => {
    try {
      setOperatingExperiment(experimentId);
      const response = await apiClient.startCompleteExperiment(experimentId);

      if (response.status === 'success') {
        toast.success(response.message || 'Experiment started successfully');
      } else {
        toast.error(response.message || 'Failed to start experiment');
      }

      fetchExperiments();
    } catch (error) {
      toast.error('Failed to start experiment');
      console.error('Start experiment error:', error);
    } finally {
      setOperatingExperiment(null);
    }
  };

  const handleStopExperiment = async (experimentId: string) => {
    try {
      setOperatingExperiment(experimentId);
      const response = await apiClient.stopCompleteExperiment(experimentId);

      if (response.status === 'success') {
        toast.success(response.message || 'Experiment stopped and data collected successfully');
      } else if (response.status === 'partial') {
        toast.warning(response.message || 'Experiment stopped with partial success');
      } else {
        toast.error(response.message || 'Failed to stop experiment');
      }

      fetchExperiments();
    } catch (error) {
      toast.error('Failed to stop experiment');
      console.error('Stop experiment error:', error);
    } finally {
      setOperatingExperiment(null);
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

  const formatDate = (date: string | Date) => {
    return new Date(date).toLocaleString();
  };

  const getExperimentStatus = (experiment: Experiment) => {
    if (experiment.status) {
      return experiment.status;
    }

    // Check phases if status is not set
    const phases = experiment.phases;
    if (!phases) return 'pending';

    const allPhases = [
      phases.collectorStart,
      phases.requesterStart,
      phases.requesterStop,
      phases.collectorStop
    ];

    const hasInProgress = allPhases.some(p => p?.status === 'running');
    const allCompleted = allPhases.every(p => p?.status === 'completed');
    const hasFailed = allPhases.some(p => p?.status === 'failed');

    if (hasInProgress) return 'running';
    if (allCompleted) return 'completed';
    if (hasFailed) return 'failed';
    return 'pending';
  };

  const prepareCpuChartData = (experimentData: ExperimentDataResponse) => {
    const dataPoints: any[] = [];

    // Collect target host metrics
    if (experimentData.targetHosts) {
      experimentData.targetHosts.forEach(host => {
        if (host.collectorData?.metrics) {
          host.collectorData.metrics.forEach((metric: MetricDataPoint) => {
            const timestamp = new Date(metric.timestamp).getTime();
            const time = new Date(metric.timestamp).toLocaleTimeString();

            let existingPoint = dataPoints.find(p => p.timestamp === timestamp);
            if (!existingPoint) {
              existingPoint = { timestamp, time };
              dataPoints.push(existingPoint);
            }

            existingPoint[`${host.name}_cpu`] = metric.systemMetrics.cpuUsagePercent;
          });
        }
      });
    }

    // Sort by timestamp
    const sortedData = dataPoints.sort((a, b) => a.timestamp - b.timestamp);

    // Log for debugging
    if (sortedData.length > 0) {
      console.log('Chart data sample:', sortedData[0]);
      console.log('Total data points:', sortedData.length);
    }

    return sortedData;
  };

  const getCpuChartConfig = (experimentData: ExperimentDataResponse): ChartConfig => {
    const config: ChartConfig = {};
    const colors = ['#8884d8', '#82ca9d', '#ffc658', '#ff7c7c', '#8dd1e1'];

    if (experimentData.targetHosts) {
      experimentData.targetHosts.forEach((host, index) => {
        config[`${host.name}_cpu`] = {
          label: `${host.name} CPU`,
          color: colors[index % colors.length],
        };
      });
    }

    return config;
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
                  <TableHead>Hosts</TableHead>
                  <TableHead>Request Config</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {experiments.map((experiment) => {
                  const status = getExperimentStatus(experiment);
                  const statusVariant =
                    status === 'completed' ? 'success' :
                    status === 'running' ? 'default' :
                    status === 'failed' ? 'destructive' : 'secondary';

                  return (
                    <TableRow key={experiment.experimentId}>
                      <TableCell className="font-medium">{experiment.experimentId}</TableCell>
                      <TableCell>
                        <div className="space-y-1 text-sm">
                          <div className="flex items-center gap-1">
                            <Server className="h-3 w-3" />
                            {experiment.targetHosts?.length || 0} targets
                          </div>
                          <div className="flex items-center gap-1">
                            <Laptop className="h-3 w-3" />
                            {experiment.clientHost?.name || 'No client'}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        {experiment.requestConfig ? (
                          <div className="text-sm">
                            {experiment.requestConfig.qps} QPS × {experiment.requestConfig.requestTimeout}s timeout
                          </div>
                        ) : '-'}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusVariant}>
                          {status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(experiment.createdAt)}
                      </TableCell>
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
                          {status === 'pending' ? (
                            <Button
                              onClick={() => handleStartExperiment(experiment.experimentId)}
                              variant="outline"
                              size="sm"
                              disabled={operatingExperiment === experiment.experimentId}
                            >
                              {operatingExperiment === experiment.experimentId ? (
                                <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                              ) : (
                                <Play className="h-3 w-3 mr-1" />
                              )}
                              Start
                            </Button>
                          ) : status === 'running' ? (
                            <Button
                              onClick={() => handleStopExperiment(experiment.experimentId)}
                              variant="outline"
                              size="sm"
                              disabled={operatingExperiment === experiment.experimentId}
                            >
                              {operatingExperiment === experiment.experimentId ? (
                                <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                              ) : (
                                <Square className="h-3 w-3 mr-1" />
                              )}
                              Stop
                            </Button>
                          ) : null}
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
        <DialogContent className="sm:max-w-[90vw] md:max-w-5xl lg:max-w-6xl xl:max-w-7xl max-h-[85vh] flex flex-col top-[40%] -translate-y-[40%]">
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
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <strong>Experiment ID:</strong> {experimentData.experimentId}
                  </div>
                  <div>
                    <strong>Target Hosts:</strong> {experimentData.targetHosts?.length || 0}
                  </div>
                </div>

                {/* Requester Data Summary */}
                {experimentData.clientHost?.requesterData && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <Laptop className="h-5 w-5" />
                        Request Performance Summary
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="grid grid-cols-4 gap-4">
                        <div>
                          <div className="text-sm text-muted-foreground">Total Requests</div>
                          <div className="text-2xl font-bold">
                            {experimentData.clientHost.requesterData.stats?.totalRequests || 0}
                          </div>
                        </div>
                        <div>
                          <div className="text-sm text-muted-foreground">Success Rate</div>
                          <div className="text-2xl font-bold">
                            {experimentData.clientHost.requesterData.stats?.totalRequests
                              ? Math.round(((experimentData.clientHost.requesterData.stats.successfulRequests || 0) /
                                experimentData.clientHost.requesterData.stats.totalRequests) * 100)
                              : 0}%
                          </div>
                        </div>
                        <div>
                          <div className="text-sm text-muted-foreground">Avg Response Time</div>
                          <div className="text-2xl font-bold">
                            {experimentData.clientHost.requesterData.stats?.averageResponseTime?.toFixed(2) || 0}ms
                          </div>
                        </div>
                        <div>
                          <div className="text-sm text-muted-foreground">P99 Latency</div>
                          <div className="text-2xl font-bold">
                            {experimentData.clientHost.requesterData.stats?.responseTimeP99?.toFixed(2) || 0}ms
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* CPU Usage Chart */}
                {experimentData.targetHosts && experimentData.targetHosts.length > 0 &&
                 prepareCpuChartData(experimentData).length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <TrendingUp className="h-5 w-5" />
                        CPU Usage Over Time
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="w-full h-[400px]">
                        <ChartContainer config={getCpuChartConfig(experimentData)} className="w-full h-full">
                          <LineChart
                            accessibilityLayer
                            data={prepareCpuChartData(experimentData)}
                            margin={{
                              left: 20,
                              right: 20,
                              top: 20,
                              bottom: 20,
                            }}
                            width={undefined}
                            height={undefined}
                          >
                            <CartesianGrid vertical={false} />
                            <XAxis
                              dataKey="time"
                              tickLine={false}
                              axisLine={false}
                              tickMargin={8}
                              angle={-45}
                              textAnchor="end"
                              height={60}
                              tickFormatter={(value) => value.slice(0, 8)}
                            />
                            <YAxis
                              domain={[0, 100]}
                              tickLine={false}
                              axisLine={false}
                              tickMargin={8}
                              width={60}
                              tickFormatter={(value) => `${value}%`}
                            />
                            <ChartTooltip
                              content={<ChartTooltipContent />}
                            />
                            {experimentData.targetHosts?.map((host, index) => {
                              const colors = ['#8884d8', '#82ca9d', '#ffc658', '#ff7c7c', '#8dd1e1'];
                              return (
                                <Line
                                  key={host.name}
                                  dataKey={`${host.name}_cpu`}
                                  type="monotone"
                                  stroke={colors[index % colors.length]}
                                  strokeWidth={2}
                                  dot={false}
                                />
                              );
                            })}
                          </LineChart>
                        </ChartContainer>
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* Host Data Table */}
                {experimentData.targetHosts && experimentData.targetHosts.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle>Target Host Data Summary</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Host</TableHead>
                            <TableHead>External IP</TableHead>
                            <TableHead>Internal IP</TableHead>
                            <TableHead>Data Available</TableHead>
                            <TableHead>Duration</TableHead>
                            <TableHead>Data Points</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {experimentData.targetHosts.map((host) => (
                            <TableRow key={host.name}>
                              <TableCell className="font-medium">{host.name}</TableCell>
                              <TableCell>{host.externalIP}</TableCell>
                              <TableCell>{host.internalIP || '-'}</TableCell>
                              <TableCell>
                                <Badge variant={host.collectorData ? "default" : "secondary"}>
                                  {host.collectorData ? "Available" : "No Data"}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {host.collectorData?.duration ? `${host.collectorData.duration}s` : '-'}
                              </TableCell>
                              <TableCell>
                                {host.collectorData?.metrics?.length || 0}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </CardContent>
                  </Card>
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