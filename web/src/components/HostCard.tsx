import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { Host, HostHealth } from '@/api/types';
import { Activity, Server, Wifi, WifiOff } from 'lucide-react';

interface HostCardProps {
  host: Host;
  health?: HostHealth;
  onViewDetails: (host: Host) => void;
  onRunTest: (host: Host) => void;
}

export function HostCard({ host, health, onViewDetails, onRunTest }: HostCardProps) {
  const isHealthy = health?.cpuServiceHealthy && health?.collectorServiceHealthy;

  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5" />
              {host.name}
            </CardTitle>
            <CardDescription>{host.ip}</CardDescription>
          </div>
          <Badge variant={isHealthy ? "default" : health ? "destructive" : "secondary"}>
            {isHealthy ? (
              <>
                <Wifi className="h-3 w-3 mr-1" />
                Online
              </>
            ) : health ? (
              <>
                <WifiOff className="h-3 w-3 mr-1" />
                Offline
              </>
            ) : (
              "Unknown"
            )}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">CPU Service:</span>
            <Badge variant={health?.cpuServiceHealthy ? "success" : "destructive"} className="text-xs">
              {health?.cpuServiceHealthy ? "Healthy" : "Unhealthy"}
            </Badge>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Collector Service:</span>
            <Badge variant={health?.collectorServiceHealthy ? "success" : "destructive"} className="text-xs">
              {health?.collectorServiceHealthy ? "Healthy" : "Unhealthy"}
            </Badge>
          </div>
          {health?.collectorHealth && (
            <div className="text-xs text-muted-foreground mt-2">
              Last updated: {new Date(health.collectorHealth.timestamp).toLocaleTimeString()}
            </div>
          )}
        </div>

        <div className="flex gap-2">
          <Button
            onClick={() => onViewDetails(host)}
            variant="outline"
            size="sm"
            className="flex-1"
          >
            <Activity className="h-4 w-4 mr-1" />
            Details
          </Button>
          <Button
            onClick={() => onRunTest(host)}
            size="sm"
            className="flex-1"
            disabled={!isHealthy}
          >
            Run Test
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}