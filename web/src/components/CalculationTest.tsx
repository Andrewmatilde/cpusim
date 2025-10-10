import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { apiClient } from '@/api/client';
import type { Host, CalculationRequest, CalculationResponse } from '@/api/types';
import { Calculator, Clock, Activity } from 'lucide-react';
import { toast } from 'sonner';

interface CalculationTestProps {
  host: Host;
}

export function CalculationTest({ host }: CalculationTestProps) {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<CalculationRequest>({
    a: 12345678,
    b: 87654321
  });
  const [result, setResult] = useState<CalculationResponse | null>(null);

  const handleRunTest = async () => {
    try {
      setLoading(true);
      setResult(null);
      const response = await apiClient.testHostCalculation(host.name || '', formData);
      setResult(response);
      toast.success('Calculation completed successfully');
    } catch (error) {
      toast.error('Failed to run calculation test');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Calculator className="h-5 w-5" />
          CPU Calculation Test
        </CardTitle>
        <CardDescription>
          Test CPU performance by calculating GCD (Greatest Common Divisor)
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <Label htmlFor="a">First Number</Label>
            <Input
              id="a"
              type="number"
              value={formData.a}
              onChange={(e) => setFormData(prev => ({ ...prev, a: parseInt(e.target.value) || 0 }))}
              placeholder="Enter a number"
            />
          </div>
          <div>
            <Label htmlFor="b">Second Number</Label>
            <Input
              id="b"
              type="number"
              value={formData.b}
              onChange={(e) => setFormData(prev => ({ ...prev, b: parseInt(e.target.value) || 0 }))}
              placeholder="Enter a number"
            />
          </div>
        </div>

        <Button
          onClick={handleRunTest}
          disabled={loading}
          className="w-full"
        >
          {loading ? (
            <>
              <Activity className="h-4 w-4 mr-2 animate-spin" />
              Running Test...
            </>
          ) : (
            <>
              <Calculator className="h-4 w-4 mr-2" />
              Run Calculation Test
            </>
          )}
        </Button>

        {result && (
          <Alert>
            <AlertDescription className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-medium">GCD Result:</span>
                <span className="font-mono text-lg">{result.gcd}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="font-medium flex items-center gap-1">
                  <Clock className="h-4 w-4" />
                  Processing Time:
                </span>
                <span className="font-mono">{result.processTime}</span>
              </div>
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  );
}