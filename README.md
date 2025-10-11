# CPU仿真测试平台

一个完整的分布式CPU性能测试和监控平台，包含CPU计算服务、指标收集器、请求发送器、管理仪表盘的端到端实验系统。

## 系统概述

该平台由四个核心服务组成：

- **CPU仿真服务 (cpusim-server)**: 执行高精度GCD计算，模拟CPU密集型负载
- **指标收集器 (collector-server)**: 实时监控目标主机系统指标和实验管理
- **请求发送器 (requester-server)**: 生成HTTP负载，向目标服务发送请求并收集响应统计
- **管理仪表盘 (dashboard-server)**: 控制平面，协调分布式实验的启动、停止和数据收集

## 核心特性

### 🧮 CPU仿真服务
- 基于300位大整数的GCD计算
- 稳定可预测的计算负载
- RESTful API接口
- 高并发处理能力

### 📊 指标收集器
- 实时系统性能监控（CPU、内存、网络）
- 进程级别监控（目标计算服务）
- 实验生命周期管理（Pending/Running状态机）
- 可配置的数据收集间隔
- JSON格式数据持久化

### 🚀 请求发送器
- 可配置QPS的HTTP负载生成
- 目标服务请求统计（成功/失败/响应时间）
- 支持超时控制
- 实验生命周期管理
- 请求结果数据收集

### 🎛️ 管理仪表盘
- 分布式实验协调编排
- 多目标主机 collector 管理
- 统一 requester 控制
- 两阶段实验启动（collectors → requester）
- 自动错误回滚和清理
- 聚合实验数据收集

## 项目结构

```text
cpusim/
├── api/                    # OpenAPI规范（collector/dashboard/requester）
├── bin/                    # 编译输出目录
├── calculator/             # GCD计算核心模块
├── cmd/                    # 服务启动入口
│   ├── cpusim-server/      # CPU仿真服务
│   ├── collector-server/   # 指标收集器
│   ├── requester-server/   # 请求发送器
│   └── dashboard-server/   # 管理仪表盘
├── pkg/                    # 共享包
│   ├── exp/                # 统一实验管理框架（泛型）
│   ├── collector/          # Collector服务实现
│   ├── requester/          # Requester服务实现
│   └── dashboard/          # Dashboard服务实现
├── collector/api/          # Collector生成的API代码
├── requester/api/          # Requester生成的API代码
├── dashboard/api/          # Dashboard生成的API代码
├── ansible/                # Ansible部署脚本
├── configs/                # 配置文件目录
├── data/                   # 实验数据存储目录
├── web/                    # React前端应用
├── go.mod                  # Go模块定义
├── Makefile                # 构建脚本
└── README.md               # 本文档
```

## 快速开始

### 系统要求

- Go 1.25+
- Node.js 18+ (前端开发)
- Linux/macOS/Windows
- 至少2GB内存

### 编译所有服务

```bash
# 编译所有服务
go build -o bin/cpusim-server ./cmd/cpusim-server
go build -o bin/collector-server ./cmd/collector-server
go build -o bin/requester-server ./cmd/requester-server
go build -o bin/dashboard-server ./cmd/dashboard-server
```

### 本地开发快速启动

#### 目标主机（运行 cpusim + collector）
```bash
# 终端1: 启动CPU仿真服务（端口80）
sudo ./bin/cpusim-server

# 终端2: 启动指标收集器（端口8080）
STORAGE_PATH=./data/experiments PORT=8080 CALCULATOR_PROCESS_NAME=cpusim-server ./bin/collector-server
```

#### 客户端主机（运行 requester）
```bash
# 终端3: 启动请求发送器（端口8081）
PORT=8081 REQUESTER_DATA_DIR=./data/requester ./bin/requester-server
```

#### 控制节点（运行 dashboard）
```bash
# 终端4: 启动管理仪表盘（端口9090）
# 需要先创建配置文件 configs/config.json
PORT=9090 CONFIG_PATH=./configs/config.json ./bin/dashboard-server
```

### 配置文件示例 (configs/config.json)
```json
{
  "target_hosts": [
    {
      "name": "local-target",
      "external_ip": "127.0.0.1",
      "internal_ip": "127.0.0.1",
      "cpu_service_url": "http://127.0.0.1:80",
      "collector_service_url": "http://127.0.0.1:8080"
    }
  ],
  "client_host": {
    "name": "local-client",
    "external_ip": "127.0.0.1",
    "internal_ip": "127.0.0.1",
    "requester_service_url": "http://127.0.0.1:8081"
  }
}
```

**注意**: Dashboard 配置只包含服务 URL。各服务的运行参数（如 collection interval, QPS 等）通过环境变量在各自服务启动时配置。

## API文档

### CPU仿真服务 API (端口80)

#### 计算接口
**端点:** `POST /calculate`

```bash
curl -X POST http://localhost:80/calculate \
  -H "Content-Type: application/json" \
  -d '{}'
```

**响应:**
```json
{
  "gcd": "1",
  "process_time": "2.356ms"
}
```

### 指标收集器 API (端口8080)

#### 健康检查
```bash
curl http://localhost:8080/health
```

#### 获取状态
```bash
curl http://localhost:8080/status
# 返回: {"status": "Pending"} 或 {"status": "Running"}
```

#### 开始收集实验
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "collector-exp-001",
    "timeout": 60
  }'
```

#### 停止实验
```bash
curl -X POST http://localhost:8080/experiments/collector-exp-001/stop
```

#### 获取实验数据
```bash
curl http://localhost:8080/experiments/collector-exp-001
```

### 请求发送器 API (端口8081)

#### 健康检查
```bash
curl http://localhost:8081/health
```

#### 获取状态
```bash
curl http://localhost:8081/status
```

#### 开始请求实验
```bash
curl -X POST http://localhost:8081/experiments/request \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "requester-exp-001",
    "timeout": 60
  }'
```

#### 停止实验
```bash
curl -X POST http://localhost:8081/experiments/request/requester-exp-001/stop
```

#### 获取实验统计
```bash
curl http://localhost:8081/experiments/request/requester-exp-001
```

### 管理仪表盘 API (端口9090)

#### 健康检查
```bash
curl http://localhost:9090/health
```

#### 获取服务配置
```bash
curl http://localhost:9090/config
```

#### 获取状态
```bash
curl http://localhost:9090/status
```

#### 开始分布式实验（协调 collectors + requester）
```bash
curl -X POST http://localhost:9090/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "dashboard-exp-001",
    "timeout": 60
  }'
```

#### 停止分布式实验
```bash
curl -X POST http://localhost:9090/experiments/dashboard-exp-001/stop
```

#### 获取聚合实验数据
```bash
curl http://localhost:9090/experiments/dashboard-exp-001
```

## 架构设计

### 服务架构
```text
┌──────────────────────┐
│   控制平面           │
│   Dashboard Server   │  协调分布式实验
│   (Port 9090)        │  - 启动/停止 collectors
└──────┬───────────────┘  - 启动/停止 requester
       │                  - 聚合数据收集
       │
       ├────────────────────┬─────────────────────┐
       ↓                    ↓                     ↓
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ Target Host1│      │ Target Host2│      │ Client Host │
├─────────────┤      ├─────────────┤      ├─────────────┤
│ cpusim (80) │      │ cpusim (80) │      │ requester   │
│ collector   │      │ collector   │      │ (8081)      │
│ (8080)      │      │ (8080)      │      │             │
└─────────────┘      └─────────────┘      └──────┬──────┘
       ↑                    ↑                     │
       └────────────────────┴─────────────────────┘
            Requester 向 Target Hosts 发送 HTTP 请求
```

### 核心技术

#### CPU仿真算法
- **固定大整数GCD计算**: 使用两个预定义的300位斐波那契数
- **欧几里得算法**: 通过Go的`math/big`包处理大整数运算
- **稳定负载**: 每次计算耗时2-5ms，负载可预测

#### 指标收集
- **系统指标**: CPU使用率、内存使用量、网络I/O统计
- **服务监控**: 目标计算服务健康状态检测
- **数据存储**: JSON格式的时序数据持久化

#### 分布式实验管理
- **统一实验框架**: 基于 Go 泛型的 `pkg/exp` 框架，所有服务共享相同的实验生命周期
- **两状态机**: Pending/Running 状态，严格的状态转换规则
- **多阶段协调**: Dashboard 先启动 collectors，再启动 requester
- **自动回滚**: 任何子实验失败时自动停止所有已启动的服务
- **数据聚合**: 自动收集和合并各主机实验数据

## 数据格式

### 实验数据结构
```json
{
  "experiment": {
    "experimentId": "exp-1759135859485",
    "description": "多主机CPU负载测试",
    "collectionInterval": 1000,
    "timeout": 300,
    "participatingHosts": [
      {"name": "host1", "ip": "192.168.1.100"}
    ],
    "createdAt": "2025-09-29T16:51:06+08:00"
  },
  "hosts": [
    {
      "name": "host1",
      "ip": "192.168.1.100",
      "data": {
        "metrics": [
          {
            "timestamp": "2025-09-29T16:51:07+08:00",
            "systemMetrics": {
              "cpuUsagePercent": 1.98,
              "memoryUsageBytes": 115986432,
              "memoryUsagePercent": 11.50,
              "calculatorServiceHealthy": true,
              "networkIOBytes": {
                "bytesReceived": 63476,
                "bytesSent": 63681,
                "packetsReceived": 640,
                "packetsSent": 432
              }
            }
          }
        ]
      }
    }
  ]
}
```

## Makefile命令参考

| 命令 | 说明 |
|------|------|
| `make build` | 编译CPU仿真服务 |
| `make build-collector` | 编译指标收集器 |
| `make build-dashboard` | 编译管理仪表盘 |
| `make run` | 运行CPU仿真服务 |
| `make run-collector` | 运行指标收集器 |
| `make run-dashboard` | 运行管理仪表盘 |
| `make run-both` | 同时运行CPU服务和收集器 |
| `make test-collector` | 测试收集器API |
| `make clean` | 清理所有编译产物 |
| `make help` | 显示所有可用命令 |

## 配置管理

### 环境变量

**Collector Server:**
- `PORT`: 服务监听端口 (默认: 8080)
- `STORAGE_PATH`: 实验数据存储路径
- `CALCULATOR_PROCESS_NAME`: CPU计算服务进程名 (用于监控)

**Requester Server:**
- `PORT`: 服务监听端口 (默认: 8081)
- `REQUESTER_DATA_DIR`: 数据存储目录
- `TARGET_HOST`: 目标服务器地址
- `TARGET_PORT`: 目标服务器端口
- `QPS`: 每秒请求数
- `TIMEOUT`: 请求超时时间(秒)

**Dashboard Server:**
- `PORT`: 服务监听端口 (默认: 9090)
- `CONFIG_PATH`: 配置文件路径 (必需)
- `STORAGE_PATH`: 实验数据存储路径

## 性能指标

### CPU仿真服务
- **计算延迟**: 2-5ms per request
- **内存占用**: < 10MB
- **并发处理**: 支持高并发请求
- **吞吐量**: > 1000 requests/second

### 指标收集器
- **采样间隔**: 可配置 (推荐1000ms)
- **数据准确度**: 毫秒级时间戳
- **存储格式**: JSON格式，易于分析
- **资源开销**: 低CPU和内存占用

## 使用场景

- 🧪 **CPU性能基准测试**: 标准化的计算负载评估
- ⚖️ **负载均衡验证**: 多主机负载分布分析
- 🐳 **容器性能评估**: Kubernetes环境压力测试
- 🔍 **微服务监控**: 分布式系统性能分析
- 📈 **容量规划**: 基于历史数据的资源规划

## 故障排查

### 常见问题

1. **端口占用错误**
   ```bash
   sudo lsof -i :80  # 检查80端口占用
   sudo lsof -i :8080  # 检查8080端口占用
   ```

2. **权限不足**
   ```bash
   sudo make run  # CPU服务需要root权限绑定80端口
   ```

3. **配置文件错误**
   - 检查 `configs/config.json` 文件格式
   - 确保主机IP地址可达

4. **数据目录权限**
   ```bash
   mkdir -p data/experiments
   chmod 755 data/experiments
   ```

## 开发指南

### 代码生成
```bash
# 重新生成OpenAPI客户端代码
go generate ./...
```

### 添加新主机
1. 在目标主机部署CPU仿真服务和收集器
2. 更新 `configs/config.json` 配置文件
3. 重启管理仪表盘服务

## 许可证

MIT License - 详见LICENSE文件
