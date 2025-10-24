# Infrastructure Reference Documentation

## Network Architecture

```text
┌─────────────────────┐
│   Load Balancer     │
│ (internal IP only)  │
└──────────┬──────────┘
           │
    ┌──────┴──────┬──────────┬─────────┐
    │             │          │         │
┌───▼────┐   ┌───▼────┐ ┌──▼─────┐  (More targets...)
│Target-1│   │Target-2│ │Target-N│
├────────┤   ├────────┤ ├────────┤
│ cpusim │   │ cpusim │ │ cpusim │
│collector   │collector │collector
└────────┘   └────────┘ └────────┘

┌──────────────┐       ┌────────────────┐
│  Client Host │       │ Dashboard Host │
├──────────────┤       ├────────────────┤
│  requester   │       │  dashboard     │
│              │       │  frontend      │
└──────────────┘       └────────────────┘
```

**Important**: The requester service sends requests to the Load Balancer's internal IP (defined in `inventory.yml` as `load_balancer_internal_ip`), which distributes traffic to target hosts.

## Host and Service Mapping

Use `ansible/inventory.yml` to find actual host IPs. The structure is:

| Service | Host Group | Port | Path |
|---------|------------|------|------|
| cpusim-server | target_hosts | 80 | /calculate |
| collector-server | target_hosts | 8080 | /health |
| requester-server | client_hosts | 80 | /health |
| dashboard-server | dashboard_hosts | 9090 | /health, /api/* |
| dashboard-frontend | dashboard_hosts | 8080 | / |
| load-balancer | (internal only) | 80 | /calculate |

**Note**: Multiple target hosts can exist (e.g., cpusim-cloud-1 through cpusim-cloud-N). Each runs both cpusim-server and collector-server.

## Ansible Configuration

### Inventory Structure

**File**: `ansible/inventory.yml`

```yaml
all:
  children:
    target_hosts:
      hosts:
        cpusim-cloud:
          ansible_host: 128.1.40.151
    client_hosts:
      hosts:
        cpusim-client:
          ansible_host: 45.43.63.88
    dashboard_hosts:
      hosts:
        cpusim-dashboard:
          ansible_host: 118.194.234.132
  vars:
    load_balancer_internal_ip: "10.35.55.87"
    release_version: v0.15.0  # Update this for each release
```

### Playbook Files

- `deploy.yml`: Deploys cpusim-server and collector-server to target hosts
- `deploy-requester.yml`: Deploys requester-server to client hosts
- `deploy-dashboard.yml`: Deploys dashboard-server and frontend to dashboard hosts

### Template Files

**`ansible/templates/requester.env.j2`** - Requester service environment:

```bash
PORT={{ requester_port }}
TARGET_IP={{ load_balancer_internal_ip }}  # Points to LB, not target directly
TARGET_PORT={{ service_port }}
QPS=10
TIMEOUT=30
STORAGE_PATH={{ requester_home }}/data
```

## Service Configuration

### Dashboard Configuration

**File**: `configs/config.json`

```json
{
  "target_hosts": [...],
  "client_host": {...},
  "load_balancer": {
    "name": "lb-1",
    "external_ip": "",
    "internal_ip": "10.35.55.87"
  }
}
```

### Environment Variables

#### cpusim-server
- `PORT`: Service port (default: 80)

#### collector-server
- `PORT`: Service port (default: 8080)
- `STORAGE_PATH`: Data storage directory
- `CALCULATOR_PROCESS_NAME`: Process name to monitor (cpusim-server)

#### requester-server
- `PORT`: Service port (default: 80)
- `TARGET_IP`: Load balancer internal IP (10.35.55.87)
- `TARGET_PORT`: Target service port (80)
- `QPS`: Requests per second (default: 10)
- `TIMEOUT`: Request timeout in seconds (default: 30)
- `STORAGE_PATH`: Data storage directory

#### dashboard-server
- `PORT`: Service port (default: 9090)
- `CONFIG_PATH`: Path to config.json file

## Service File Locations

On deployed hosts:

- **Binaries**: `/opt/{service}/bin/`
- **Data**: `/opt/{service}/data/`
- **Logs**: `/opt/{service}/logs/`
- **Environment**: `/opt/{service}/{service}.env`
- **Systemd**: `/etc/systemd/system/{service}.service`

## GitHub Actions Workflow

**File**: `.github/workflows/release.yml`

This workflow is triggered when a tag is pushed (pattern: `v*`).

### Build Matrix

The workflow builds binaries for:
- linux-amd64
- linux-arm64
- darwin-amd64
- darwin-arm64

### Artifacts Created

For each release `v0.X.X`:

- `cpusim-server-v0.X.X-linux-amd64`
- `cpusim-server-v0.X.X-linux-arm64`
- `cpusim-server-v0.X.X-darwin-amd64`
- `cpusim-server-v0.X.X-darwin-arm64`
- `collector-server-v0.X.X-linux-amd64`
- `collector-server-v0.X.X-linux-arm64`
- `collector-server-v0.X.X-darwin-amd64`
- `collector-server-v0.X.X-darwin-arm64`
- `requester-server-v0.X.X-linux-amd64`
- `requester-server-v0.X.X-linux-arm64`
- `requester-server-v0.X.X-darwin-amd64`
- `requester-server-v0.X.X-darwin-arm64`
- `dashboard-server-v0.X.X-linux-amd64`
- `dashboard-server-v0.X.X-linux-arm64`
- `dashboard-server-v0.X.X-darwin-amd64`
- `dashboard-server-v0.X.X-darwin-arm64`
- `dashboard-frontend-v0.X.X.tar.gz`

## Useful Ansible Commands

### Testing and Diagnostics

```bash
# Test Ansible connectivity to all hosts
ansible all -m ping

# Check service status on all hosts
ansible all -m shell -a "systemctl status cpusim-server" -b

# View Ansible facts for a host
ansible {host} -m setup

# Run playbook with verbose output
ansible-playbook deploy.yml -vvv
```

### Selective Deployment

```bash
# Deploy specific components using tags
ansible-playbook deploy.yml --tags "cpusim"
ansible-playbook deploy.yml --tags "collector"

# Deploy to specific host groups
ansible-playbook deploy.yml --limit target_hosts
ansible-playbook deploy.yml --limit client_hosts
```

## Advanced Troubleshooting

### Port Conflicts

If a service fails with "port already in use":

```bash
# Find process using the port
lsof -i :{port}

# Stop the conflicting service
systemctl stop {service-name}

# Or kill the process directly
kill -9 {pid}
```

### Service Logs Analysis

```bash
# View all logs for a service
journalctl -u {service-name} --no-pager

# Filter logs by time
journalctl -u {service-name} --since "2024-01-01" --until "2024-01-02"

# Filter logs by priority (error, warning, etc.)
journalctl -u {service-name} -p err

# Export logs to file
journalctl -u {service-name} > /tmp/{service}-logs.txt
```

### Network Connectivity Testing

```bash
# Test connectivity from requester to load balancer
ssh -i ~/.ssh/andrew.pem ubuntu@45.43.63.88
curl -v http://10.35.55.87:80/calculate

# Test connectivity from target to services
ssh -i ~/.ssh/andrew.pem ubuntu@128.1.40.151
curl -v http://localhost:80/calculate
curl -v http://localhost:8080/health
```

### Binary Verification

```bash
# Check binary version and architecture
file /opt/{service}/bin/{service}

# Verify binary is executable
ls -la /opt/{service}/bin/{service}
chmod +x /opt/{service}/bin/{service}

# Test binary execution
/opt/{service}/bin/{service} --version
```

## Automation Examples

### Automated Deployment Script

```bash
#!/bin/bash
# Complete release and deployment automation

VERSION=$1

if [ -z "$VERSION" ]; then
  echo "Usage: ./auto-deploy.sh v0.X.X"
  exit 1
fi

# 1. Create release
git tag $VERSION
git push origin $VERSION

# 2. Wait for GitHub Actions
echo "Waiting for GitHub Actions..."
gh run watch

# 3. Verify release
gh release view $VERSION

# 4. Update inventory
sed -i '' "s/release_version: v.*/release_version: $VERSION/" ansible/inventory.yml

# 5. Deploy
cd ansible && ./deploy.sh

# 6. Verify deployment
echo "Testing endpoints..."
curl -s http://128.1.40.151:80/calculate
curl -s http://128.1.40.151:8080/health
curl -s http://45.43.63.88:80/health
curl -s http://118.194.234.132:9090/health

echo "Deployment complete!"
```

### Health Monitoring Script

```bash
#!/bin/bash
# Monitor all services health

SERVICES=("cpusim-server" "collector-server" "requester-server" "dashboard-server")

for service in "${SERVICES[@]}"; do
  echo "=== $service ==="
  status=$(ansible all -m shell -a "systemctl is-active $service 2>/dev/null" -b 2>&1 | grep -E "active|inactive|failed")
  echo "$status"
  echo ""
done
```
