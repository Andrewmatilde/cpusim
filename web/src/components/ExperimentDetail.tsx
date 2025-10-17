import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Download, LineChart as LineChartIcon, BarChart3, ArrowLeft } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { apiClient } from '@/api/client';
import type { ExperimentData } from '@/api/types';
import { toast } from 'sonner';

export function ExperimentDetail() {
  const { experimentId } = useParams<{ experimentId: string }>();
  const navigate = useNavigate();
  const [experimentData, setExperimentData] = useState<ExperimentData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!experimentId) {
      navigate('/');
      return;
    }

    const loadExperiment = async () => {
      try {
        setLoading(true);
        const data = await apiClient.getExperimentData(experimentId);
        setExperimentData(data);
        toast.success('Experiment data loaded');
      } catch (error) {
        toast.error('Failed to load experiment data');
        console.error('Load data error:', error);
        navigate('/');
      } finally {
        setLoading(false);
      }
    };

    loadExperiment();
  }, [experimentId, navigate]);

  if (loading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">Loading experiment data...</div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!experimentData) {
    return null;
  }

  return (
    <div className="space-y-6">
      {/* Header with Back Button */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Download className="h-5 w-5" />
              Experiment: {experimentId}
            </CardTitle>
            <Button onClick={() => navigate('/')} variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Dashboard
            </Button>
          </div>
        </CardHeader>
      </Card>

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
    </div>
  );
}
