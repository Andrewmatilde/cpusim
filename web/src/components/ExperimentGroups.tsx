import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiClient } from '@/api/client';
import type {
  ExperimentGroupListResponse,
  StartExperimentGroupRequest,
  ExperimentGroup
} from '@/api/generated';
import { RefreshCw, AlertCircle, Play, Loader2, Layers, FileText, BarChart3, Clock, RotateCw } from 'lucide-react';
import { toast } from 'sonner';

export function ExperimentGroups() {
  const navigate = useNavigate();
  const [groupsList, setGroupsList] = useState<ExperimentGroupListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Form state
  const [groupId, setGroupId] = useState(`group-${Date.now()}`);
  const [description, setDescription] = useState('');
  const [qpsMin, setQpsMin] = useState(100);
  const [qpsMax, setQpsMax] = useState(500);
  const [qpsStep, setQpsStep] = useState(100);
  const [repeatCount, setRepeatCount] = useState(10);
  const [timeout, setTimeout] = useState(60);
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
      const qpsValues = [];
      for (let qps = qpsMin; qps <= qpsMax; qps += qpsStep) {
        qpsValues.push(qps);
      }
      const request: StartExperimentGroupRequest = {
        groupId,
        description,
        qpsMin,
        qpsMax,
        qpsStep,
        repeatCount,
        timeout,
        delayBetween
      };
      await apiClient.startExperimentGroup(request);
      toast.success(`Experiment group started: ${qpsValues.length} QPS points × ${repeatCount} runs = ${qpsValues.length * repeatCount} total experiments`);
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

  const handleViewGroup = (groupId: string) => {
    navigate(`/groups/${groupId}`);
  };

  const handleResumeGroup = async (gId: string, e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent triggering view group
    try {
      await apiClient.resumeExperimentGroup({ groupId: gId });
      toast.success(`Experiment group resumed: ${gId}`);
      fetchData();
    } catch (err) {
      toast.error('Failed to resume experiment group');
      console.error('Resume group error:', err);
    }
  };

  const formatDuration = (start: Date | string, end?: Date | string) => {
    if (!end) return 'In progress...';
    const endDate = end instanceof Date ? end : new Date(end);
    // Check if end time is zero value (0001-01-01)
    if (endDate.getFullYear() < 1900) return 'In progress...';
    const startTime = start instanceof Date ? start.getTime() : new Date(start).getTime();
    const endTime = endDate.getTime();
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
            Test QPS range with multiple repetitions per QPS value
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

            <div className="grid grid-cols-3 gap-4">
              <div>
                <Label htmlFor="qpsMin">QPS Min</Label>
                <Input
                  id="qpsMin"
                  type="number"
                  value={qpsMin}
                  onChange={(e) => setQpsMin(Number(e.target.value))}
                  min={1}
                  max={1000}
                />
              </div>
              <div>
                <Label htmlFor="qpsMax">QPS Max</Label>
                <Input
                  id="qpsMax"
                  type="number"
                  value={qpsMax}
                  onChange={(e) => setQpsMax(Number(e.target.value))}
                  min={1}
                  max={1000}
                />
              </div>
              <div>
                <Label htmlFor="qpsStep">QPS Step</Label>
                <Input
                  id="qpsStep"
                  type="number"
                  value={qpsStep}
                  onChange={(e) => setQpsStep(Number(e.target.value))}
                  min={1}
                  max={1000}
                />
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4">
              <div>
                <Label htmlFor="repeatCount">Repeat Count (per QPS)</Label>
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
                        {group.status === 'running' || group.status === 'failed' ? (
                          <Button
                            onClick={(e) => group.groupId && handleResumeGroup(group.groupId, e)}
                            variant="outline"
                            size="sm"
                            className="ml-2"
                          >
                            <RotateCw className="h-3 w-3 mr-1" />
                            Resume
                          </Button>
                        ) : null}
                      </div>
                      {group.description && (
                        <div className="text-sm text-muted-foreground mb-2">
                          {group.description}
                        </div>
                      )}
                      <div className="grid grid-cols-2 gap-2 text-sm">
                        <div className="flex items-center gap-1">
                          <BarChart3 className="h-3 w-3 text-muted-foreground" />
                          <span className="text-muted-foreground">QPS Range:</span>
                          <span className="font-medium">{group.config?.qpsMin}-{group.config?.qpsMax} (step {group.config?.qpsStep})</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <Clock className="h-3 w-3 text-muted-foreground" />
                          <span className="text-muted-foreground">Duration:</span>
                          <span className="font-medium">{group.startTime && formatDuration(group.startTime, group.endTime)}</span>
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground mt-1">
                        Progress: QPS {group.currentQPS}, Run {group.currentRun}/{group.config?.repeatCount}
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
    </div>
  );
}
