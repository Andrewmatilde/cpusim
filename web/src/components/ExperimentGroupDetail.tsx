import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Layers, ArrowLeft } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, ComposedChart } from 'recharts';
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
        const dataPoints: Array<{qps: number; cpuMean: number; cpuConfLower: number; cpuConfUpper: number; groupId: string; linearRef?: number}> = [];

        groupData.qpsPoints.forEach((qpsPoint: any) => {
          if (qpsPoint.statistics && Object.keys(qpsPoint.statistics).length > 0) {
            const hostName = Object.keys(qpsPoint.statistics)[0];
            const stats = qpsPoint.statistics[hostName];

            dataPoints.push({
              qps: qpsPoint.qps || 0,
              cpuMean: stats?.cpuMean || 0,
              cpuConfLower: stats?.cpuConfLower || 0,
              cpuConfUpper: stats?.cpuConfUpper || 0,
              groupId: `qps-${qpsPoint.qps}`,
            });
          }
        });

        if (dataPoints.length === 0) return null;

        const chartData = dataPoints.sort((a, b) => a.qps - b.qps);

        chartData.unshift({
          qps: 0,
          cpuMean: 0,
          cpuConfLower: 0,
          cpuConfUpper: 0,
          groupId: 'origin',
        });

        if (chartData.length >= 2) {
          const lastPoint = chartData[chartData.length - 1];
          const slope = lastPoint.cpuMean / lastPoint.qps;

          chartData.forEach(point => {
            point.linearRef = slope * point.qps;
          });
        }

        const chartConfig: ChartConfig = {
          cpuMean: {
            label: "Mean CPU Usage",
            color: "hsl(var(--chart-1))",
          },
          cpuConfLower: {
            label: "95% CI Lower",
            color: "hsl(var(--chart-1))",
          },
          cpuConfUpper: {
            label: "95% CI Upper",
            color: "hsl(var(--chart-1))",
          },
        };

        return (
          <Card>
            <CardHeader>
              <CardTitle>QPS vs CPU Usage Analysis</CardTitle>
              <CardDescription>
                Average CPU usage across different QPS levels with 95% confidence interval boundaries
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ChartContainer config={chartConfig}>
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
                            <div className="font-semibold text-sm mb-2">{data.groupId}</div>
                            <div className="space-y-1 text-sm">
                              <div>
                                <span className="text-muted-foreground">QPS:</span>{' '}
                                <span className="font-medium">{data.qps}</span>
                              </div>
                              <div>
                                <span className="text-muted-foreground">Mean CPU:</span>{' '}
                                <span className="font-medium">{data.cpuMean.toFixed(2)}%</span>
                              </div>
                              <div>
                                <span className="text-muted-foreground">95% CI:</span>{' '}
                                <span className="font-medium">
                                  [{data.cpuConfLower.toFixed(2)}%, {data.cpuConfUpper.toFixed(2)}%]
                                </span>
                              </div>
                            </div>
                          </div>
                        );
                      }
                      return null;
                    }}
                  />
                  <Line
                    type="monotone"
                    dataKey="cpuConfUpper"
                    stroke="#8884d8"
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    dot={false}
                  />
                  <Line
                    type="monotone"
                    dataKey="cpuConfLower"
                    stroke="#8884d8"
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    dot={false}
                  />
                  <Line
                    type="monotone"
                    dataKey="cpuMean"
                    stroke="#8884d8"
                    strokeWidth={3}
                    dot={{ fill: '#8884d8', r: 5 }}
                  />
                  <Line
                    type="linear"
                    dataKey="linearRef"
                    stroke="#f97316"
                    strokeWidth={2}
                    strokeDasharray="3 3"
                    dot={false}
                  />
                </ComposedChart>
              </ChartContainer>
              <div className="mt-4 text-sm text-muted-foreground">
                <div>Solid thick blue line: mean CPU usage</div>
                <div>Blue dashed lines: 95% confidence interval boundaries</div>
                <div>Orange dashed line: linear reference (origin to last point)</div>
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
