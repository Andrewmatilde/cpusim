# 请求发送服务 (Request Sender Service)

## 概述

请求发送服务专门用于向CPU仿真服务发送负载请求，通过精确的QPS控制来模拟不同强度的网络流量负载。

## 核心设计

### 固定请求配置
- **目标服务**: CPU仿真服务 (cpusim-server)
- **请求路径**: `POST /calculate`
- **请求体**: `{}` (空JSON对象)
- **请求头**: `Content-Type: application/json`

### QPS控制机制
- 基于QPS参数精确控制请求频率
- **请求间隔** = 1/QPS 秒
- 每个时间间隔启动一个goroutine发送请求
- 支持1-1000 QPS的配置范围

### HTTP连接优化
为支持高并发请求，HTTP Transport需要优化配置：

```go
transport := &http.Transport{
    MaxIdleConns:        100,              // 最大空闲连接数
    MaxIdleConnsPerHost: 100,              // 每个host最大空闲连接数
    IdleConnTimeout:     90 * time.Second, // 空闲连接超时
    DisableKeepAlives:   false,            // 启用连接复用
}

client := &http.Client{
    Transport: transport,
    Timeout:   5 * time.Second, // 单个请求超时
}
```

## API接口

### 启动实验

**请求:**
```bash
curl -X POST http://localhost:80/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "req-exp-001",
    "targetIP": "192.168.1.100",
    "targetPort": 80,
    "timeout": 300,
    "qps": 10,
    "description": "CPU负载测试 - 10 QPS"
  }'
```

**响应:**
```json
{
  "experimentId": "req-exp-001",
  "targetIP": "192.168.1.100",
  "targetPort": 80,
  "timeout": 300,
  "qps": 10,
  "description": "CPU负载测试 - 10 QPS",
  "status": "running",
  "startTime": "2025-09-29T17:30:00Z",
  "createdAt": "2025-09-29T17:30:00Z"
}
```

### 获取实验状态

```bash
curl http://localhost:80/experiments/req-exp-001
```

### 停止实验

```bash
curl -X POST http://localhost:80/experiments/req-exp-001/stop
```

### 获取统计数据

```bash
curl http://localhost:80/experiments/req-exp-001/stats
```

**统计数据示例:**
```json
{
  "experimentId": "req-exp-001",
  "status": "completed",
  "totalRequests": 1000,
  "successfulRequests": 998,
  "failedRequests": 2,
  "averageResponseTime": 2.45,
  "minResponseTime": 1.89,
  "maxResponseTime": 5.67,
  "requestsPerSecond": 9.8,
  "errorRate": 0.2,
  "responseTimeP50": 2.34,
  "responseTimeP95": 3.12,
  "responseTimeP99": 4.56,
  "startTime": "2025-09-29T17:30:00Z",
  "endTime": "2025-09-29T17:31:42Z",
  "duration": 102
}
```

## 实现要点

### 1. QPS控制算法
```go
ticker := time.NewTicker(time.Second / time.Duration(qps))
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        go sendRequest(targetURL)
    case <-stopChan:
        return
    case <-timeoutTimer.C:
        return
    }
}
```

### 2. 超时控制
```go
timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
defer timeoutTimer.Stop()

for {
    select {
    case <-ticker.C:
        go sendRequest(targetURL)
    case <-stopChan:
        return // 手动停止
    case <-timeoutTimer.C:
        return // 超时停止
    }
}
```

### 3. 统计数据收集
```go
type RequestStats struct {
    mu sync.RWMutex

    TotalRequests     int64
    SuccessfulRequests int64
    FailedRequests    int64
    ResponseTimes     []float64
    StartTime         time.Time
    EndTime           *time.Time
}

func (s *RequestStats) RecordRequest(duration time.Duration, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.TotalRequests++
    if err != nil {
        s.FailedRequests++
    } else {
        s.SuccessfulRequests++
        s.ResponseTimes = append(s.ResponseTimes, duration.Seconds()*1000) // 毫秒
    }
}
```

### 4. 响应时间分位数计算
```go
import "sort"

func (s *RequestStats) CalculatePercentiles() (p50, p95, p99 float64) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if len(s.ResponseTimes) == 0 {
        return 0, 0, 0
    }

    sorted := make([]float64, len(s.ResponseTimes))
    copy(sorted, s.ResponseTimes)
    sort.Float64s(sorted)

    p50 = percentile(sorted, 0.5)
    p95 = percentile(sorted, 0.95)
    p99 = percentile(sorted, 0.99)

    return p50, p95, p99
}

func percentile(sorted []float64, p float64) float64 {
    index := int(float64(len(sorted)) * p)
    if index >= len(sorted) {
        index = len(sorted) - 1
    }
    return sorted[index]
}
```

## 使用场景

### 1. CPU负载压测
```bash
# 低强度负载 - 1 QPS
curl -X POST http://localhost:80/experiments \
  -d '{"experimentId": "low-load", "targetIP": "192.168.1.100", "targetPort": 80, "timeout": 600, "qps": 1}'

# 中等强度负载 - 50 QPS
curl -X POST http://localhost:80/experiments \
  -d '{"experimentId": "med-load", "targetIP": "192.168.1.100", "targetPort": 80, "timeout": 300, "qps": 50}'

# 高强度负载 - 200 QPS
curl -X POST http://localhost:80/experiments \
  -d '{"experimentId": "high-load", "targetIP": "192.168.1.100", "targetPort": 80, "timeout": 120, "qps": 200}'
```

### 2. 性能基线测试
```bash
# 建立性能基线
curl -X POST http://localhost:80/experiments \
  -d '{"experimentId": "baseline", "targetIP": "192.168.1.100", "targetPort": 80, "timeout": 300, "qps": 10, "maxRequests": 1000}'
```

### 3. 容量规划测试
```bash
# 逐步增加负载找到性能拐点
for qps in 10 20 50 100 200 500; do
  curl -X POST http://localhost:80/experiments \
    -d "{\"experimentId\": \"capacity-$qps\", \"targetIP\": \"192.168.1.100\", \"targetPort\": 80, \"timeout\": 60, \"qps\": $qps}"
  sleep 70  # 等待实验完成
done
```

## 错误处理

### 常见错误情况
1. **目标服务不可达**: 网络连接失败
2. **请求超时**: 单个请求响应超时
3. **服务过载**: 目标服务返回5xx错误
4. **QPS过高**: 请求发送服务资源不足

### 错误统计
- 所有错误都会被记录到 `failedRequests` 计数器
- 错误率 = failedRequests / totalRequests * 100%
- 建议错误率控制在5%以下

## 监控指标

### 关键指标
- **QPS**: 实际每秒请求数
- **响应时间**: P50、P95、P99分位数
- **错误率**: 失败请求百分比
- **吞吐量**: 成功请求总数/实验时长

### 性能基准
- **正常响应时间**: < 10ms (P95)
- **可接受错误率**: < 5%
- **目标QPS达成率**: > 95%

## 部署建议

### 资源配置
- **CPU**: 2核心 (支持500+ QPS)
- **内存**: 512MB (统计数据缓存)
- **网络**: 千兆网卡 (避免网络瓶颈)

### 配置优化
- 调整操作系统文件描述符限制: `ulimit -n 65536`
- 优化TCP内核参数以支持高并发连接
- 使用SSD存储以提高日志写入性能