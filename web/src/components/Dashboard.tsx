import { useState, useEffect } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiClient } from '@/api/client';
import type {
  ServiceConfig,
  StatusResponse,
  ExperimentData,
  ExperimentListResponse,
  StartExperimentRequest,
  HostsStatusResponse
} from '@/api/types';
import { RefreshCw, Server, AlertCircle, Play, Square, Download, Loader2, Activity, Laptop, History, FileText, LineChart as LineChartIcon, BarChart3 } from 'lucide-react';
import { toast } from 'sonner';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

export function Dashboard() {
  const [config, setConfig] = useState<ServiceConfig | null>(null);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [hostsStatus, setHostsStatus] = useState<HostsStatusResponse | null>(null);
  const [experimentsList, setExperimentsList] = useState<ExperimentListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [experimentId, setExperimentId] = useState(`exp-${Date.now()}`);
  const [timeout, setTimeout] = useState(60);
  const [qps, setQps] = useState(10);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [experimentData, setExperimentData] = useState<ExperimentData | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);
      const [configData, statusData, hostsStatusData, experimentsData] = await Promise.all([
        apiClient.getConfig(),
        apiClient.getStatus(),
        apiClient.getHostsStatus(),
        apiClient.listExperiments()
      ]);
      setConfig(configData);
      setStatus(statusData);
      setHostsStatus(hostsStatusData);
      setExperimentsList(experimentsData);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch data';
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

  const handleStartExperiment = async () => {
    try {
      setStarting(true);
      const request: StartExperimentRequest = {
        experimentId,
        timeout,
        qps
      };
      await apiClient.startExperiment(request);
      toast.success('Experiment started successfully');
      setExperimentId(`exp-${Date.now()}`); // Generate new ID for next experiment
      fetchData();
    } catch (error) {
      toast.error('Failed to start experiment');
      console.error('Start experiment error:', error);
    } finally {
      setStarting(false);
    }
  };

  const handleStopExperiment = async () => {
    if (!experimentId) return;

    try {
      setStopping(true);
      await apiClient.stopExperiment(experimentId);
      toast.success('Experiment stopped successfully');
      fetchData();
    } catch (error) {
      toast.error('Failed to stop experiment');
      console.error('Stop experiment error:', error);
    } finally {
      setStopping(false);
    }
  };

  const handleViewData = async (expId?: string) => {
    const idToLoad = expId || experimentId;
    if (!idToLoad) return;

    try {
      const data = await apiClient.getExperimentData(idToLoad);
      setExperimentData(data);
      toast.success('Experiment data loaded');
    } catch (error) {
      toast.error('Failed to load experiment data');
      console.error('Load data error:', error);
    }
  };

  const isRunning = status?.status === 'Running';

  return (
    <div className="space-y-6">
      {/* Refresh Button */}
      <div className="flex justify-end">
        <Button onClick={fetchData} variant="outline" size="sm" disabled={loading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>
        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Service Status Banner */}
        {status && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Activity className="h-5 w-5" />
                    Dashboard Status
                  </CardTitle>
                  {isRunning && experimentId && (
                    <CardDescription className="mt-1">
                      Current Experiment: {experimentId}
                    </CardDescription>
                  )}
                </div>
                <Badge variant={isRunning ? "default" : "secondary"} className="text-lg px-4 py-1">
                  {status.status}
                </Badge>
              </div>
            </CardHeader>
          </Card>
        )}

        {/* Configuration Display */}
        {config && (
          <div className="grid lg:grid-cols-2 gap-6">
            {/* Target Hosts */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Server className="h-5 w-5" />
                  Target Hosts ({config.targetHosts?.length || 0})
                </CardTitle>
                <CardDescription>
                  Running cpusim-server + collector-server
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {config.targetHosts?.map((host) => {
                    const hostStatus = hostsStatus?.targetHostsStatus?.find(h => h.name === host.name);
                    return (
                      <div key={host.name} className="border rounded-lg p-3 space-y-1">
                        <div className="flex items-center justify-between">
                          <div className="font-medium">{host.name}</div>
                          {hostStatus && (
                            <Badge variant={hostStatus.status === 'Running' ? 'default' : 'secondary'}>
                              {hostStatus.status}
                            </Badge>
                          )}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          External IP: {host.externalIP}
                        </div>
                        {host.internalIP && (
                          <div className="text-sm text-muted-foreground">
                            Internal IP: {host.internalIP}
                          </div>
                        )}
                        {hostStatus?.currentExperimentId && (
                          <div className="text-xs font-medium text-blue-600 pt-1">
                            Experiment: {hostStatus.currentExperimentId}
                          </div>
                        )}
                        {hostStatus?.error && (
                          <div className="text-xs text-red-500 pt-1">
                            Error: {hostStatus.error}
                          </div>
                        )}
                        <div className="text-xs text-muted-foreground pt-1">
                          CPU: {host.cpuServiceURL}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          Collector: {host.collectorServiceURL}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            {/* Client Host */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Laptop className="h-5 w-5" />
                  Client Host
                </CardTitle>
                <CardDescription>
                  Running requester-server
                </CardDescription>
              </CardHeader>
              <CardContent>
                {config.clientHost && (
                  <div className="border rounded-lg p-3 space-y-1">
                    <div className="flex items-center justify-between">
                      <div className="font-medium">{config.clientHost.name}</div>
                      {hostsStatus?.clientHostStatus && (
                        <Badge variant={hostsStatus.clientHostStatus.status === 'Running' ? 'default' : 'secondary'}>
                          {hostsStatus.clientHostStatus.status}
                        </Badge>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      External IP: {config.clientHost.externalIP}
                    </div>
                    {config.clientHost.internalIP && (
                      <div className="text-sm text-muted-foreground">
                        Internal IP: {config.clientHost.internalIP}
                      </div>
                    )}
                    {hostsStatus?.clientHostStatus?.currentExperimentId && (
                      <div className="text-xs font-medium text-blue-600 pt-1">
                        Experiment: {hostsStatus.clientHostStatus.currentExperimentId}
                      </div>
                    )}
                    {hostsStatus?.clientHostStatus?.error && (
                      <div className="text-xs text-red-500 pt-1">
                        Error: {hostsStatus.clientHostStatus.error}
                      </div>
                    )}
                    <div className="text-xs text-muted-foreground pt-1">
                      Requester: {config.clientHost.requesterServiceURL}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        )}

        {/* Experiment Control */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Play className="h-5 w-5" />
              Experiment Control
            </CardTitle>
            <CardDescription>
              Start or stop distributed experiments
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <Label htmlFor="experimentId">Experiment ID</Label>
                  <Input
                    id="experimentId"
                    value={experimentId}
                    onChange={(e) => setExperimentId(e.target.value)}
                    placeholder="exp-001"
                    disabled={isRunning}
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
                    disabled={isRunning}
                  />
                </div>
                <div>
                  <Label htmlFor="qps">QPS (Queries/sec)</Label>
                  <Input
                    id="qps"
                    type="number"
                    value={qps}
                    onChange={(e) => setQps(Number(e.target.value))}
                    min={1}
                    max={1000}
                    disabled={isRunning}
                  />
                </div>
              </div>

              <div className="flex gap-2">
                {!isRunning ? (
                  <Button
                    onClick={handleStartExperiment}
                    disabled={starting || !experimentId}
                    className="flex-1"
                  >
                    {starting ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Starting...
                      </>
                    ) : (
                      <>
                        <Play className="mr-2 h-4 w-4" />
                        Start Experiment
                      </>
                    )}
                  </Button>
                ) : (
                  <>
                    <Button
                      onClick={handleStopExperiment}
                      disabled={stopping}
                      variant="destructive"
                      className="flex-1"
                    >
                      {stopping ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Stopping...
                        </>
                      ) : (
                        <>
                          <Square className="mr-2 h-4 w-4" />
                          Stop Experiment
                        </>
                      )}
                    </Button>
                    <Button
                      onClick={() => handleViewData()}
                      variant="outline"
                    >
                      <Download className="mr-2 h-4 w-4" />
                      View Data
                    </Button>
                  </>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Stored Experiments List */}
        {experimentsList && experimentsList.experiments && experimentsList.experiments.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <History className="h-5 w-5" />
                Stored Experiments ({experimentsList.total})
              </CardTitle>
              <CardDescription>
                Past experiment results stored on disk
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {experimentsList.experiments
                  .sort((a, b) => new Date(b.modifiedAt || 0).getTime() - new Date(a.modifiedAt || 0).getTime())
                  .map((exp) => (
                    <div
                      key={exp.id}
                      className="border rounded-lg p-3 hover:bg-accent cursor-pointer transition-colors"
                      onClick={() => {
                        handleViewData(exp.id || '');
                      }}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <FileText className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{exp.id}</span>
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {exp.fileSizeKB} KB
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground mt-1">
                        Modified: {exp.modifiedAt ? new Date(exp.modifiedAt).toLocaleString() : 'Unknown'}
                      </div>
                    </div>
                  ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Experiment Data Display */}
        {experimentData && (
          <div className="space-y-6">
            {/* Experiment Summary */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Download className="h-5 w-5" />
                  Experiment Summary
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* Basic Info */}
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <div className="text-sm text-muted-foreground">Status</div>
                      <Badge className="mt-1">{experimentData.status}</Badge>
                    </div>
                    {experimentData.duration && (
                      <div>
                        <div className="text-sm text-muted-foreground">Duration</div>
                        <div className="text-lg font-medium">{experimentData.duration.toFixed(2)}s</div>
                      </div>
                    )}
                    <div>
                      <div className="text-sm text-muted-foreground">Time Range</div>
                      <div className="text-sm">
                        {experimentData.startTime && new Date(experimentData.startTime).toLocaleTimeString()} -
                        {experimentData.endTime && new Date(experimentData.endTime).toLocaleTimeString()}
                      </div>
                    </div>
                  </div>

                  {/* Errors */}
                  {experimentData.errors && experimentData.errors.length > 0 && (
                    <div>
                      <h3 className="font-semibold mb-2 text-red-500">Errors</h3>
                      <div className="space-y-1">
                        {experimentData.errors.map((error, idx) => (
                          <Alert key={idx} variant="destructive">
                            <AlertDescription>
                              [{error.phase}] {error.message}
                            </AlertDescription>
                          </Alert>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Requester Statistics */}
            {experimentData.requesterResult?.stats && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <BarChart3 className="h-5 w-5" />
                    Request Statistics
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <div className="text-sm text-muted-foreground">Total Requests</div>
                      <div className="text-2xl font-bold">{experimentData.requesterResult.stats.totalRequests || 0}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Successful</div>
                      <div className="text-2xl font-bold text-green-600">{experimentData.requesterResult.stats.successfulRequests || 0}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Failed</div>
                      <div className="text-2xl font-bold text-red-600">{experimentData.requesterResult.stats.failedRequests || 0}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">QPS</div>
                      <div className="text-2xl font-bold">{experimentData.requesterResult.stats.requestsPerSecond?.toFixed(2) || 0}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Avg Response Time</div>
                      <div className="text-xl font-bold">{experimentData.requesterResult.stats.averageResponseTime?.toFixed(2) || 0} ms</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">P50</div>
                      <div className="text-xl font-bold">{experimentData.requesterResult.stats.responseTimeP50?.toFixed(2) || 0} ms</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">P95</div>
                      <div className="text-xl font-bold">{experimentData.requesterResult.stats.responseTimeP95?.toFixed(2) || 0} ms</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">P99</div>
                      <div className="text-xl font-bold">{experimentData.requesterResult.stats.responseTimeP99?.toFixed(2) || 0} ms</div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Collector Metrics - CPU Charts */}
            {experimentData.collectorResults && Object.entries(experimentData.collectorResults).map(([hostName, result]) => {
              if (!result.data?.metrics || result.data.metrics.length === 0) return null;

              // Prepare chart data
              const chartData = result.data.metrics.map((metric: any, index: number) => ({
                index: index,
                time: new Date(metric.timestamp).toLocaleTimeString(),
                cpuUsage: metric.systemMetrics?.cpuUsagePercent || 0,
                memoryUsage: metric.systemMetrics?.memoryUsagePercent || 0,
                networkIn: (metric.systemMetrics?.networkIOBytes?.bytesReceived || 0) / 1024, // KB
                networkOut: (metric.systemMetrics?.networkIOBytes?.bytesSent || 0) / 1024, // KB
              }));

              return (
                <Card key={hostName}>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <LineChartIcon className="h-5 w-5" />
                      {hostName} - System Metrics ({result.data.metrics.length} data points)
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    {/* CPU Usage Chart */}
                    <div>
                      <h4 className="text-sm font-semibold mb-3">CPU Usage (%)</h4>
                      <ResponsiveContainer width="100%" height={250}>
                        <LineChart data={chartData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis
                            dataKey="time"
                            tick={{ fontSize: 12 }}
                            interval="preserveStartEnd"
                          />
                          <YAxis domain={[0, 100]} />
                          <Tooltip />
                          <Legend />
                          <Line
                            type="monotone"
                            dataKey="cpuUsage"
                            stroke="#8884d8"
                            strokeWidth={2}
                            name="CPU Usage (%)"
                            dot={false}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>

                    {/* Memory Usage Chart */}
                    <div>
                      <h4 className="text-sm font-semibold mb-3">Memory Usage (%)</h4>
                      <ResponsiveContainer width="100%" height={250}>
                        <LineChart data={chartData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis
                            dataKey="time"
                            tick={{ fontSize: 12 }}
                            interval="preserveStartEnd"
                          />
                          <YAxis domain={[0, 100]} />
                          <Tooltip />
                          <Legend />
                          <Line
                            type="monotone"
                            dataKey="memoryUsage"
                            stroke="#82ca9d"
                            strokeWidth={2}
                            name="Memory Usage (%)"
                            dot={false}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>

                    {/* Network I/O Chart */}
                    <div>
                      <h4 className="text-sm font-semibold mb-3">Network I/O (KB/s)</h4>
                      <ResponsiveContainer width="100%" height={250}>
                        <LineChart data={chartData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis
                            dataKey="time"
                            tick={{ fontSize: 12 }}
                            interval="preserveStartEnd"
                          />
                          <YAxis />
                          <Tooltip />
                          <Legend />
                          <Line
                            type="monotone"
                            dataKey="networkIn"
                            stroke="#ffc658"
                            strokeWidth={2}
                            name="Network In (KB/s)"
                            dot={false}
                          />
                          <Line
                            type="monotone"
                            dataKey="networkOut"
                            stroke="#ff7c7c"
                            strokeWidth={2}
                            name="Network Out (KB/s)"
                            dot={false}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
    </div>
  );
}
