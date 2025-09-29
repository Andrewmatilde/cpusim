# CPU仿真测试平台

一个完整的分布式CPU性能测试和监控平台，包含CPU计算服务、指标收集器、管理仪表盘和Web界面。

## 系统概述

该平台由三个核心服务组成：

- **CPU仿真服务 (cpusim-server)**: 执行高精度GCD计算，模拟CPU密集型负载
- **指标收集器 (collector-server)**: 实时监控系统指标和实验管理
- **管理仪表盘 (dashboard-server)**: 统一管理多主机实验和数据分析
- **Web前端界面**: 直观的可视化管理界面

## 核心特性

### 🧮 CPU仿真服务
- 基于300位大整数的GCD计算
- 稳定可预测的计算负载
- RESTful API接口
- 高并发处理能力

### 📊 指标收集器
- 实时系统性能监控（CPU、内存、网络）
- 实验生命周期管理
- 可配置的数据收集间隔
- JSON格式数据存储

### 🎛️ 管理仪表盘
- 多主机集中管理
- 分布式实验编排
- 实时健康状态监控
- 统一数据收集和分析

### 🌐 Web界面
- 现代化React前端
- 实时指标可视化
- 一键实验管理
- 响应式设计

## 项目结构

```text
cpusim/
├── api/                    # OpenAPI规范和代码生成配置
├── bin/                    # 编译输出目录
├── calculator/             # GCD计算核心模块
├── cmd/                    # 服务启动入口
│   ├── cpusim-server/      # CPU仿真服务
│   ├── collector-server/   # 指标收集器
│   └── dashboard-server/   # 管理仪表盘
├── collector/              # 收集器实现
├── dashboard/              # 仪表盘服务实现
├── configs/                # 配置文件目录
├── data/                   # 实验数据存储
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
# 编译CPU仿真服务
make build

# 编译指标收集器
make build-collector

# 编译管理仪表盘
make build-dashboard
```

### 快速启动

#### 1. 启动CPU仿真服务
```bash
# 需要root权限绑定80端口
make run
```

#### 2. 启动指标收集器
```bash
# 在新终端窗口中运行，监听8080端口
make run-collector
```

#### 3. 启动管理仪表盘
```bash
# 在新终端窗口中运行，监听9090端口
make run-dashboard
```

#### 4. 启动Web界面
```bash
cd web
npm install
npm run dev
# 访问 http://localhost:5173
```

### 一键部署（开发环境）

```bash
# 同时启动CPU服务和收集器
make run-both
```

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

### 指标收集器API (端口8080)

#### 健康检查
```bash
curl http://localhost:8080/health
```

#### 创建实验
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "test-exp-001",
    "description": "CPU负载测试",
    "timeout": 300,
    "collectionInterval": 1000
  }'
```

#### 停止实验并收集数据
```bash
curl -X POST http://localhost:8080/experiments/{experimentId}/stop
```

### 管理仪表盘API (端口9090)

#### 获取所有主机
```bash
curl http://localhost:9090/api/hosts
```

#### 创建全局实验
```bash
curl -X POST http://localhost:9090/api/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "global-test-001",
    "description": "多主机性能测试",
    "timeout": 300,
    "collectionInterval": 1000,
    "participatingHosts": [
      {"name": "host1", "ip": "192.168.1.100"}
    ]
  }'
```

## 架构设计

### 服务架构
```text
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web前端界面   │    │  管理仪表盘服务  │    │ 多个测试主机     │
│   (React)       │────│ (dashboard)     │────│ (各自运行cpu+   │
│   Port: 5173    │    │ Port: 9090      │    │ collector服务)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ├─── Host1: CPU(80) + Collector(8080)
                                ├─── Host2: CPU(80) + Collector(8080)
                                └─── Host3: CPU(80) + Collector(8080)
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
- **多主机协调**: 统一创建、启动、停止分布式实验
- **数据聚合**: 自动收集和合并各主机实验数据
- **故障处理**: 部分主机失败时的优雅降级

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
- `PORT`: 服务监听端口
- `DATA_DIR`: 数据存储目录
- `CONFIG_PATH`: 配置文件路径
- `STORAGE_PATH`: 实验数据存储路径
- `CALCULATOR_PROCESS_NAME`: CPU计算服务进程名

### 配置文件示例 (configs/config.json)
```json
{
  "hosts": [
    {
      "name": "host1",
      "ip": "192.168.1.100",
      "cpuPort": 80,
      "collectorPort": 8080
    }
  ]
}
```

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
