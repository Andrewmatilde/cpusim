# Repository Guidelines

## Project Structure & Modules
- `cmd/` — service entrypoints: `cpusim-server`, `collector-server`, `dashboard-server`, `requester-server`.
- `calculator/` — GCD core; `collector/`, `dashboard/`, `requester/` — service logic.
- `api/` — OpenAPI specs + oapi-codegen configs; generated code in `<service>/api/generated/`.
- `web/` — React + Vite frontend. Runtime data in `data/`, configs in `configs/`, binaries in `bin/`.

## Build, Test, and Dev
- Go (CI uses 1.22):
  - `make build|build-collector|build-dashboard|build-requester|build-all` → builds to `bin/`.
  - `make run` (port 80, requires sudo), `make run-collector` (8080), `make run-dashboard` (9090), `make run-requester` (80).
  - `go build -v ./...`, `go vet ./...` for local checks.
  - API codegen: `go generate ./...` then commit any changes (CI verifies no diff).
- Web:
  - `cd web && npm i`
  - `npm run dev` (Vite), `npm run build`, `npm run lint`.

## Coding Style & Naming
- Go: format with `go fmt ./...`; run `go vet ./...`. Keep package names short, lowercase; files use `snake_case.go`.
- Follow existing module layout: business code in packages, thin `cmd/<service>/main.go`.
- JSON field names must match OpenAPI (do not change generated tags). Avoid global mutable state; pass deps explicitly.

## Testing Guidelines
- Framework: Go `testing`. Place tests as `*_test.go` with `TestXxx` functions.
- Run all tests: `go test ./...` (or targeted: `go test ./collector/... ./requester/...`). Use `t.TempDir()` for filesystem.
- Keep tests deterministic and fast; prefer table tests and explicit timeouts when using goroutines/timers.

## Commit & PR Guidelines
- Use Conventional Commits when possible: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:` (history includes `feat:`/`fix:` examples).
- PRs must include: clear description, linked issues, relevant screenshots for `web/` changes, and notes on breaking changes.
- Before opening a PR: `go generate ./...` (no git diff), `go vet ./...`, `make build-all` passes, and update docs/configs as needed.

## Security & Config Tips
- Default ports: CPU 80, Collector 8080, Dashboard 9090; set with `PORT`. Use unprivileged ports for local dev when possible.
- Never commit secrets or local data; `data/` is runtime-only.
- For privileged ports (80), use `sudo make run` or override `-port` flag in dev.

