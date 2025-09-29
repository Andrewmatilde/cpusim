import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { apiClient } from '@/api/client';
import type { Host, CreateExperimentRequest } from '@/api/types';
import { Plus, FlaskConical } from 'lucide-react';
import { toast } from 'sonner';

interface ExperimentFormProps {
  hosts: Host[];
  onExperimentCreated: () => void;
}

export function ExperimentForm({ hosts, onExperimentCreated }: ExperimentFormProps) {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<CreateExperimentRequest>({
    experimentId: `exp-${Date.now()}`,
    description: '',
    timeout: 300,
    collectionInterval: 1000,
    participatingHosts: []
  });

  const [selectedHosts, setSelectedHosts] = useState<Set<string>>(new Set());

  const handleHostToggle = (hostName: string, hostIp: string, checked: boolean) => {
    const newSelectedHosts = new Set(selectedHosts);

    if (checked) {
      newSelectedHosts.add(hostName);
      setFormData(prev => ({
        ...prev,
        participatingHosts: [
          ...prev.participatingHosts.filter(h => h.name !== hostName),
          { name: hostName, ip: hostIp }
        ]
      }));
    } else {
      newSelectedHosts.delete(hostName);
      setFormData(prev => ({
        ...prev,
        participatingHosts: prev.participatingHosts.filter(h => h.name !== hostName)
      }));
    }

    setSelectedHosts(newSelectedHosts);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (formData.participatingHosts.length === 0) {
      toast.error('Please select at least one host');
      return;
    }

    try {
      setLoading(true);
      await apiClient.createGlobalExperiment(formData);
      toast.success('Experiment created successfully');

      // Reset form
      setFormData({
        experimentId: `exp-${Date.now()}`,
        description: '',
        timeout: 300,
        collectionInterval: 1000,
        participatingHosts: []
      });
      setSelectedHosts(new Set());

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
          Create Global Experiment
        </CardTitle>
        <CardDescription>
          Create a new experiment that will run across multiple hosts simultaneously
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <Label htmlFor="experimentId">Experiment ID</Label>
            <Input
              id="experimentId"
              value={formData.experimentId}
              onChange={(e) => setFormData(prev => ({ ...prev, experimentId: e.target.value }))}
              placeholder="exp-001"
              required
              pattern="^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"
              title="Must be lowercase letters, numbers, and hyphens only"
            />
          </div>

          <div>
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              placeholder="Describe the experiment purpose and goals..."
              rows={3}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="timeout">Timeout (seconds)</Label>
              <Input
                id="timeout"
                type="number"
                min="60"
                max="3600"
                value={formData.timeout}
                onChange={(e) => setFormData(prev => ({ ...prev, timeout: parseInt(e.target.value) }))}
              />
            </div>
            <div>
              <Label htmlFor="collectionInterval">Collection Interval (milliseconds)</Label>
              <Input
                id="collectionInterval"
                type="number"
                min="100"
                max="10000"
                value={formData.collectionInterval}
                onChange={(e) => setFormData(prev => ({ ...prev, collectionInterval: parseInt(e.target.value) }))}
              />
            </div>
          </div>

          <div>
            <Label className="text-base font-medium">Participating Hosts</Label>
            <div className="mt-2 space-y-2 max-h-48 overflow-y-auto border rounded-md p-3">
              {hosts.length === 0 ? (
                <p className="text-sm text-muted-foreground">No hosts available</p>
              ) : (
                hosts.map((host) => (
                  <div key={host.name} className="flex items-center space-x-2">
                    <Checkbox
                      id={`host-${host.name}`}
                      checked={selectedHosts.has(host.name || '')}
                      onCheckedChange={(checked) =>
                        handleHostToggle(host.name || '', host.ip || '', !!checked)
                      }
                    />
                    <Label
                      htmlFor={`host-${host.name}`}
                      className="text-sm font-normal flex-1 cursor-pointer"
                    >
                      {host.name} ({host.ip})
                    </Label>
                  </div>
                ))
              )}
            </div>
            {selectedHosts.size > 0 && (
              <p className="mt-1 text-sm text-muted-foreground">
                {selectedHosts.size} host(s) selected
              </p>
            )}
          </div>

          <Button
            type="submit"
            disabled={loading || hosts.length === 0}
            className="w-full"
          >
            <Plus className="h-4 w-4 mr-2" />
            {loading ? 'Creating...' : 'Create Experiment'}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}