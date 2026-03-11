# Development Book

## Prerequisites

- **Docker** (Docker Desktop or Docker Engine) — Everything runs inside containers
- **mise** — Task runner

No Go, Node, or other toolchains needed on the host. The Docker app container includes Go 1.26, templ, air, and Node.js.

## Setup

```bash
mise run setup
```

## Daily Development

```bash
mise run dev    # Start hot-reload dev server
```

This runs `air` inside the Docker container, which watches for changes and automatically rebuilds.

All mise tasks use `mise run in-app-container` under the hood, which runs commands inside the Docker app container via `docker compose exec`.

## Available Tasks

| Task                           | Description                                         |
| ------------------------------ | --------------------------------------------------- |
| `mise run setup`               | One-command bootstrap                               |
| `mise run dev`                 | Hot reload development                              |
| `mise run build`               | Full production build                               |
| `mise run quality`             | Go + frontend/docs/config linting and format checks |
| `mise run quality-strict`      | `quality` plus generated/format drift gate          |
| `mise run security`            | govulncheck + npm audit checks                      |
| `mise run tests`               | Unit tests                                          |
| `mise run tests-integration`   | Integration tests (requires Docker)                 |
| `mise run tests-e2e`           | Playwright E2E tests                                |
| `mise run all-checks`          | Everything                                          |
| `mise run migrate-db:business` | Run business database migrations                    |
| `mise run migrate-db:rag`      | Run RAG database migrations                         |
| `mise run browser`             | Open app URL in your browser (best effort)          |

## Adding a New Vertical

1. Create directory structure: `internal/myvertical/{domain,facade,infra,web/templates,subscriber}`
2. Define domain entities, aggregates, and domain services (for cross-aggregate invariants) in `domain/`
    - `domain/myentity.go` — aggregate struct, command methods, Apply()
    - `domain/events.go` — domain event types and constants
    - `domain/repository.go` — read model interface (if CQRS)
    - `domain/service.go` — domain service functions with injected interfaces (if cross-aggregate invariants exist)
    - `domain/serialization.go` — DeserializeEvent factory (if event-sourced)
3. Define facade DTOs/events in `facade/`
4. Define cross-vertical interfaces where they are consumed (consumer package)
5. Implement repository in `infra/`
6. Create handlers and routes in `web/`
7. Create templ templates in `web/templates/`
8. Create migration in `migrations/<database>/` (e.g., `migrations/business/`, `migrations/rag/`)
9. Wire everything in `cmd/server/main.go`
10. Register event subscribers if needed
11. Add to `tools/archtest/` verticals list

## Code Style

- Go canonical formatting (`gofmt`)
- Explicit error handling (no panics)
- Consumer-owned interfaces (define interfaces where they are used)
- DTO-first data transfer across boundaries
- Sentinel errors for domain errors
- Prettier checks for JS/TS/CSS/JSON/YAML/Markdown
- Optional drift gate for CI/release checks via `mise run quality-strict` (`git diff --exit-code`)

## Coverage Threshold

`mise run tests` enforces a minimum unit-test coverage threshold (default 60%) using average package coverage across core unit-test packages under `internal/...`.

Coverage excludes thin adapter/generated-oriented package layers:

- `/web`
- `/web/templates`
- `/infra`
- `/facade`
- `/testharness`
- `/common/web/templates/layouts`
- `/platform/auth`
- `/platform/config`
- `/platform/csrf`
- `/platform/database`
- `/platform/flash`
- `/platform/outbox`
- `/platform/render`
- `/platform/session`

Override locally when needed:

```bash
MIN_COVERAGE=65 mise run tests
```
