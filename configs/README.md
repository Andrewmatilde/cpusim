# Dashboard Configuration

## Local Dashboard Setup

Dashboard 服务在本地运行，需要手动配置连接到远程服务。

### 1. 创建配置文件

```bash
cp dashboard-config.example.json dashboard-config.json
```

### 2. 编辑配置文件

编辑 `dashboard-config.json`，填入你的远程服务器信息：

```json
{
  "target_hosts": [
    {
      "name": "target-1",
      "external_ip": "YOUR_TARGET_IP",
      "internal_ip": "YOUR_TARGET_IP",
      "cpu_service_url": "http://YOUR_TARGET_IP:80",
      "collector_service_url": "http://YOUR_TARGET_IP:8080"
    }
  ],
  "client_host": {
    "name": "client-1",
    "external_ip": "YOUR_CLIENT_IP",
    "internal_ip": "YOUR_CLIENT_IP",
    "requester_service_url": "http://YOUR_CLIENT_IP:80"
  }
}
```

### 3. 启动 Dashboard

```bash
PORT=9090 CONFIG_PATH=./configs/dashboard-config.json ./bin/dashboard-server
```

### 4. 验证

```bash
# 检查配置
curl http://localhost:9090/config

# 检查状态
curl http://localhost:9090/status

# 启动实验
curl -X POST http://localhost:9090/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experimentId": "test-001",
    "timeout": 60
  }'
```

## 配置说明

### target_hosts
目标主机列表，每个主机需要运行：
- **cpusim-server** (端口 80): CPU 计算服务
- **collector-server** (端口 8080): 指标收集服务

### client_host
客户端主机，需要运行：
- **requester-server** (端口 80): 请求发送服务

### 注意事项

1. **IP 地址**: 使用可从本地访问的公网 IP 或内网 IP
2. **防火墙**: 确保端口 80, 8080 在远程服务器上开放
3. **服务状态**: 使用 Ansible 部署后，服务应该自动启动
4. **多目标**: 可以添加多个 target_hosts，Dashboard 会协调所有目标

## 示例：生产环境配置（多目标）

```json
{
  "target_hosts": [
    {
      "name": "prod-server-1",
      "external_ip": "203.0.113.10",
      "internal_ip": "10.0.1.10",
      "cpu_service_url": "http://203.0.113.10:80",
      "collector_service_url": "http://203.0.113.10:8080"
    },
    {
      "name": "prod-server-2",
      "external_ip": "203.0.113.11",
      "internal_ip": "10.0.1.11",
      "cpu_service_url": "http://203.0.113.11:80",
      "collector_service_url": "http://203.0.113.11:8080"
    }
  ],
  "client_host": {
    "name": "prod-client",
    "external_ip": "203.0.113.20",
    "internal_ip": "10.0.2.10",
    "requester_service_url": "http://203.0.113.20:80"
  }
}
```
