---
name: deploy-release
description: This skill should be used when deploying or releasing the CPU simulation distributed system, including creating GitHub releases, building binaries, and deploying services to cloud infrastructure via Ansible. Use when the user asks to deploy, release, update services, or manage the distributed system infrastructure.
---

# Deploy and Release Skill

## Purpose

This skill provides procedures for releasing and deploying the CPU simulation distributed system, which consists of four server components (cpusim-server, collector-server, requester-server, dashboard-server) and a frontend application deployed across multiple cloud hosts.

## When to Use This Skill

Use this skill when:
- Creating a new release version
- Deploying services to production infrastructure
- Updating deployed services to a new version
- Managing service lifecycle (start, stop, restart)
- Troubleshooting deployment issues
- Verifying service health after deployment

## System Components Overview

The system consists of:
- **cpusim-server**: CPU simulation service (GCD calculator)
- **collector-server**: Metrics collection service
- **requester-server**: Request generator service
- **dashboard-server**: Management backend API
- **dashboard-frontend**: Web UI for management

See [references/infrastructure.md](references/infrastructure.md) for detailed architecture, host information, and service configuration.

## Release Workflow

### Creating a GitHub Release

To create a new release:

1. **Check existing tags first**:
   ```bash
   # List recent tags sorted by version
   git tag --sort=-v:refname | head -10

   # Check if current HEAD already has a tag
   git tag --points-at HEAD

   # View what commit a specific tag points to
   git log <tag> --oneline -1
   ```

   This helps avoid creating duplicate tags or overwriting existing tags.

2. Create and push a new tag:
   ```bash
   git tag v0.X.X
   git push origin v0.X.X
   ```

   If you need to delete and recreate a tag:
   ```bash
   # Delete local tag
   git tag -d v0.X.X

   # Delete remote tag
   git push origin :refs/tags/v0.X.X

   # Create new tag
   git tag v0.X.X
   git push origin v0.X.X
   ```

3. Monitor the GitHub Actions workflow:
   ```bash
   gh run list --workflow=release.yml
   gh run watch
   ```

3. Verify release artifacts are created:
   ```bash
   gh release view v0.X.X
   ```

The GitHub Actions workflow automatically:
- Builds all server binaries for multiple platforms
- Builds and packages the dashboard frontend
- Creates a GitHub Release with all artifacts
- Uploads binaries and frontend package to the release

**Note**: If the "Create Release" step fails due to credential issues, but all build artifacts are created successfully, you can manually create the release and upload artifacts:

```bash
# Create the release manually
gh release create v0.X.X --title "Release v0.X.X - Description" --notes "Release notes here"

# Download artifacts from the workflow run
mkdir -p /tmp/v0.X.X-artifacts
gh run download <run-id> -D /tmp/v0.X.X-artifacts -R Andrewmatilde/cpusim

# Upload artifacts to the release
cd /tmp/v0.X.X-artifacts
for dir in */; do
  file="${dir%/}/${dir%/}"
  if [ -f "$file" ]; then
    gh release upload v0.X.X "$file" -R Andrewmatilde/cpusim
  fi
done
```

### Manual Release Trigger

To manually trigger the release workflow:
1. Navigate to GitHub Actions in the repository
2. Select the "Build and Release" workflow
3. Click "Run workflow" and specify the tag/branch

## Deployment Workflow

### Prerequisites Check

Before deploying, verify:

1. Ansible is installed: `ansible --version`
2. SSH key exists and has correct permissions:
   ```bash
   ls -la ~/.ssh/andrew.pem
   chmod 600 ~/.ssh/andrew.pem
   ```
3. Ansible inventory is configured: Check `ansible/inventory.yml`

### Updating Release Version

Before deployment, update the release version in the inventory:

```bash
# Edit ansible/inventory.yml and update:
# release_version: v0.X.X
```

Or use sed to update programmatically:
```bash
sed -i '' 's/release_version: v.*/release_version: v0.X.X/' ansible/inventory.yml
```

### Deploying All Services

To deploy all services to all hosts:

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

### Deploying Individual Services

To deploy specific components:

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
```

## Service Management

### Checking Service Status

To check status of all services:

```bash
ansible all -m shell -a "systemctl status cpusim-server" -b
ansible all -m shell -a "systemctl status collector-server" -b
ansible all -m shell -a "systemctl status requester-server" -b
ansible all -m shell -a "systemctl status dashboard-server" -b
```

### Starting/Stopping Services

To manage individual services on a host:

```bash
# Start a service
sudo systemctl start cpusim-server

# Stop a service
sudo systemctl stop cpusim-server

# Restart a service
sudo systemctl restart cpusim-server
```

### Viewing Service Logs

To view logs for troubleshooting:

```bash
# Follow logs in real-time
sudo journalctl -u cpusim-server -f

# View recent logs
sudo journalctl -u cpusim-server --since "10 minutes ago"
sudo journalctl -u cpusim-server -n 50
```

## Verification After Deployment

### Testing Service Endpoints

After deployment, verify each service is responding. Replace `<host-ip>` with the actual IP from `ansible/inventory.yml`:

```bash
# CPU simulation service (on any target host)
curl -X POST http://<target-host-ip>:80/calculate -H 'Content-Type: application/json' -d '{}'

# Collector service (on any target host)
curl http://<target-host-ip>:8080/health

# Requester service (on client host)
curl http://<client-host-ip>:80/health

# Dashboard backend
curl http://<dashboard-host-ip>:9090/health

# Dashboard frontend
curl http://<dashboard-host-ip>:8080
```

To get the actual IPs from inventory:

```bash
# View all host IPs
grep ansible_host ansible/inventory.yml
```

### Checking System Health

To get a comprehensive health check:

```bash
for service in cpusim-server collector-server requester-server dashboard-server; do
  echo "=== $service ==="
  ansible all -m shell -a "systemctl is-active $service 2>/dev/null || echo 'not found'" -b
done
```

## Troubleshooting Common Issues

### Connection Issues

If deployment fails with "Connection refused":
1. Check SSH key permissions: `chmod 600 ~/.ssh/andrew.pem`
2. Test connectivity: `ansible all -m ping`
3. Verify host IP addresses in inventory.yml

### Service Startup Failures

If a service fails to start:
1. Check logs: `journalctl -u {service-name} -n 50`
2. Verify binary permissions: `ls -la /opt/{service}/bin/`
3. Check environment file: `cat /opt/{service}/{service}.env`
4. Check port availability: `lsof -i :{port}`

### Binary Download Failures

If binary download fails during deployment:
1. Verify GitHub release exists: `gh release view v0.X.X`
2. Check release artifacts are uploaded
3. Verify GitHub token (SIM_TOKEN secret) is set correctly

## Complete Release and Deployment Checklist

When performing a full release and deployment:

1. Commit all changes to the repository
2. Create and push a new tag: `git tag v0.X.X && git push origin v0.X.X`
3. Wait for GitHub Actions workflow to complete: `gh run watch`
4. Verify release artifacts: `gh release view v0.X.X`
5. Update release version in `ansible/inventory.yml`
6. Run deployment: `cd ansible && ./deploy.sh`
7. Verify all services are running (see "Checking Service Status" above)
8. Test all service endpoints (see "Testing Service Endpoints" above)
9. Monitor logs for any issues: `journalctl -u {service} -f`

## Reference Documentation

For detailed information about:
- Infrastructure architecture and network topology
- Host IP addresses and port mappings
- Service configuration files and environment variables
- Ansible playbook structure and templates
- Advanced troubleshooting scenarios

See [references/infrastructure.md](references/infrastructure.md).
