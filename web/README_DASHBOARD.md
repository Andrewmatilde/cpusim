# CPU仿真测试平台 - Web管理界面

## 概览
这是CPU仿真测试平台的现代化Web管理界面。提供直观的可视化操作界面，用于管理和监控分布式CPU仿真主机、运行性能实验和进行计算测试。

## 核心功能
- 🖥️ **实时主机监控**: 多主机健康状态和性能指标实时显示
- 🧪 **实验管理**: 可视化创建、启动、停止分布式实验
- 📊 **性能测试**: 一键执行CPU计算负载测试
- 📈 **指标可视化**: CPU、内存、网络实时图表展示
- 🔄 **自动刷新**: 每10秒自动更新数据，保持数据实时性
- 🎨 **现代UI**: 基于shadcn/ui的响应式设计
- 🌐 **多主机支持**: 统一管理多个测试节点

## 环境准备

### 1. 启动后端管理服务
管理仪表盘服务需要运行在9090端口。从项目根目录执行：

```bash
# 方式1: 直接运行Go程序
cd cmd/dashboard-server
go run main.go

# 方式2: 使用编译后的二进制文件
make build-dashboard
./bin/dashboard-server

# 方式3: 使用Makefile快捷命令
make run-dashboard
```

后端API应该可以通过以下地址访问: http://localhost:9090

### 2. 确保CPU仿真服务运行
管理界面连接的每个测试主机都需要运行：
- **CPU仿真服务** (端口80): 执行GCD计算任务
- **指标收集器** (端口8080): 收集系统性能指标

### 3. 配置主机信息
在项目根目录创建配置文件 `configs/config.json`：

```json
{
  "hosts": [
    {
      "name": "主机1",
      "ip": "192.168.1.100",
      "cpuPort": 80,
      "collectorPort": 8080
    },
    {
      "name": "主机2",
      "ip": "192.168.1.101",
      "cpuPort": 80,
      "collectorPort": 8080
    }
  ]
}
```

## 运行Web界面

### 安装依赖
首次运行需要安装前端依赖：
```bash
cd web
npm install
```

### 开发模式
```bash
npm run dev
```
开发服务器将启动在: http://localhost:5173

### 生产构建
```bash
npm run build
npm run preview
```

生产版本预览服务器将启动在: http://localhost:4173

## 项目结构
```
src/
├── api/
│   ├── client.ts             # API客户端实现
│   └── types.ts              # TypeScript类型定义
├── components/
│   ├── Dashboard.tsx         # 主仪表盘组件
│   ├── HostCard.tsx          # 单主机展示卡片
│   ├── ExperimentManager.tsx # 实验管理组件
│   ├── CalculationTest.tsx   # CPU计算测试组件
│   └── ui/                   # shadcn/ui基础组件
├── hooks/
│   └── useHosts.ts           # 主机数据React Hook
└── lib/
    └── utils.ts              # 工具函数
```

## API接口说明
前端通过以下API与后端通信：

### 主机管理
- `GET /api/hosts` - 获取所有主机列表
- `GET /api/hosts/{name}/health` - 获取主机健康状态
- `POST /api/hosts/{name}/calculate` - 执行CPU计算测试

### 全局实验管理
- `GET /api/experiments` - 获取所有实验列表
- `POST /api/experiments` - 创建新的全局实验
- `GET /api/experiments/{id}` - 获取实验详情
- `GET /api/experiments/{id}/data` - 获取实验数据
- `POST /api/experiments/{id}/stop` - 停止实验并收集数据

### 单主机实验管理
- `GET /api/hosts/{name}/experiments` - 获取主机实验列表
- `POST /api/hosts/{name}/experiments` - 在主机上启动实验
- `GET /api/hosts/{name}/experiments/{id}/status` - 获取实验状态

## 界面功能详解

### 主仪表盘
- **主机概览**: 显示所有配置主机的在线状态和基本信息
- **健康监控**: 实时显示CPU服务和收集器服务的健康状态
- **快速测试**: 一键执行GCD计算性能测试
- **实验列表**: 查看所有历史和当前运行的实验

### 实验管理
- **创建实验**: 设置实验ID、描述、超时时间和采集间隔
- **选择主机**: 多选参与实验的目标主机
- **实时监控**: 查看实验进度和各主机状态
- **数据收集**: 一键停止实验并自动收集所有主机数据

### 数据可视化
- **指标图表**: CPU使用率、内存占用、网络I/O趋势图
- **实时更新**: 10秒自动刷新，保持数据最新
- **历史数据**: 查看已完成实验的详细指标数据

## 故障排查

### "无法连接后端API" 错误
```bash
# 1. 确认管理仪表盘服务正在运行 (端口9090)
curl http://localhost:9090/api/hosts

# 2. 检查端口占用情况
sudo lsof -i :9090

# 3. 重新启动后端服务
make run-dashboard
```

### 主机列表为空
1. **检查配置文件**: 确保 `configs/config.json` 存在且格式正确
2. **验证主机连通性**: ping配置的主机IP地址
3. **检查服务状态**: 确认目标主机的CPU服务和收集器正在运行
4. **查看后端日志**: 检查是否有连接错误信息

### 实验创建失败
1. **检查主机服务**: 确保所选主机的收集器服务(8080端口)可访问
2. **验证实验ID**: 确保实验ID唯一，不与现有实验冲突
3. **检查权限**: 确认后端服务有写入data目录的权限

### 数据显示异常
1. **清除浏览器缓存**: 刷新浏览器或清除缓存
2. **检查数据文件**: 验证 `data/` 目录下的实验数据文件完整性
3. **重启服务**: 依次重启collector和dashboard服务

### 构建错误
```bash
# 清理依赖并重新安装
rm -rf node_modules package-lock.json
npm install

# 清理Vite缓存
rm -rf .vite
npm run dev
```

## 技术栈

### 前端框架
- **React 19**: 现代化React框架，支持最新特性
- **TypeScript**: 类型安全的JavaScript开发
- **Vite**: 快速的构建工具和开发服务器

### UI组件
- **Tailwind CSS**: 实用优先的CSS框架
- **shadcn/ui**: 现代化组件库，基于Radix UI
- **Lucide React**: 美观的SVG图标库

### 状态管理
- **React Hooks**: 使用内置Hooks管理组件状态
- **Custom Hooks**: 自定义Hooks封装业务逻辑

### 通信协议
- **REST API**: 标准HTTP接口与后端通信
- **JSON**: 数据交换格式

## 开发指南

### 添加新功能
1. 在 `src/components/` 下创建新组件
2. 更新API类型定义 (`src/api/types.ts`)
3. 添加相应的API调用方法 (`src/api/client.ts`)
4. 在主Dashboard组件中集成新功能

### 自定义主题
修改 `tailwind.config.js` 文件中的颜色配置：
```javascript
module.exports = {
  theme: {
    extend: {
      colors: {
        // 自定义颜色
      }
    }
  }
}
```

### 部署到生产环境
```bash
# 构建生产版本
npm run build

# 使用静态文件服务器部署
# 例如使用nginx指向dist目录
```