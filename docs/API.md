# CPU仿真测试平台 API文档

## 概述

CPU仿真测试平台提供三套RESTful API：
- **CPU仿真服务API** (端口80): 执行GCD计算
- **指标收集器API** (端口8080): 系统监控和实验管理
- **管理仪表盘API** (端口9090): 多主机统一管理

所有API均使用JSON格式进行数据交换，支持跨域请求(CORS)。

---

## CPU仿真服务API (端口80)

### POST /calculate
执行GCD计算任务

**请求:**
```json
{}
```

**响应:**
```json
{
  "gcd": "1",
  "process_time": "2.356ms"
}
```

**响应字段:**
- `gcd`: 计算结果（两个300位斐波那契数的最大公约数）
- `process_time`: 计算耗时（毫秒）

---

## 指标收集器API (端口8080)

### GET /health
健康检查接口

**响应:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-29T08:30:00Z"
}
```

### GET /experiments
获取所有实验列表

**响应:**
```json
{
  "experiments": [
    {
      "experimentId": "exp-001",
      "description": "CPU负载测试",
      "status": "running",
      "startTime": "2025-09-29T08:30:00Z",
      "collectionInterval": 1000,
      "timeout": 300
    }
  ]
}
```

### POST /experiments
创建新实验

**请求:**
```json
{
  "experimentId": "exp-001",
  "description": "CPU负载测试",
  "timeout": 300,
  "collectionInterval": 1000
}
```

**请求字段:**
- `experimentId`: 实验唯一标识符
- `description`: 实验描述（可选）
- `timeout`: 超时时间（秒）
- `collectionInterval`: 数据采集间隔（毫秒）

**响应:**
```json
{
  "experimentId": "exp-001",
  "status": "started",
  "startTime": "2025-09-29T08:30:00Z"
}
```

### GET /experiments/{experimentId}
获取实验状态

**响应:**
```json
{
  "experimentId": "exp-001",
  "description": "CPU负载测试",
  "status": "running",
  "startTime": "2025-09-29T08:30:00Z",
  "duration": 45,
  "collectionInterval": 1000,
  "timeout": 300
}
```

### GET /experiments/{experimentId}/data
获取实验数据

**响应:**
```json
{
  "experimentId": "exp-001",
  "startTime": "2025-09-29T08:30:00Z",
  "endTime": "2025-09-29T08:35:00Z",
  "duration": 300,
  "collectionInterval": 1000,
  "metrics": [
    {
      "timestamp": "2025-09-29T08:30:01Z",
      "systemMetrics": {
        "cpuUsagePercent": 25.5,
        "memoryUsageBytes": 134217728,
        "memoryUsagePercent": 12.8,
        "calculatorServiceHealthy": true,
        "networkIOBytes": {
          "bytesReceived": 1024,
          "bytesSent": 2048,
          "packetsReceived": 10,
          "packetsSent": 15
        }
      }
    }
  ]
}
```

### POST /experiments/{experimentId}/stop
停止实验

**响应:**
```json
{
  "experimentId": "exp-001",
  "status": "stopped",
  "endTime": "2025-09-29T08:35:00Z",
  "duration": 300
}
```

---

## 管理仪表盘API (端口9090)

### GET /api/hosts
获取所有主机列表

**响应:**
```json
{
  "hosts": [
    {
      "name": "主机1",
      "ip": "192.168.1.100",
      "cpuServiceUrl": "http://192.168.1.100:80",
      "collectorServiceUrl": "http://192.168.1.100:8080"
    }
  ]
}
```

### GET /api/hosts/{hostName}/health
获取主机健康状态

**响应:**
```json
{
  "name": "主机1",
  "ip": "192.168.1.100",
  "timestamp": "2025-09-29T08:30:00Z",
  "cpuServiceHealthy": true,
  "collectorServiceHealthy": true
}
```

### POST /api/hosts/{hostName}/calculate
在指定主机执行计算测试

**请求:**
```json
{
  "a": 12345,
  "b": 67890
}
```

**响应:**
```json
{
  "gcd": "15",
  "processTime": "2.356"
}
```

### GET /api/experiments
获取全局实验列表

**查询参数:**
- `limit`: 返回数量限制（默认20）

**响应:**
```json
{
  "experiments": [
    {
      "experimentId": "global-exp-001",
      "description": "多主机性能测试",
      "createdAt": "2025-09-29T08:30:00Z",
      "timeout": 300,
      "collectionInterval": 1000,
      "participatingHosts": [
        {"name": "主机1", "ip": "192.168.1.100"}
      ]
    }
  ],
  "total": 1,
  "hasMore": false
}
```

### POST /api/experiments
创建全局实验

**请求:**
```json
{
  "experimentId": "global-exp-001",
  "description": "多主机性能测试",
  "timeout": 300,
  "collectionInterval": 1000,
  "participatingHosts": [
    {"name": "主机1", "ip": "192.168.1.100"},
    {"name": "主机2", "ip": "192.168.1.101"}
  ]
}
```

**响应:**
```json
{
  "experimentId": "global-exp-001",
  "description": "多主机性能测试",
  "createdAt": "2025-09-29T08:30:00Z",
  "timeout": 300,
  "collectionInterval": 1000,
  "participatingHosts": [
    {"name": "主机1", "ip": "192.168.1.100"},
    {"name": "主机2", "ip": "192.168.1.101"}
  ]
}
```

### GET /api/experiments/{experimentId}
获取全局实验详情

**响应:** 同创建实验响应格式

### GET /api/experiments/{experimentId}/data
获取全局实验数据

**查询参数:**
- `hostName`: 指定主机名（可选，不指定则返回所有主机摘要）

**响应:**
```json
{
  "experimentId": "global-exp-001",
  "experiment": {
    "experimentId": "global-exp-001",
    "description": "多主机性能测试",
    "createdAt": "2025-09-29T08:30:00Z",
    "participatingHosts": [
      {"name": "主机1", "ip": "192.168.1.100"}
    ]
  },
  "hosts": [
    {
      "name": "主机1",
      "ip": "192.168.1.100",
      "data": {
        "experimentId": "global-exp-001",
        "startTime": "2025-09-29T08:30:00Z",
        "endTime": "2025-09-29T08:35:00Z",
        "duration": 300,
        "metrics": [...]
      }
    }
  ]
}
```

### POST /api/experiments/{experimentId}/stop
停止全局实验并收集数据

**响应:**
```json
{
  "experimentId": "global-exp-001",
  "status": "success",
  "timestamp": "2025-09-29T08:35:00Z",
  "message": "Successfully stopped experiment and collected data from 2 hosts",
  "hostsCollected": [
    {"name": "主机1", "ip": "192.168.1.100"},
    {"name": "主机2", "ip": "192.168.1.101"}
  ],
  "hostsFailed": []
}
```

---

## 错误处理

### 标准错误响应格式

```json
{
  "error": "错误类型",
  "message": "详细错误信息",
  "timestamp": "2025-09-29T08:30:00Z"
}
```

### 常见HTTP状态码

- `200`: 成功
- `201`: 创建成功
- `400`: 请求参数错误
- `404`: 资源不存在
- `409`: 资源冲突（如实验ID重复）
- `500`: 服务器内部错误
- `503`: 服务不可用

### 错误示例

**实验ID重复:**
```json
{
  "error": "ExperimentExists",
  "message": "experiment exp-001 already exists",
  "timestamp": "2025-09-29T08:30:00Z"
}
```

**主机不存在:**
```json
{
  "error": "HostNotFound",
  "message": "host not found: 主机1",
  "timestamp": "2025-09-29T08:30:00Z"
}
```

---

## 使用示例

### 完整实验流程

```bash
# 1. 检查主机健康状态
curl http://localhost:9090/api/hosts/主机1/health

# 2. 创建全局实验
curl -X POST http://localhost:9090/api/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "test-001",
    "description": "性能基线测试",
    "timeout": 60,
    "collectionInterval": 1000,
    "participatingHosts": [
      {"name": "主机1", "ip": "192.168.1.100"}
    ]
  }'

# 3. 等待实验运行一段时间后停止
curl -X POST http://localhost:9090/api/experiments/test-001/stop

# 4. 获取实验数据
curl http://localhost:9090/api/experiments/test-001/data
```

### 并发测试

```bash
# 多个主机并发执行计算测试
for i in {1..10}; do
  curl -X POST http://localhost:80/calculate &
done
wait
```

---

## 速率限制

- CPU仿真服务: 无限制（受硬件性能限制）
- 指标收集器: 无限制
- 管理仪表盘: 无限制

## 认证授权

当前版本未实现认证授权机制，所有API均为公开访问。生产环境建议：
- 使用防火墙限制访问IP
- 配置反向代理添加认证
- 使用VPN或内网部署

## 版本兼容性

当前API版本: v1.0.0
- 向后兼容承诺：小版本更新不破坏现有API
- 废弃字段将在新版本中标记并保留至少一个大版本
- 新增字段不影响现有客户端