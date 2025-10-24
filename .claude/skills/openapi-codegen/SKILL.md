---
name: openapi-codegen
description: Regenerate backend and frontend code from OpenAPI specifications. Use this skill when OpenAPI files in api/ directory are modified (collector.openapi.yaml, dashboard.openapi.yaml, or requester.openapi.yaml), or when the user requests to regenerate API types or client code.
---

# OpenAPI Code Generation

## Overview

This skill automates the regeneration of backend Go code and frontend TypeScript client code from OpenAPI specifications. It handles all three API services (collector, dashboard, requester) in the backend and dashboard client code in the frontend.

## When to Use This Skill

Use this skill when:

- OpenAPI files in `api/` directory are modified
- User requests to "regenerate API types" or "generate client code"
- User mentions updating OpenAPI specs and needing code regeneration
- After changes to any of: `api/collector.openapi.yaml`, `api/dashboard.openapi.yaml`, or `api/requester.openapi.yaml`

## Code Generation Process

### Backend Code Generation

Generate Go server code for all three API services:

```bash
go generate ./...
```

This command:

- Reads [generate.go](generate.go:1) which contains go:generate directives
- Processes all three OpenAPI files:
  - [api/collector.openapi.yaml](api/collector.openapi.yaml) → generates collector server code
  - [api/dashboard.openapi.yaml](api/dashboard.openapi.yaml) → generates dashboard server code
  - [api/requester.openapi.yaml](api/requester.openapi.yaml) → generates requester server code
- Uses oapi-codegen tool with respective config files

### Frontend Code Generation (Dashboard Only)

Generate TypeScript client code for the dashboard:

```bash
cd web && ./generate-api-types.sh
```

This script:

- Only processes [api/dashboard.openapi.yaml](api/dashboard.openapi.yaml)
- Generates TypeScript fetch client in `web/src/api/generated/`
- Configures the client with ES6 support and TypeScript 3+ features

**Note:** Only dashboard API has a frontend client. Collector and requester APIs are backend-only.

## Complete Workflow

When OpenAPI files are modified, execute both generation steps:

1. **Backend generation** (from project root):

   ```bash
   go generate ./...
   ```

2. **Frontend generation** (if dashboard.openapi.yaml was modified):

   ```bash
   cd web && ./generate-api-types.sh
   ```

No validation is required before generation - code generation is a routine operation in this project's workflow.
