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
                  {/* Hardcoded lines for each potential host - Recharts doesn't support dynamic Line generation via map() or conditional rendering */}
                  {/* Target 1 */}
                  <Line type="monotone" dataKey="target-1_cpuConfUpper" stroke={colors[0]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-1_cpuConfLower" stroke={colors[0]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-1_cpuMean" stroke={colors[0]} strokeWidth={3} dot={{ fill: colors[0], r: 5 }} />
                  <Line type="linear" dataKey="target-1_linearRef" stroke={linearRefColors[0]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
                  {/* Target 2 */}
                  <Line type="monotone" dataKey="target-2_cpuConfUpper" stroke={colors[1]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-2_cpuConfLower" stroke={colors[1]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-2_cpuMean" stroke={colors[1]} strokeWidth={3} dot={{ fill: colors[1], r: 5 }} />
                  <Line type="linear" dataKey="target-2_linearRef" stroke={linearRefColors[1]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
                  {/* Target 3 */}
                  <Line type="monotone" dataKey="target-3_cpuConfUpper" stroke={colors[2]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-3_cpuConfLower" stroke={colors[2]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-3_cpuMean" stroke={colors[2]} strokeWidth={3} dot={{ fill: colors[2], r: 5 }} />
                  <Line type="linear" dataKey="target-3_linearRef" stroke={linearRefColors[2]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
                  {/* Target 4 */}
                  <Line type="monotone" dataKey="target-4_cpuConfUpper" stroke={colors[3]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-4_cpuConfLower" stroke={colors[3]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-4_cpuMean" stroke={colors[3]} strokeWidth={3} dot={{ fill: colors[3], r: 5 }} />
                  <Line type="linear" dataKey="target-4_linearRef" stroke={linearRefColors[3]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
                  {/* Target 5 */}
                  <Line type="monotone" dataKey="target-5_cpuConfUpper" stroke={colors[4]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-5_cpuConfLower" stroke={colors[4]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-5_cpuMean" stroke={colors[4]} strokeWidth={3} dot={{ fill: colors[4], r: 5 }} />
                  <Line type="linear" dataKey="target-5_linearRef" stroke={linearRefColors[4]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
                  {/* Target 6 */}
                  <Line type="monotone" dataKey="target-6_cpuConfUpper" stroke={colors[5 % colors.length]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-6_cpuConfLower" stroke={colors[5 % colors.length]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-6_cpuMean" stroke={colors[5 % colors.length]} strokeWidth={3} dot={{ fill: colors[5 % colors.length], r: 5 }} />
                  <Line type="linear" dataKey="target-6_linearRef" stroke={linearRefColors[5 % linearRefColors.length]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
                  {/* Target 7 */}
                  <Line type="monotone" dataKey="target-7_cpuConfUpper" stroke={colors[6 % colors.length]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-7_cpuConfLower" stroke={colors[6 % colors.length]} strokeWidth={1} strokeDasharray="5 5" dot={false} strokeOpacity={0.5} />
                  <Line type="monotone" dataKey="target-7_cpuMean" stroke={colors[6 % colors.length]} strokeWidth={3} dot={{ fill: colors[6 % colors.length], r: 5 }} />
                  <Line type="linear" dataKey="target-7_linearRef" stroke={linearRefColors[6 % linearRefColors.length]} strokeWidth={2} strokeDasharray="3 3" dot={false} strokeOpacity={0.6} />
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
