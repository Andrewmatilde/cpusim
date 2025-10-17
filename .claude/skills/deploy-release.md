# CPU Simulation Release and Deployment Skill

This skill helps with building, releasing, and deploying the CPU simulation distributed system.

## System Overview

The CPU simulation system consists of four main components:
1. **cpusim-server**: CPU simulation service (GCD calculator)
2. **collector-server**: Metrics collection service
3. **requester-server**: Request generator service
4. **dashboard-server**: Management backend API
5. **dashboard-frontend**: Web UI for management

### Infrastructure Architecture

```
┌─────────────────┐
│  Load Balancer  │
│  10.35.55.87    │
└────────┬────────┘
         │
    ┌────┴─────┐
    │          │
┌───▼──────┐   │
│ Target-1 │   │ (More targets...)
│ 128.1.x  │   │
├──────────┤   │
│ cpusim   │◄──┘
│ collector│
└──────────┘

┌─────────────┐       ┌──────────────┐
│  Client-1   │       │ Dashboard-1  │
│  45.43.x    │       │ 118.194.x    │
├─────────────┤       ├──────────────┤
│ requester   │       │ dashboard    │
│             │       │ frontend     │
└─────────────┘       └──────────────┘
```

**Important**: The requester service sends requests to the Load Balancer's internal IP (10.35.55.87), which distributes traffic to target hosts.

## Release Process

### 1. Create a GitHub Release

The release process is automated via GitHub Actions:

```bash
# Create and push a new tag
git tag v0.X.X
git push origin v0.X.X
```

This triggers the `.github/workflows/release.yml` workflow which:
- Builds all four server binaries for multiple platforms (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64)
- Builds and packages the dashboard frontend
- Creates a GitHub Release with all artifacts
- Uploads all binaries and frontend package to the release

**Artifacts created:**
- `cpusim-server-v0.X.X-linux-amd64`
- `collector-server-v0.X.X-linux-amd64`
- `requester-server-v0.X.X-linux-amd64`
- `dashboard-server-v0.X.X-linux-amd64`
- `dashboard-frontend-v0.X.X.tar.gz`

### 2. Manual Release Trigger

You can also manually trigger the release workflow:

```bash
# Go to GitHub Actions
# Select "Build and Release" workflow
# Click "Run workflow"
```

## Deployment Process

### Prerequisites

1. **Ansible installed**: `brew install ansible` (macOS)
2. **SSH Key**: `~/.ssh/andrew.pem` with correct permissions (600)
3. **Inventory configured**: `ansible/inventory.yml`

### Configuration Files

**`ansible/inventory.yml`** - Defines all hosts and variables:
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

**`configs/config.json`** - Dashboard service configuration:
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

### Deployment Commands

#### Deploy All Services

From the `ansible/` directory:

```bash
cd ansible
./deploy.sh
```

This script:
1. Checks prerequisites (Ansible, SSH key)
2. Tests server connectivity
3. Deploys target hosts (cpusim-server + collector-server)
4. Deploys client host (requester-server)
5. Deploys dashboard host (dashboard-server + frontend)

#### Deploy Individual Services

**Target hosts (cpusim + collector):**
```bash
cd ansible
ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook deploy.yml -v
```

**Client host (requester):**
```bash
cd ansible
ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook deploy-requester.yml -v
```

**Dashboard (backend + frontend):**
```bash
cd ansible
./deploy-dashboard.sh
# or
ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook deploy-dashboard.yml -v
```

### Service Configuration Templates

The Ansible deployment uses Jinja2 templates to configure services:

**`ansible/templates/requester.env.j2`** - Requester environment:
```bash
PORT={{ requester_port }}
TARGET_IP={{ load_balancer_internal_ip }}  # Points to LB, not target directly
TARGET_PORT={{ service_port }}
QPS=10
TIMEOUT=30
STORAGE_PATH={{ requester_home }}/data
```

### Deployment Workflow

1. **Update Release Version** in `ansible/inventory.yml`:
   ```yaml
   release_version: v0.16.0
   ```

2. **Run Deployment**:
   ```bash
   cd ansible
   ./deploy.sh
   ```

3. **Verify Services**:
   ```bash
   # Check all services
   ansible all -m shell -a "systemctl status cpusim-server" -b
   ansible all -m shell -a "systemctl status collector-server" -b
   ansible all -m shell -a "systemctl status requester-server" -b
   ansible all -m shell -a "systemctl status dashboard-server" -b
   ```

4. **Test Endpoints**:
   ```bash
   # CPU simulation service
   curl -X POST http://128.1.40.151:80/calculate -H 'Content-Type: application/json' -d '{}'

   # Collector service
   curl http://128.1.40.151:8080/health

   # Requester service
   curl http://45.43.63.88:80/health

   # Dashboard backend
   curl http://118.194.234.132:9090/health

   # Dashboard frontend
   curl http://118.194.234.132:8080
   ```

## Service Management

### Systemd Service Commands

```bash
# Start/Stop/Restart services
sudo systemctl start cpusim-server
sudo systemctl stop cpusim-server
sudo systemctl restart cpusim-server

# View service status
sudo systemctl status cpusim-server

# View logs
sudo journalctl -u cpusim-server -f
sudo journalctl -u cpusim-server --since "10 minutes ago"
```

### Service Files Location

- **Binaries**: `/opt/{service}/bin/`
- **Data**: `/opt/{service}/data/`
- **Logs**: `/opt/{service}/logs/`
- **Environment**: `/opt/{service}/{service}.env`
- **Systemd**: `/etc/systemd/system/{service}.service`

## Troubleshooting

### Common Issues

1. **Deployment fails with "Connection refused"**
   - Check SSH key permissions: `chmod 600 ~/.ssh/andrew.pem`
   - Test connectivity: `ansible all -m ping`

2. **Service fails to start**
   - Check logs: `journalctl -u {service-name} -n 50`
   - Verify binary permissions: `ls -la /opt/{service}/bin/`
   - Check environment file: `cat /opt/{service}/{service}.env`

3. **Port already in use**
   - Check running processes: `lsof -i :{port}`
   - Stop conflicting service: `systemctl stop {service-name}`

4. **Binary download fails**
   - Verify GitHub release exists: `gh release view v0.X.X`
   - Check GitHub token: Ensure SIM_TOKEN secret is set

### Useful Commands

```bash
# Test Ansible connectivity
ansible all -m ping

# Check service status on all hosts
ansible all -m shell -a "systemctl status cpusim-server" -b

# Deploy specific components
ansible-playbook deploy.yml --tags "cpusim"
ansible-playbook deploy.yml --tags "collector"

# View Ansible facts
ansible {host} -m setup

# Run playbook with verbose output
ansible-playbook deploy.yml -vvv
```

## Release Checklist

When creating a new release:

1. ✅ Update version in code (if versioned)
2. ✅ Commit all changes
3. ✅ Create and push tag: `git tag v0.X.X && git push origin v0.X.X`
4. ✅ Wait for GitHub Actions to complete
5. ✅ Verify release artifacts on GitHub
6. ✅ Update `release_version` in `ansible/inventory.yml`
7. ✅ Run deployment: `cd ansible && ./deploy.sh`
8. ✅ Verify all services are running
9. ✅ Test all endpoints
10. ✅ Monitor logs for any issues

## Environment Variables

### cpusim-server
- `PORT`: Service port (default: 80)

### collector-server
- `PORT`: Service port (default: 8080)
- `STORAGE_PATH`: Data storage directory
- `CALCULATOR_PROCESS_NAME`: Process name to monitor (cpusim-server)

### requester-server
- `PORT`: Service port (default: 80)
- `TARGET_IP`: Load balancer internal IP (10.35.55.87)
- `TARGET_PORT`: Target service port (80)
- `QPS`: Requests per second (default: 10)
- `TIMEOUT`: Request timeout in seconds (default: 30)
- `STORAGE_PATH`: Data storage directory

### dashboard-server
- `PORT`: Service port (default: 9090)
- `CONFIG_PATH`: Path to config.json file

## Quick Reference

| Service | Host | Port | URL |
|---------|------|------|-----|
| cpusim-server | 128.1.40.151 | 80 | http://128.1.40.151:80 |
| collector-server | 128.1.40.151 | 8080 | http://128.1.40.151:8080 |
| requester-server | 45.43.63.88 | 80 | http://45.43.63.88:80 |
| dashboard-server | 118.194.234.132 | 9090 | http://118.194.234.132:9090 |
| dashboard-frontend | 118.194.234.132 | 8080 | http://118.194.234.132:8080 |
| load-balancer | 10.35.55.87 | 80 | http://10.35.55.87:80 (internal) |

## Task Automation Examples

### Create a release and deploy

```bash
# 1. Create release
git tag v0.16.0
git push origin v0.16.0

# 2. Wait for GitHub Actions (or check status)
gh run list --workflow=release.yml

# 3. Update inventory
sed -i '' 's/release_version: v.*/release_version: v0.16.0/' ansible/inventory.yml

# 4. Deploy all services
cd ansible && ./deploy.sh
```

### Quick redeploy after code changes

```bash
# Assuming tag already exists and you want to force rebuild
cd ansible
ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook deploy.yml -v
```

### Monitor all services

```bash
# Check health of all services
for service in cpusim-server collector-server requester-server dashboard-server; do
  echo "=== $service ==="
  ansible all -m shell -a "systemctl is-active $service 2>/dev/null || echo 'not found'" -b
done
```
