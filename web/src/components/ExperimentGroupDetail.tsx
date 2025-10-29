import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Layers, ArrowLeft } from 'lucide-react';
import { Line, XAxis, YAxis, CartesianGrid, ComposedChart } from 'recharts';
import type { ChartConfig } from '@/components/ui/chart';
import { ChartContainer, ChartTooltip } from '@/components/ui/chart';
import { apiClient } from '@/api/client';
import type { ExperimentGroup } from '@/api/generated';
import { toast } from 'sonner';

export function ExperimentGroupDetail() {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const [groupData, setGroupData] = useState<ExperimentGroup | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!groupId) {
      navigate('/groups');
      return;
    }

    const loadGroupData = async () => {
      try {
        setLoading(true);
        const data = await apiClient.getExperimentGroupWithDetails(groupId);
        setGroupData(data.group || null);
      } catch (error) {
        toast.error('Failed to load experiment group');
        console.error('Load group error:', error);
        navigate('/groups');
      } finally {
        setLoading(false);
      }
    };

    loadGroupData();
  }, [groupId, navigate]);

  // Separate effect for auto-refresh
  useEffect(() => {
    if (!groupData || groupData.status !== 'running') {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const data = await apiClient.getExperimentGroupWithDetails(groupId!);
        setGroupData(data.group || null);
      } catch (error) {
        console.error('Auto-refresh error:', error);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [groupData?.status, groupId]);

  const formatDuration = (start: Date | string, end?: Date | string) => {
    if (!end) return 'In progress...';
    const endDate = end instanceof Date ? end : new Date(end);
    if (endDate.getFullYear() < 1900) return 'In progress...';
    const startTime = start instanceof Date ? start.getTime() : new Date(start).getTime();
    const endTime = endDate.getTime();
    const duration = (endTime - startTime) / 1000;
    return `${duration.toFixed(2)}s`;
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">Loading experiment group data...</div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!groupData) {
    return null;
  }

  return (
    <div className="space-y-6">
      {/* Header with Back Button */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Layers className="h-5 w-5" />
              Group: {groupId}
            </CardTitle>
            <Button onClick={() => navigate('/groups')} variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Groups
            </Button>
          </div>
        </CardHeader>
      </Card>

      {/* Group Info */}
      <Card>
        <CardHeader>
          <CardTitle>Group Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="border rounded-lg p-4 space-y-2">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">Status:</span>
              <Badge variant={groupData.status === 'completed' ? 'default' : 'secondary'}>
                {groupData.status}
              </Badge>
            </div>
            {groupData.description && (
              <div>
                <span className="text-sm text-muted-foreground">Description: </span>
                <span className="text-sm">{groupData.description}</span>
              </div>
            )}
            {/* Environment Configuration */}
            {groupData.environmentConfig && (
              <div className="pt-2 border-t">
                <div className="text-sm font-medium mb-2">Environment:</div>
                <div className="grid grid-cols-3 gap-2 text-xs">
                  <div>
                    <span className="text-muted-foreground">Client: </span>
                    <span className="font-medium">{groupData.environmentConfig.clientHost?.name}</span>
                    <span className="text-muted-foreground ml-1">({groupData.environmentConfig.clientHost?.externalIP})</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Target: </span>
                    {groupData.environmentConfig.targetHosts && groupData.environmentConfig.targetHosts.map((target, idx) => (
                      <span key={idx}>
                        <span className="font-medium">{target.name}</span>
                        <span className="text-muted-foreground ml-1">({target.externalIP})</span>
                      </span>
                    ))}
                  </div>
                  {groupData.environmentConfig.loadBalancer && (
                    <div>
                      <span className="text-muted-foreground">LoadBalancer: </span>
                      <span className="font-medium">{groupData.environmentConfig.loadBalancer.name}</span>
                      <span className="text-muted-foreground ml-1">({groupData.environmentConfig.loadBalancer.internalIP})</span>
                    </div>
                  )}
                </div>
              </div>
            )}
            <div className="grid grid-cols-2 gap-4 pt-2">
              <div>
                <div className="text-sm text-muted-foreground">QPS Range</div>
                <div className="text-lg font-medium">
                  {groupData.config?.qpsMin} - {groupData.config?.qpsMax} (step {groupData.config?.qpsStep})
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Duration</div>
                <div className="text-lg font-medium">
                  {groupData.startTime && formatDuration(groupData.startTime, groupData.endTime)}
                </div>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4 pt-2 text-sm">
              <div>
                <span className="text-muted-foreground">Timeout: </span>
                <span className="font-medium">{groupData.config?.timeout}s</span>
              </div>
              <div>
                <span className="text-muted-foreground">Repeat per QPS: </span>
                <span className="font-medium">{groupData.config?.repeatCount}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Delay: </span>
                <span className="font-medium">{groupData.config?.delayBetween}s</span>
              </div>
            </div>
            <div className="text-sm pt-2">
              <span className="text-muted-foreground">Progress: </span>
              <span className="font-medium">QPS {groupData.currentQPS}, Run {groupData.currentRun}/{groupData.config?.repeatCount}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* QPS vs CPU Chart */}
      {groupData.qpsPoints && groupData.qpsPoints.length > 0 && (() => {
        // Collect all unique host names
        const allHostNames = new Set<string>();
        groupData.qpsPoints.forEach((qpsPoint: any) => {
          if (qpsPoint.statistics) {
            Object.keys(qpsPoint.statistics).forEach(hostName => allHostNames.add(hostName));
          }
        });

        const hostNamesArray = Array.from(allHostNames);
        if (hostNamesArray.length === 0) return null;

        // Build data points with all hosts' data
        const chartData: any[] = [];
        groupData.qpsPoints.forEach((qpsPoint: any) => {
          const point: any = { qps: qpsPoint.qps || 0 };

          if (qpsPoint.statistics) {
            Object.entries(qpsPoint.statistics).forEach(([hostName, stats]: [string, any]) => {
              point[`${hostName}_cpuMean`] = stats?.cpuMean || 0;
              point[`${hostName}_cpuConfLower`] = stats?.cpuConfLower || 0;
              point[`${hostName}_cpuConfUpper`] = stats?.cpuConfUpper || 0;
            });
          }

          chartData.push(point);
        });

        // Sort by QPS
        chartData.sort((a, b) => a.qps - b.qps);

        // Add origin point
        const originPoint: any = { qps: 0 };
        hostNamesArray.forEach(hostName => {
          originPoint[`${hostName}_cpuMean`] = 0;
          originPoint[`${hostName}_cpuConfLower`] = 0;
          originPoint[`${hostName}_cpuConfUpper`] = 0;
        });
        chartData.unshift(originPoint);

        // Calculate linear reference for each host
        if (chartData.length >= 2) {
          const lastPoint = chartData[chartData.length - 1];
          hostNamesArray.forEach(hostName => {
            const cpuMean = lastPoint[`${hostName}_cpuMean`];
            if (cpuMean && lastPoint.qps) {
              const slope = cpuMean / lastPoint.qps;
              chartData.forEach(point => {
                point[`${hostName}_linearRef`] = slope * point.qps;
              });
            }
          });
        }

        // Define colors for different hosts
        const colors = [
          '#8884d8',  // blue
          '#82ca9d',  // green
          '#ffc658',  // yellow
          '#ff7c7c',  // red
          '#a28dff',  // purple
        ];

        const linearRefColors = [
          '#f97316',  // orange
          '#10b981',  // emerald
          '#f59e0b',  // amber
          '#ef4444',  // red
          '#8b5cf6',  // violet
        ];

        // Build chart config dynamically
        const chartConfig: ChartConfig = {};
        hostNamesArray.forEach((hostName, index) => {
          const color = colors[index % colors.length];
          chartConfig[`${hostName}_cpuMean`] = {
            label: `${hostName} Mean CPU`,
            color: color,
          };
        });

        return (
          <Card>
            <CardHeader>
              <CardTitle>QPS vs CPU Usage Analysis</CardTitle>
              <CardDescription>
                Average CPU usage across different QPS levels with 95% confidence interval boundaries
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ChartContainer config={chartConfig} className="min-h-[400px]">
                <ComposedChart
                  data={chartData}
                  margin={{
                    top: 20,
                    right: 20,
                    bottom: 40,
                    left: 20,
                  }}
                >
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis
                    dataKey="qps"
                    type="number"
                    domain={['dataMin', 'dataMax']}
                    label={{ value: 'QPS (Requests per Second)', position: 'insideBottom', offset: -10 }}
                  />
                  <YAxis
                    label={{ value: 'CPU Usage (%)', angle: -90, position: 'insideLeft' }}
                  />
                  <ChartTooltip
                    content={({ active, payload }) => {
                      if (active && payload && payload.length) {
                        const data = payload[0].payload;
                        return (
                          <div className="bg-background border rounded-lg p-3 shadow-lg">
                            <div className="font-semibold text-sm mb-2">QPS: {data.qps}</div>
                            <div className="space-y-2 text-sm">
                              {hostNamesArray.map(hostName => {
                                const mean = data[`${hostName}_cpuMean`];
                                const lower = data[`${hostName}_cpuConfLower`];
                                const upper = data[`${hostName}_cpuConfUpper`];
                                if (mean !== undefined) {
                                  return (
                                    <div key={hostName} className="border-t pt-1 first:border-t-0 first:pt-0">
                                      <div className="font-medium">{hostName}</div>
                                      <div>
                                        <span className="text-muted-foreground">Mean CPU:</span>{' '}
                                        <span className="font-medium">{mean.toFixed(2)}%</span>
                                      </div>
                                      {lower !== undefined && upper !== undefined && (
                                        <div>
                                          <span className="text-muted-foreground">95% CI:</span>{' '}
                                          <span className="font-medium">
                                            [{lower.toFixed(2)}%, {upper.toFixed(2)}%]
                                          </span>
                                        </div>
                                      )}
                                    </div>
                                  );
                                }
                                return null;
                              })}
                            </div>
                          </div>
                        );
                      }
                      return null;
                    }}
                  />
                  {/* Dynamically render lines for each host */}
                  {hostNamesArray.map((hostName, index) => {
                    const meanColor = colors[index % colors.length];
                    const linearColor = linearRefColors[index % linearRefColors.length];
                    return [
                      <Line
                        key={`${hostName}_cpuConfUpper`}
                        type="monotone"
                        dataKey={`${hostName}_cpuConfUpper`}
                        stroke={meanColor}
                        strokeWidth={1}
                        strokeDasharray="5 5"
                        dot={false}
                        strokeOpacity={0.5}
                      />,
                      <Line
                        key={`${hostName}_cpuConfLower`}
                        type="monotone"
                        dataKey={`${hostName}_cpuConfLower`}
                        stroke={meanColor}
                        strokeWidth={1}
                        strokeDasharray="5 5"
                        dot={false}
                        strokeOpacity={0.5}
                      />,
                      <Line
                        key={`${hostName}_cpuMean`}
                        type="monotone"
                        dataKey={`${hostName}_cpuMean`}
                        stroke={meanColor}
                        strokeWidth={3}
                        dot={{ fill: meanColor, r: 5 }}
                      />,
                      <Line
                        key={`${hostName}_linearRef`}
                        type="linear"
                        dataKey={`${hostName}_linearRef`}
                        stroke={linearColor}
                        strokeWidth={2}
                        strokeDasharray="3 3"
                        dot={false}
                        strokeOpacity={0.6}
                      />
                    ];
                  })}
                </ComposedChart>
              </ChartContainer>
              <div className="mt-4 text-sm text-muted-foreground space-y-1">
                <div className="font-medium mb-2">Legend:</div>
                {hostNamesArray.map((hostName, index) => {
                  const meanColor = colors[index % colors.length];
                  const linearColor = linearRefColors[index % linearRefColors.length];
                  return (
                    <div key={hostName} className="flex items-center gap-2">
                      <div style={{ backgroundColor: meanColor }} className="w-3 h-3 rounded-full"></div>
                      <span className="font-medium">{hostName}:</span>
                      <span>thick line = mean CPU, dashed lines = 95% CI</span>
                      <div style={{ backgroundColor: linearColor }} className="w-3 h-3 rounded-full ml-2"></div>
                      <span>linear reference</span>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        );
      })()}

      {/* QPS vs Latency Chart (P50, P90, P95, P99, Mean) */}
      {groupData.qpsPoints && groupData.qpsPoints.length > 0 && (() => {
        // Build latency data points
        const latencyData: any[] = [];
        groupData.qpsPoints.forEach((qpsPoint: any) => {
          if (qpsPoint.latencyStats && qpsPoint.latencyStats.latencyP50 !== undefined) {
            latencyData.push({
              qps: qpsPoint.qps || 0,
              p50: qpsPoint.latencyStats.latencyP50 || 0,
              p90: qpsPoint.latencyStats.latencyP90 || 0,
              p95: qpsPoint.latencyStats.latencyP95 || 0,
              p99: qpsPoint.latencyStats.latencyP99 || 0,
              mean: qpsPoint.latencyStats.latencyMean || 0,
              min: qpsPoint.latencyStats.latencyMin || 0,
              max: qpsPoint.latencyStats.latencyMax || 0,
            });
          }
        });

        // Sort by QPS
        latencyData.sort((a, b) => a.qps - b.qps);

        if (latencyData.length === 0) return null;

        // Add origin point
        latencyData.unshift({ qps: 0, p50: 0, p90: 0, p95: 0, p99: 0, mean: 0, min: 0, max: 0 });

        const latencyChartConfig = {
          min: { label: "Min", color: "#d1d5db" },
          p50: { label: "P50", color: "#06b6d4" },
          mean: { label: "Mean", color: "#10b981" },
          p90: { label: "P90", color: "#f59e0b" },
          p95: { label: "P95", color: "#f97316" },
          p99: { label: "P99", color: "#ef4444" },
          max: { label: "Max", color: "#9ca3af" },
        } satisfies ChartConfig;

        return (
          <Card>
            <CardHeader>
              <CardTitle>QPS vs Latency</CardTitle>
              <CardDescription>
                Response time percentiles (P50, P90, P95, P99), mean, and range across different load levels
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ChartContainer config={latencyChartConfig} className="h-[400px] w-full">
                <ComposedChart data={latencyData} margin={{ top: 20, right: 30, left: 20, bottom: 20 }}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis
                    dataKey="qps"
                    label={{ value: 'QPS (Requests/sec)', position: 'insideBottom', offset: -10 }}
                  />
                  <YAxis
                    label={{ value: 'Latency (ms)', angle: -90, position: 'insideLeft' }}
                  />
                  <ChartTooltip />
                  {/* Min/Max as reference lines */}
                  <Line
                    type="monotone"
                    dataKey="min"
                    stroke={latencyChartConfig.min.color}
                    strokeWidth={1}
                    strokeDasharray="3 3"
                    dot={false}
                    strokeOpacity={0.5}
                    name="Min"
                  />
                  <Line
                    type="monotone"
                    dataKey="max"
                    stroke={latencyChartConfig.max.color}
                    strokeWidth={1}
                    strokeDasharray="3 3"
                    dot={false}
                    strokeOpacity={0.5}
                    name="Max"
                  />
                  {/* Main percentile lines */}
                  <Line
                    type="monotone"
                    dataKey="p50"
                    stroke={latencyChartConfig.p50.color}
                    strokeWidth={2}
                    dot={{ r: 4 }}
                    name="P50 (Median)"
                  />
                  <Line
                    type="monotone"
                    dataKey="mean"
                    stroke={latencyChartConfig.mean.color}
                    strokeWidth={2}
                    dot={{ r: 4 }}
                    name="Mean"
                  />
                  <Line
                    type="monotone"
                    dataKey="p90"
                    stroke={latencyChartConfig.p90.color}
                    strokeWidth={2}
                    dot={{ r: 4 }}
                    name="P90"
                  />
                  <Line
                    type="monotone"
                    dataKey="p95"
                    stroke={latencyChartConfig.p95.color}
                    strokeWidth={2.5}
                    dot={{ r: 4 }}
                    name="P95"
                  />
                  <Line
                    type="monotone"
                    dataKey="p99"
                    stroke={latencyChartConfig.p99.color}
                    strokeWidth={3}
                    dot={{ r: 5 }}
                    name="P99"
                  />
                </ComposedChart>
              </ChartContainer>
              <div className="mt-4 text-sm text-muted-foreground space-y-2">
                <div className="font-medium mb-2">Latency Metrics:</div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="flex items-center gap-2">
                    <div style={{ backgroundColor: latencyChartConfig.p50.color }} className="w-3 h-3 rounded-full"></div>
                    <span><strong>P50 (Median):</strong> 50% of requests faster</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div style={{ backgroundColor: latencyChartConfig.mean.color }} className="w-3 h-3 rounded-full"></div>
                    <span><strong>Mean:</strong> Average response time</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div style={{ backgroundColor: latencyChartConfig.p90.color }} className="w-3 h-3 rounded-full"></div>
                    <span><strong>P90:</strong> 90% of requests faster</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div style={{ backgroundColor: latencyChartConfig.p95.color }} className="w-3 h-3 rounded-full"></div>
                    <span><strong>P95:</strong> 95% of requests faster</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div style={{ backgroundColor: latencyChartConfig.p99.color }} className="w-3 h-3 rounded-full"></div>
                    <span><strong>P99:</strong> 99% of requests faster</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div style={{ backgroundColor: latencyChartConfig.min.color }} className="w-3 h-3 rounded-full opacity-50"></div>
                    <span className="text-xs"><strong>Min/Max:</strong> Response time range (dashed)</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        );
      })()}

      {/* QPS Points with Experiments */}
      <Card>
        <CardHeader>
          <CardTitle>QPS Points and Experiments</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {groupData.qpsPoints?.map((qpsPoint: any, qpsIdx) => (
              <div key={qpsIdx} className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-3">
                  <h4 className="font-semibold">QPS: {qpsPoint.qps}</h4>
                  <Badge variant={qpsPoint.status === 'completed' ? 'default' : qpsPoint.status === 'running' ? 'secondary' : 'outline'}>
                    {qpsPoint.status}
                  </Badge>
                </div>

                {/* Statistics for this QPS */}
                {qpsPoint.statistics && Object.keys(qpsPoint.statistics).length > 0 && (
                  <div className="mb-3 p-2 bg-muted rounded">
                    {Object.entries(qpsPoint.statistics).map(([hostName, stats]: [string, any]) => (
                      <div key={hostName} className="text-sm">
                        <span className="text-muted-foreground">{hostName}: </span>
                        <span className="font-medium">Mean CPU: {stats.cpuMean?.toFixed(2)}%</span>
                        <span className="text-muted-foreground ml-2">95% CI: [{stats.cpuConfLower?.toFixed(2)}%, {stats.cpuConfUpper?.toFixed(2)}%]</span>
                      </div>
                    ))}
                  </div>
                )}

                {/* Experiments list for this QPS */}
                <div className="space-y-1">
                  <div className="text-xs text-muted-foreground">Experiments ({qpsPoint.experiments?.length || 0}):</div>
                  <div className="flex flex-wrap gap-1">
                    {qpsPoint.experiments?.map((expId: string) => (
                      <Badge
                        key={expId}
                        variant="outline"
                        className="text-xs cursor-pointer hover:bg-accent"
                        onClick={() => navigate(`/experiment/${expId}`)}
                      >
                        {expId}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
