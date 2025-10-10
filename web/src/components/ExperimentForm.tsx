import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { apiClient } from '@/api/client';
import type { Host, CreateExperimentRequest, HostConfig, RequestConfig } from '@/api/types';
import { Plus, FlaskConical, Server, Laptop, Loader2 } from 'lucide-react';
import { toast } from 'sonner';

interface ExperimentFormProps {
  hosts: Host[];
  onExperimentCreated: () => void;
}

export function ExperimentForm({ hosts, onExperimentCreated }: ExperimentFormProps) {
  const [loading, setLoading] = useState(false);
  const [selectedTargetHosts, setSelectedTargetHosts] = useState<Set<string>>(new Set());
  const [selectedClientHost, setSelectedClientHost] = useState<string>('');
  const [targetForRequest, setTargetForRequest] = useState<string>('');

  const [formData, setFormData] = useState({
    experimentId: `exp-${Date.now()}`,
    description: '',
    timeout: 300,
    collectionInterval: 1000,
    qps: 10,
    duration: 60
  });

  // Separate hosts by type
  const targetHosts = hosts.filter(h => h.hostType === 'target' || !h.hostType);
  const clientHosts = hosts.filter(h => h.hostType === 'client');

  const handleTargetHostToggle = (hostName: string, checked: boolean) => {
    const newSelectedHosts = new Set(selectedTargetHosts);
    if (checked) {
      newSelectedHosts.add(hostName);
    } else {
      newSelectedHosts.delete(hostName);
      // If unchecking the target for requests, clear it
      if (targetForRequest === hostName) {
        setTargetForRequest('');
      }
    }
    setSelectedTargetHosts(newSelectedHosts);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (selectedTargetHosts.size === 0) {
      toast.error('Please select at least one target host');
      return;
    }

    if (!selectedClientHost) {
      toast.error('Please select a client host');
      return;
    }

    if (!targetForRequest) {
      toast.error('Please select a target host for requests');
      return;
    }

    // Build request
    const targetHostsArray: HostConfig[] = Array.from(selectedTargetHosts).map(name => {
      const host = hosts.find(h => h.name === name)!;
      return {
        name: host.name!,
        externalIP: host.externalIP!,
        internalIP: host.internalIP
      };
    });

    const clientHost = hosts.find(h => h.name === selectedClientHost)!;
    const clientHostConfig: HostConfig = {
      name: clientHost.name!,
      externalIP: clientHost.externalIP!,
      internalIP: clientHost.internalIP
    };

    const requestConfig: RequestConfig = {
      qps: formData.qps,
      requestTimeout: formData.duration,
      targetHostName: targetForRequest
    };

    const request: CreateExperimentRequest = {
      experimentId: formData.experimentId,
      description: formData.description,
      timeout: formData.timeout,
      collectionInterval: formData.collectionInterval,
      targetHosts: targetHostsArray,
      clientHost: clientHostConfig,
      requestConfig: requestConfig
    };

    try {
      setLoading(true);
      await apiClient.createGlobalExperiment(request);
      toast.success('Experiment created successfully');

      // Reset form
      setFormData({
        experimentId: `exp-${Date.now()}`,
        description: '',
        timeout: 300,
        collectionInterval: 1000,
        qps: 10,
        duration: 60
      });
      setSelectedTargetHosts(new Set());
      setSelectedClientHost('');
      setTargetForRequest('');

      onExperimentCreated();
    } catch (error) {
      toast.error('Failed to create experiment');
      console.error('Create experiment error:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FlaskConical className="h-5 w-5" />
          Create New Experiment
        </CardTitle>
        <CardDescription>
          Configure and start a new load testing experiment
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Information */}
          <div className="space-y-4">
            <div>
              <Label htmlFor="experimentId">Experiment ID</Label>
              <Input
                id="experimentId"
                value={formData.experimentId}
                onChange={(e) => setFormData(prev => ({ ...prev, experimentId: e.target.value }))}
                placeholder="exp-001"
                required
              />
            </div>

            <div>
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                placeholder="Describe the experiment purpose..."
                rows={3}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={formData.timeout}
                  onChange={(e) => setFormData(prev => ({ ...prev, timeout: Number(e.target.value) }))}
                  min={60}
                  max={3600}
                />
              </div>
              <div>
                <Label htmlFor="collectionInterval">Collection Interval (ms)</Label>
                <Input
                  id="collectionInterval"
                  type="number"
                  value={formData.collectionInterval}
                  onChange={(e) => setFormData(prev => ({ ...prev, collectionInterval: Number(e.target.value) }))}
                  min={100}
                  max={10000}
                />
              </div>
            </div>
          </div>

          {/* Target Hosts Selection */}
          <div>
            <Label className="flex items-center gap-2 mb-2">
              <Server className="h-4 w-4" />
              Target Hosts (Running cpusim-server + collector)
            </Label>
            <div className="border rounded-lg p-3 space-y-2 max-h-48 overflow-y-auto">
              {targetHosts.length === 0 ? (
                <p className="text-sm text-muted-foreground">No target hosts available</p>
              ) : (
                targetHosts.map(host => (
                  <div key={host.name} className="flex items-center space-x-2">
                    <Checkbox
                      id={`target-${host.name}`}
                      checked={selectedTargetHosts.has(host.name!)}
                      onCheckedChange={(checked) => handleTargetHostToggle(host.name!, checked as boolean)}
                    />
                    <Label
                      htmlFor={`target-${host.name}`}
                      className="flex-1 cursor-pointer text-sm font-normal"
                    >
                      {host.name} ({host.externalIP})
                    </Label>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Client Host Selection */}
          <div>
            <Label className="flex items-center gap-2 mb-2">
              <Laptop className="h-4 w-4" />
              Client Host (Running requester)
            </Label>
            <Select value={selectedClientHost} onValueChange={setSelectedClientHost}>
              <SelectTrigger>
                <SelectValue placeholder="Select client host" />
              </SelectTrigger>
              <SelectContent>
                {clientHosts.length === 0 ? (
                  <div className="p-2 text-sm text-muted-foreground">No client hosts available</div>
                ) : (
                  clientHosts.map(host => (
                    <SelectItem key={host.name} value={host.name!}>
                      {host.name} ({host.externalIP})
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </div>

          {/* Request Configuration */}
          <div>
            <Label className="mb-2">Request Configuration</Label>
            <div className="space-y-4 border rounded-lg p-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="qps">QPS (Queries Per Second)</Label>
                  <Input
                    id="qps"
                    type="number"
                    value={formData.qps}
                    onChange={(e) => setFormData(prev => ({ ...prev, qps: Number(e.target.value) }))}
                    min={1}
                    max={1000}
                  />
                </div>
                <div>
                  <Label htmlFor="duration">Duration (seconds)</Label>
                  <Input
                    id="duration"
                    type="number"
                    value={formData.duration}
                    onChange={(e) => setFormData(prev => ({ ...prev, duration: Number(e.target.value) }))}
                    min={10}
                    max={600}
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="targetForRequest">Target Host for Requests</Label>
                <Select value={targetForRequest} onValueChange={setTargetForRequest}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select target for requests" />
                  </SelectTrigger>
                  <SelectContent>
                    {Array.from(selectedTargetHosts).map(hostName => (
                      <SelectItem key={hostName} value={hostName}>
                        {hostName}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground mt-1">
                  The client will send requests to this target host
                </p>
              </div>
            </div>
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating Experiment...
              </>
            ) : (
              <>
                <Plus className="mr-2 h-4 w-4" />
                Create Experiment
              </>
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}