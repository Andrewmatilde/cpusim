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
import type { Experiment, ExperimentDataResponse, StopAndCollectResponse } from '@/api/types';
import { RefreshCw, Square, Download, Eye, Users, Clock, TrendingUp } from 'lucide-react';
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

  const prepareCpuChartData = (experimentData: ExperimentDataResponse) => {
    if (!experimentData.hosts || experimentData.hosts.length === 0) return [];

    // 收集所有主机的 CPU 数据
    const allDataPoints = experimentData.hosts.flatMap(host => {
      if (!host.data?.metrics) return [];

      return host.data.metrics.map(metric => ({
        timestamp: new Date(metric.timestamp).getTime(),
        time: new Date(metric.timestamp).toLocaleTimeString(),
        [`${host.name}_cpu`]: metric.systemMetrics.cpuUsagePercent,
        hostName: host.name
      }));
    });

    // 按时间戳分组
    const groupedByTime = allDataPoints.reduce((acc, point) => {
      const timeKey = point.timestamp;
      if (!acc[timeKey]) {
        acc[timeKey] = { timestamp: timeKey, time: point.time };
      }
      acc[timeKey][`${point.hostName}_cpu`] = point[`${point.hostName}_cpu`];
      return acc;
    }, {} as Record<number, any>);

    // 转换为数组并排序
    return Object.values(groupedByTime).sort((a: any, b: any) => a.timestamp - b.timestamp);
  };

  const getCpuChartConfig = (experimentData: ExperimentDataResponse): ChartConfig => {
    if (!experimentData.hosts) return {};

    const config: ChartConfig = {};
    experimentData.hosts.forEach((host, index) => {
      config[`${host.name}_cpu`] = {
        label: `${host.name} CPU`,
        color: `hsl(var(--chart-${(index % 5) + 1}))`,
      };
    });

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
                  <TableHead>Created</TableHead>
                  <TableHead>Timeout</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {experiments.map((experiment) => {
                  return (
                    <TableRow key={experiment.experimentId}>
                      <TableCell className="font-medium">{experiment.experimentId}</TableCell>
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
                          <Button
                            onClick={() => handleStopExperiment(experiment.experimentId)}
                            variant="outline"
                            size="sm"
                          >
                            <Square className="h-3 w-3 mr-1" />
                            Collect
                          </Button>
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
                    <strong>Participating Hosts:</strong> {experimentData.hosts?.length || 0}
                  </div>
                </div>

                {/* CPU Usage Chart */}
                {experimentData.hosts && experimentData.hosts.length > 0 && prepareCpuChartData(experimentData).length > 0 && (
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
                            {experimentData.hosts?.map((host) => (
                              <Line
                                key={host.name}
                                dataKey={`${host.name}_cpu`}
                                type="monotone"
                                stroke={`var(--color-${host.name}_cpu)`}
                                strokeWidth={2}
                                dot={false}
                              />
                            ))}
                          </LineChart>
                        </ChartContainer>
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* Host Data Table */}
                {experimentData.hosts && experimentData.hosts.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle>Host Data Summary</CardTitle>
                    </CardHeader>
                    <CardContent>
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
                                <Badge variant={host.data ? "default" : "secondary"}>
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