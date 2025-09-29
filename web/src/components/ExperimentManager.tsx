import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { apiClient } from '@/api/client';
import type { Host, ExperimentStatus, ExperimentRequest } from '@/api/types';
import { Play, Square, RefreshCw, Activity, Cpu, MemoryStick, Network } from 'lucide-react';
import { toast } from 'sonner';

interface ExperimentManagerProps {
  host: Host;
}

export function ExperimentManager({ host }: ExperimentManagerProps) {
  const [experiments, setExperiments] = useState<ExperimentStatus[]>([]);
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<ExperimentRequest>({
    experimentId: `exp-${Date.now()}`,
    description: '',
    timeout: 300,
    collectionInterval: 5
  });

  const fetchExperiments = async () => {
    try {
      const response = await apiClient.getHostExperiments(host.name);
      setExperiments(response.experiments || []);
    } catch (error) {
      toast.error('Failed to fetch experiments');
    }
  };

  useEffect(() => {
    fetchExperiments();
    const interval = setInterval(fetchExperiments, 5000);
    return () => clearInterval(interval);
  }, [host.name]);

  const handleStartExperiment = async () => {
    try {
      setLoading(true);
      await apiClient.startHostExperiment(host.name, formData);
      toast.success('Experiment started successfully');
      setFormData({
        experimentId: `exp-${Date.now()}`,
        description: '',
        timeout: 300,
        collectionInterval: 5
      });
      await fetchExperiments();
    } catch (error) {
      toast.error('Failed to start experiment');
    } finally {
      setLoading(false);
    }
  };

  const handleStopExperiment = async (experimentId: string) => {
    try {
      await apiClient.stopHostExperiment(host.name, experimentId);
      toast.success('Experiment stopped');
      await fetchExperiments();
    } catch (error) {
      toast.error('Failed to stop experiment');
    }
  };

  const formatMetrics = (metrics: ExperimentStatus['lastMetrics']) => {
    if (!metrics) return 'No data';
    return (
      <div className="space-y-1 text-xs">
        <div className="flex items-center gap-2">
          <Cpu className="h-3 w-3" />
          CPU: {metrics.cpuUsagePercent?.toFixed(2)}%
        </div>
        <div className="flex items-center gap-2">
          <MemoryStick className="h-3 w-3" />
          Memory: {metrics.memoryUsagePercent?.toFixed(2)}%
        </div>
        {metrics.networkIOBytes && (
          <div className="flex items-center gap-2">
            <Network className="h-3 w-3" />
            Net: ↓{(metrics.networkIOBytes.bytesReceived / 1024).toFixed(0)}KB ↑{(metrics.networkIOBytes.bytesSent / 1024).toFixed(0)}KB
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-6 pb-6">
      <Card>
        <CardHeader>
          <CardTitle>New Experiment</CardTitle>
          <CardDescription>Configure and start a new experiment on {host.name}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="experimentId">Experiment ID</Label>
                <Input
                  id="experimentId"
                  value={formData.experimentId}
                  onChange={(e) => setFormData(prev => ({ ...prev, experimentId: e.target.value }))}
                  placeholder="exp-001"
                />
              </div>
              <div>
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={formData.timeout}
                  onChange={(e) => setFormData(prev => ({ ...prev, timeout: parseInt(e.target.value) }))}
                />
              </div>
            </div>
            <div>
              <Label htmlFor="interval">Collection Interval (seconds)</Label>
              <Input
                id="interval"
                type="number"
                value={formData.collectionInterval}
                onChange={(e) => setFormData(prev => ({ ...prev, collectionInterval: parseInt(e.target.value) }))}
              />
            </div>
            <div>
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                placeholder="Describe the experiment..."
                rows={3}
              />
            </div>
            <Button onClick={handleStartExperiment} disabled={loading}>
              <Play className="h-4 w-4 mr-2" />
              Start Experiment
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Running Experiments</CardTitle>
              <CardDescription>{experiments.length} experiment(s) found</CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={fetchExperiments}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {experiments.length === 0 ? (
            <Alert>
              <AlertDescription>No experiments running on this host</AlertDescription>
            </Alert>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Data Points</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Last Metrics</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {experiments.map((exp) => (
                  <TableRow key={exp.experimentId}>
                    <TableCell className="font-medium">{exp.experimentId}</TableCell>
                    <TableCell>
                      <Badge variant={exp.isActive ? "default" : "secondary"}>
                        {exp.status}
                      </Badge>
                    </TableCell>
                    <TableCell>{exp.dataPointsCollected}</TableCell>
                    <TableCell>{exp.duration ? `${exp.duration}s` : '-'}</TableCell>
                    <TableCell>{formatMetrics(exp.lastMetrics)}</TableCell>
                    <TableCell>
                      {exp.isActive && (
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleStopExperiment(exp.experimentId)}
                        >
                          <Square className="h-3 w-3 mr-1" />
                          Stop
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}