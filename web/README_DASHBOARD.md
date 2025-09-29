# CPU Simulation Dashboard - Frontend

## Overview
This is the frontend management tool for the CPU Simulation Dashboard API. It provides a web interface to manage and monitor CPU simulation hosts, run experiments, and perform calculation tests.

## Features
- 🖥️ Real-time host health monitoring
- 🧪 Experiment creation and management
- 📊 CPU performance testing with GCD calculations
- 📈 Live metrics tracking (CPU, Memory, Network)
- 🔄 Auto-refresh every 10 seconds
- 🎨 Modern UI with shadcn/ui components

## Prerequisites

### 1. Start the Backend API Server
The dashboard backend must be running on port 9090. From the project root:

```bash
cd ../cmd/dashboard-server
go run main.go
```

Or if you have a compiled binary:
```bash
../cmd/dashboard-server/dashboard-server
```

The backend should be accessible at: http://localhost:9090

### 2. Ensure CPU Simulation Services are Running
The dashboard connects to CPU simulation hosts that should have:
- CPU Service (for calculations)
- Collector Service (for metrics)

## Running the Frontend

### Development Mode
```bash
npm run dev
```
The application will be available at: http://localhost:5173

### Production Build
```bash
npm run build
npm run preview
```

## Project Structure
```
src/
├── api/
│   ├── client.ts        # API client implementation
│   └── types.ts         # TypeScript type definitions
├── components/
│   ├── Dashboard.tsx    # Main dashboard component
│   ├── HostCard.tsx     # Individual host display
│   ├── ExperimentManager.tsx  # Experiment management
│   ├── CalculationTest.tsx    # CPU calculation testing
│   └── ui/              # shadcn/ui components
└── hooks/
    └── useHosts.ts      # React hook for host data

```

## API Endpoints
The frontend connects to the following API endpoints:
- `GET /api/hosts` - List all hosts
- `GET /api/hosts/{name}/health` - Host health status
- `POST /api/hosts/{name}/calculate` - Run CPU calculation test
- `GET /api/hosts/{name}/experiments` - List experiments
- `POST /api/hosts/{name}/experiments` - Start new experiment
- `GET /api/hosts/{name}/experiments/{id}/status` - Experiment status
- `POST /api/hosts/{name}/experiments/{id}/stop` - Stop experiment

## Troubleshooting

### "Cannot connect to backend API" error
- Ensure the dashboard backend is running on port 9090
- Check: `curl http://localhost:9090/api/hosts`
- Verify no firewall is blocking port 9090

### No hosts showing
- Verify your CPU simulation hosts are registered with the dashboard
- Check the backend logs for any connection issues

### Build errors
- Clear node_modules: `rm -rf node_modules && npm install`
- Clear Vite cache: `rm -rf .vite`

## Technology Stack
- React 19 with TypeScript
- Vite for build tooling
- Tailwind CSS for styling
- shadcn/ui component library
- Sonner for toast notifications
- Lucide React for icons