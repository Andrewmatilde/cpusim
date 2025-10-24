import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Pagination } from './Pagination';
import { apiClient } from '@/api/client';
import type { ExperimentListResponse } from '@/api/types';
import { History, FileText, ArrowUpDown, AlertCircle, RefreshCw } from 'lucide-react';

type SortField = 'createdAt' | 'modifiedAt' | 'id';
type SortOrder = 'asc' | 'desc';

export function ExperimentsList() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [data, setData] = useState<ExperimentListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Read state from URL params with defaults
  const currentPage = parseInt(searchParams.get('page') || '1', 10);
  const pageSize = parseInt(searchParams.get('pageSize') || '10', 10);
  const sortBy = (searchParams.get('sortBy') || 'modifiedAt') as SortField;
  const sortOrder = (searchParams.get('sortOrder') || 'desc') as SortOrder;

  const fetchExperiments = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiClient.listExperiments({
        page: currentPage,
        pageSize,
        sortBy,
        sortOrder
      });
      setData(response);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load experiments';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchExperiments();
  }, [currentPage, pageSize, sortBy, sortOrder]);

  // Helper to update URL params
  const updateParams = (updates: Record<string, string>) => {
    const newParams = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([key, value]) => {
      newParams.set(key, value);
    });
    setSearchParams(newParams);
  };

  const handlePageChange = (page: number) => {
    updateParams({ page: page.toString() });
  };

  const handlePageSizeChange = (newPageSize: number) => {
    updateParams({
      pageSize: newPageSize.toString(),
      page: '1' // Reset to first page when changing page size
    });
  };

  const toggleSort = (field: SortField) => {
    const newOrder = sortBy === field && sortOrder === 'desc' ? 'asc' : 'desc';
    updateParams({
      sortBy: field,
      sortOrder: newOrder,
      page: '1' // Reset to first page when sorting
    });
  };

  const handleViewExperiment = (experimentId: string) => {
    // Preserve current pagination state when navigating to experiment detail
    navigate(`/experiment/${experimentId}`, {
      state: { from: `/?${searchParams.toString()}` }
    });
  };

  const SortButton = ({ field, label }: { field: SortField; label: string }) => (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => toggleSort(field)}
      className="h-8 px-2 text-xs"
    >
      {label}
      <ArrowUpDown className={`ml-1 h-3 w-3 ${sortBy === field ? 'text-primary' : 'text-muted-foreground'}`} />
      {sortBy === field && (
        <span className="ml-1 text-xs text-primary">
          {sortOrder === 'asc' ? '↑' : '↓'}
        </span>
      )}
    </Button>
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
              实验历史记录
            </CardTitle>
            <CardDescription>
              查看和管理已保存的实验数据
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={fetchExperiments}
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>
      </CardHeader>

      <CardContent>
        {error && (
          <Alert variant="destructive" className="mb-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Sort Controls */}
        <div className="flex items-center gap-2 mb-4 pb-3 border-b">
          <span className="text-sm text-muted-foreground">排序：</span>
          <SortButton field="modifiedAt" label="修改时间" />
          <SortButton field="createdAt" label="创建时间" />
          <SortButton field="id" label="实验ID" />
        </div>

        {/* Experiments List */}
        <div className="space-y-2 min-h-[400px]">
          {loading ? (
            // Loading skeleton
            Array.from({ length: pageSize }).map((_, i) => (
              <div key={i} className="border rounded-lg p-4">
                <Skeleton className="h-5 w-2/3 mb-2" />
                <Skeleton className="h-4 w-1/3" />
              </div>
            ))
          ) : data?.experiments && data.experiments.length > 0 ? (
            // Experiments list
            data.experiments.map((exp) => (
              <div
                key={exp.id}
                className="border rounded-lg p-4 hover:bg-accent cursor-pointer transition-colors"
                onClick={() => handleViewExperiment(exp.id || '')}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3 flex-1">
                    <FileText className="h-5 w-5 text-muted-foreground" />
                    <div className="flex-1">
                      <div className="font-medium text-base">{exp.id}</div>
                      <div className="flex items-center gap-4 mt-1 text-sm text-muted-foreground">
                        <span>
                          创建：{exp.createdAt ? new Date(exp.createdAt).toLocaleString('zh-CN') : '-'}
                        </span>
                        <span>
                          修改：{exp.modifiedAt ? new Date(exp.modifiedAt).toLocaleString('zh-CN') : '-'}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {exp.fileSizeKB} KB
                  </div>
                </div>
              </div>
            ))
          ) : (
            // Empty state
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <History className="h-12 w-12 mb-4 opacity-20" />
              <p className="text-lg font-medium">暂无实验数据</p>
              <p className="text-sm mt-1">开始一个新实验来查看数据</p>
            </div>
          )}
        </div>

        {/* Pagination */}
        {data && data.total > 0 && (
          <Pagination
            currentPage={currentPage}
            totalPages={data.totalPages || 1}
            pageSize={pageSize}
            total={data.total}
            onPageChange={handlePageChange}
            onPageSizeChange={handlePageSizeChange}
            loading={loading}
          />
        )}
      </CardContent>
    </Card>
  );
}
