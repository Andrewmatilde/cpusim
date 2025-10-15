import { useState, useEffect } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiClient } from '@/api/client';
import type {
  ExperimentGroupListResponse,
  ExperimentGroupDetail,
  StartExperimentGroupRequest,
  ExperimentGroup
} from '@/api/generated';
import { RefreshCw, AlertCircle, Play, Loader2, Layers, FileText, BarChart3, Clock } from 'lucide-react';
import { toast } from 'sonner';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Area, ComposedChart } from 'recharts';
import type { ChartConfig } from '@/components/ui/chart';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';

export function ExperimentGroups() {
  const [groupsList, setGroupsList] = useState<ExperimentGroupListResponse | null>(null);
  const [groupDetail, setGroupDetail] = useState<ExperimentGroupDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [groupId, setGroupId] = useState(`group-${Date.now()}`);
  const [description, setDescription] = useState('');
  const [repeatCount, setRepeatCount] = useState(3);
  const [timeout, setTimeout] = useState(60);
  const [qps, setQps] = useState(10);
  const [delayBetween, setDelayBetween] = useState(5);
  const [starting, setStarting] = useState(false);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);
      const groupsData = await apiClient.listExperimentGroups();
      setGroupsList(groupsData);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch experiment groups';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000); // Refresh every 5s
    return () => clearInterval(interval);
  }, []);

  const handleStartGroup = async () => {
    try {
      setStarting(true);
      const request: StartExperimentGroupRequest = {
        groupId,
        description,
        repeatCount,
        timeout,
        qps,
        delayBetween
      };
      await apiClient.startExperimentGroup(request);
      toast.success(`Experiment group started: ${repeatCount} experiments`);
      setGroupId(`group-${Date.now()}`);
      setDescription('');
      fetchData();
    } catch (err) {
      toast.error('Failed to start experiment group');
      console.error('Start group error:', err);
    } finally {
      setStarting(false);
    }
  };

  const handleViewGroup = async (gId: string) => {
    try {
      const detail = await apiClient.getExperimentGroupWithDetails(gId);
      setGroupDetail(detail);
      toast.success('Group details loaded');
    } catch (err) {
      toast.error('Failed to load group details');
      console.error('Load group error:', err);
    }
  };

  const formatDuration = (start: Date | string, end?: Date | string) => {
    if (!end) return 'In progress...';
    const startTime = start instanceof Date ? start.getTime() : new Date(start).getTime();
    const endTime = end instanceof Date ? end.getTime() : new Date(end).getTime();
    const duration = (endTime - startTime) / 1000;
    return `${duration.toFixed(2)}s`;
  };

  return (
    <div className="space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Create Experiment Group */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Play className="h-5 w-5" />
            Create Experiment Group
          </CardTitle>
          <CardDescription>
            Run multiple experiments with the same configuration
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="groupId">Group ID</Label>
                <Input
                  id="groupId"
                  value={groupId}
                  onChange={(e) => setGroupId(e.target.value)}
                  placeholder="group-001"
                />
              </div>
              <div>
                <Label htmlFor="description">Description</Label>
                <Input
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Test description"
                />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-4">
              <div>
                <Label htmlFor="repeatCount">Repeat Count</Label>
                <Input
                  id="repeatCount"
                  type="number"
                  value={repeatCount}
                  onChange={(e) => setRepeatCount(Number(e.target.value))}
                  min={1}
                  max={100}
                />
              </div>
              <div>
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={timeout}
                  onChange={(e) => setTimeout(Number(e.target.value))}
                  min={10}
                  max={600}
                />
              </div>
              <div>
                <Label htmlFor="qps">QPS</Label>
                <Input
                  id="qps"
                  type="number"
                  value={qps}
                  onChange={(e) => setQps(Number(e.target.value))}
                  min={1}
                  max={1000}
                />
              </div>
              <div>
                <Label htmlFor="delayBetween">Delay Between (seconds)</Label>
                <Input
                  id="delayBetween"
                  type="number"
                  value={delayBetween}
                  onChange={(e) => setDelayBetween(Number(e.target.value))}
                  min={0}
                  max={60}
                />
              </div>
            </div>

            <Button
              onClick={handleStartGroup}
              disabled={starting || !groupId}
              className="w-full"
            >
              {starting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting Group...
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4" />
                  Start Experiment Group
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Experiment Groups List */}
      {groupsList && groupsList.groups && groupsList.groups.length > 0 && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Layers className="h-5 w-5" />
                  Experiment Groups ({groupsList.total})
                </CardTitle>
                <CardDescription>
                  All experiment groups (newest first)
                </CardDescription>
              </div>
              <Button onClick={fetchData} variant="outline" size="sm" disabled={loading}>
                <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {groupsList.groups.map((group: ExperimentGroup) => (
                <div
                  key={group.groupId}
                  className="border rounded-lg p-4 hover:bg-accent cursor-pointer transition-colors"
                  onClick={() => group.groupId && handleViewGroup(group.groupId)}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <FileText className="h-4 w-4 text-muted-foreground" />
                        <span className="font-medium">{group.groupId}</span>
                        <Badge variant={group.status === 'completed' ? 'default' : group.status === 'running' ? 'secondary' : 'destructive'}>
                          {group.status}
                        </Badge>
                      </div>
                      {group.description && (
                        <div className="text-sm text-muted-foreground mb-2">
                          {group.description}
                        </div>
                      )}
                      <div className="grid grid-cols-2 gap-2 text-sm">
                        <div className="flex items-center gap-1">
                          <BarChart3 className="h-3 w-3 text-muted-foreground" />
                          <span className="text-muted-foreground">Experiments:</span>
                          <span className="font-medium">{group.currentRun}/{group.config?.repeatCount}</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <Clock className="h-3 w-3 text-muted-foreground" />
                          <span className="text-muted-foreground">Duration:</span>
                          <span className="font-medium">{group.startTime && formatDuration(group.startTime, group.endTime)}</span>
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground mt-2">
                        Started: {group.startTime && (group.startTime instanceof Date ? group.startTime : new Date(group.startTime)).toLocaleString()}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* QPS vs CPU Line Chart with Confidence Intervals */}
      {groupsList && groupsList.groups && groupsList.groups.length > 0 && (() => {
        // Filter completed groups with statistics
        const completedGroups = groupsList.groups.filter(
          (g: ExperimentGroup) => g.status === 'completed' && g.statistics && Object.keys(g.statistics).length > 0
        );

        if (completedGroups.length === 0) return null;

        // Extract and sort data points by QPS
        const chartData = completedGroups
          .map((group: ExperimentGroup) => {
            const hostName = Object.keys(group.statistics || {})[0]; // Get first host
            const stats = group.statistics?.[hostName];

            return {
              qps: group.config?.qps || 0,
              cpuMean: stats?.cpuMean || 0,
              cpuConfLower: stats?.cpuConfLower || 0,
              cpuConfUpper: stats?.cpuConfUpper || 0,
              groupId: group.groupId,
            };
          })
          .sort((a, b) => a.qps - b.qps); // Sort by QPS for proper line connection

        // Add origin point (0,0)
        chartData.unshift({
          qps: 0,
          cpuMean: 0,
          cpuConfLower: 0,
          cpuConfUpper: 0,
          groupId: 'origin',
        });

        // Calculate linear reference line from origin (0,0) to last point
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
                  {/* Upper confidence interval line */}
                  <Line
                    type="monotone"
                    dataKey="cpuConfUpper"
                    stroke="#8884d8"
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    dot={false}
                  />
                  {/* Lower confidence interval line */}
                  <Line
                    type="monotone"
                    dataKey="cpuConfLower"
                    stroke="#8884d8"
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    dot={false}
                  />
                  {/* Mean CPU line */}
                  <Line
                    type="monotone"
                    dataKey="cpuMean"
                    stroke="#8884d8"
                    strokeWidth={3}
                    dot={{ fill: '#8884d8', r: 5 }}
                  />
                  {/* Linear reference line (origin to last point) - orange */}
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

      {/* Group Detail View */}
      {groupDetail && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Layers className="h-5 w-5" />
                Group Details: {groupDetail.group?.groupId}
              </CardTitle>
              <Button onClick={() => setGroupDetail(null)} variant="outline" size="sm">
                Close
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {/* Group Info */}
              <div className="border rounded-lg p-4 space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground">Status:</span>
                  <Badge variant={groupDetail.group?.status === 'completed' ? 'default' : 'secondary'}>
                    {groupDetail.group?.status}
                  </Badge>
                </div>
                {groupDetail.group?.description && (
                  <div>
                    <span className="text-sm text-muted-foreground">Description: </span>
                    <span className="text-sm">{groupDetail.group.description}</span>
                  </div>
                )}
                <div className="grid grid-cols-2 gap-4 pt-2">
                  <div>
                    <div className="text-sm text-muted-foreground">Progress</div>
                    <div className="text-lg font-medium">
                      {groupDetail.group?.currentRun}/{groupDetail.group?.config?.repeatCount} experiments
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground">Duration</div>
                    <div className="text-lg font-medium">
                      {groupDetail.group?.startTime && formatDuration(groupDetail.group.startTime, groupDetail.group.endTime)}
                    </div>
                  </div>
                </div>
                <div className="grid grid-cols-3 gap-4 pt-2 text-sm">
                  <div>
                    <span className="text-muted-foreground">Timeout: </span>
                    <span className="font-medium">{groupDetail.group?.config?.timeout}s</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">QPS: </span>
                    <span className="font-medium">{groupDetail.group?.config?.qps}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Delay: </span>
                    <span className="font-medium">{groupDetail.group?.config?.delayBetween}s</span>
                  </div>
                </div>
              </div>

              {/* Experiments List */}
              <div>
                <h3 className="font-semibold mb-3">Experiments in Group</h3>
                <div className="space-y-2">
                  {groupDetail.group?.experiments?.map((expId, idx) => {
                    const expData = groupDetail.experimentDetails?.[idx];
                    return (
                      <div key={expId} className="border rounded-lg p-3">
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-medium">{expId}</span>
                          {expData && (
                            <Badge variant={expData.status === 'completed' ? 'default' : 'secondary'}>
                              {expData.status}
                            </Badge>
                          )}
                        </div>
                        {expData && (
                          <div className="grid grid-cols-3 gap-2 text-sm">
                            <div>
                              <span className="text-muted-foreground">Duration: </span>
                              <span>{expData.duration?.toFixed(2)}s</span>
                            </div>
                            {expData.requesterResult?.stats && (
                              <>
                                <div>
                                  <span className="text-muted-foreground">Requests: </span>
                                  <span>{expData.requesterResult.stats.totalRequests}</span>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Avg RT: </span>
                                  <span>{expData.requesterResult.stats.averageResponseTime?.toFixed(2)}ms</span>
                                </div>
                              </>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Aggregated CPU Chart */}
              {groupDetail.experimentDetails && groupDetail.experimentDetails.length > 0 && (() => {
                // Process all experiments and aggregate CPU data
                // Collect all data points with relative timestamps
                const experimentSeriesData: { [expId: string]: Array<{ relativeTime: number; cpuUsage: number }> } = {};
                let maxDuration = 0;

                groupDetail.experimentDetails.forEach((expData, idx) => {
                  const expId = groupDetail.group?.experiments?.[idx];
                  if (!expId) return;
                  if (!expData.collectorResults) return;

                  // Get first target host's data
                  const hostResults = Object.values(expData.collectorResults)[0];
                  if (!hostResults?.data?.metrics || hostResults.data.metrics.length === 0) return;

                  const startTime = new Date(hostResults.data.startTime).getTime();
                  const dataPoints = hostResults.data.metrics.map((metric: any) => {
                    const timestamp = new Date(metric.timestamp).getTime();
                    const relativeTime = (timestamp - startTime) / 1000; // Convert to seconds
                    return {
                      relativeTime,
                      cpuUsage: metric.systemMetrics?.cpuUsagePercent || 0
                    };
                  });

                  experimentSeriesData[expId] = dataPoints;
                  const expDuration = Math.max(...dataPoints.map(p => p.relativeTime));
                  if (expDuration > maxDuration) maxDuration = expDuration;
                });

                // Create unified time points (every 0.5 seconds)
                const timePoints: number[] = [];
                for (let t = 0; t <= Math.ceil(maxDuration) + 1; t += 0.5) {
                  timePoints.push(t);
                }

                // Build chart data
                const chartData = timePoints.map(time => {
                  const point: any = { time: time.toFixed(1) };
                  Object.keys(experimentSeriesData).forEach((expId, idx) => {
                    const dataPoints = experimentSeriesData[expId];
                    // Find closest data point
                    let closestPoint = dataPoints[0];
                    let minDiff = Math.abs(dataPoints[0].relativeTime - time);

                    for (const dp of dataPoints) {
                      const diff = Math.abs(dp.relativeTime - time);
                      if (diff < minDiff) {
                        minDiff = diff;
                        closestPoint = dp;
                      }
                    }

                    if (minDiff <= 1.0) { // Only include if within 1 second
                      point[`run${idx + 1}`] = closestPoint.cpuUsage;
                    }
                  });
                  return point;
                });

                // Build chart config dynamically using CSS variables
                const chartConfig: ChartConfig = {};
                Object.keys(experimentSeriesData).forEach((expId, idx) => {
                  const runKey = `run${idx + 1}`;
                  const chartColorVar = `var(--chart-${(idx % 10) + 1})`;
                  chartConfig[runKey] = {
                    label: `Run ${idx + 1}`,
                    color: chartColorVar
                  };
                });

                return (
                  <Card>
                    <CardHeader>
                      <CardTitle>Aggregated CPU Usage Comparison</CardTitle>
                      <CardDescription>
                        Comparing CPU usage across {Object.keys(experimentSeriesData).length} experiment runs
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ChartContainer config={chartConfig}>
                        <LineChart
                          accessibilityLayer
                          data={chartData}
                          margin={{
                            left: 12,
                            right: 12,
                          }}
                        >
                          <CartesianGrid vertical={false} />
                          <XAxis
                            dataKey="time"
                            tickLine={false}
                            axisLine={false}
                            tickMargin={8}
                            tickFormatter={(value) => value}
                          />
                          <ChartTooltip
                            cursor={false}
                            content={<ChartTooltipContent indicator="line" />}
                          />
                          {Object.keys(experimentSeriesData).map((expId, idx) => {
                            const runKey = `run${idx + 1}`;
                            return (
                              <Line
                                key={expId}
                                dataKey={runKey}
                                type="natural"
                                stroke={`var(--color-${runKey})`}
                                strokeWidth={2}
                                dot={false}
                              />
                            );
                          })}
                        </LineChart>
                      </ChartContainer>
                    </CardContent>
                  </Card>
                );
              })()}

              {/* Steady-State Statistics with Confidence Intervals */}
              {groupDetail.group?.statistics && Object.keys(groupDetail.group.statistics).length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle>Steady-State CPU Statistics (95% Confidence Interval)</CardTitle>
                    <CardDescription>
                      Based on the last 90% of each experiment's data, showing mean and confidence intervals across all runs
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      {Object.entries(groupDetail.group.statistics).map(([hostName, stats]) => (
                        <div key={hostName} className="border rounded-lg p-4">
                          <h4 className="font-semibold mb-3 flex items-center gap-2">
                            <BarChart3 className="h-4 w-4" />
                            {hostName}
                          </h4>

                          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                            {/* Mean CPU */}
                            <div className="space-y-1">
                              <div className="text-xs text-muted-foreground">Mean CPU Usage</div>
                              <div className="text-2xl font-bold text-primary">
                                {stats.cpuMean?.toFixed(2)}%
                              </div>
                            </div>

                            {/* Confidence Interval */}
                            <div className="space-y-1">
                              <div className="text-xs text-muted-foreground">95% Confidence Interval</div>
                              <div className="text-lg font-semibold">
                                [{stats.cpuConfLower?.toFixed(2)}%, {stats.cpuConfUpper?.toFixed(2)}%]
                              </div>
                            </div>

                            {/* Standard Deviation */}
                            <div className="space-y-1">
                              <div className="text-xs text-muted-foreground">Std Deviation</div>
                              <div className="text-lg font-semibold">
                                ±{stats.cpuStdDev?.toFixed(2)}%
                              </div>
                            </div>

                            {/* Range */}
                            <div className="space-y-1">
                              <div className="text-xs text-muted-foreground">Range (Min-Max)</div>
                              <div className="text-lg font-semibold">
                                {stats.cpuMin?.toFixed(2)}% - {stats.cpuMax?.toFixed(2)}%
                              </div>
                            </div>
                          </div>

                          {/* Additional Info */}
                          <div className="mt-3 pt-3 border-t text-xs text-muted-foreground">
                            Sample size: {stats.sampleSize} experiments | Confidence level: {((stats.confidenceLevel || 0.95) * 100).toFixed(0)}%
                          </div>

                          {/* Visual representation of CI */}
                          <div className="mt-4">
                            <div className="relative h-8 bg-muted rounded-lg overflow-hidden">
                              {/* Min-Max range background */}
                              <div
                                className="absolute h-full bg-blue-100 dark:bg-blue-950"
                                style={{
                                  left: `${stats.cpuMin}%`,
                                  width: `${(stats.cpuMax || 0) - (stats.cpuMin || 0)}%`
                                }}
                              />

                              {/* Confidence Interval */}
                              <div
                                className="absolute h-full bg-blue-300 dark:bg-blue-700"
                                style={{
                                  left: `${stats.cpuConfLower}%`,
                                  width: `${(stats.cpuConfUpper || 0) - (stats.cpuConfLower || 0)}%`
                                }}
                              />

                              {/* Mean line */}
                              <div
                                className="absolute h-full w-0.5 bg-primary"
                                style={{
                                  left: `${stats.cpuMean}%`
                                }}
                              />
                            </div>
                            <div className="flex justify-between text-xs text-muted-foreground mt-1">
                              <span>0%</span>
                              <span>50%</span>
                              <span>100%</span>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
