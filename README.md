# luminor-core-go-playground

[![CI](https://github.com/luminor-project/luminor-core-go-playground/actions/workflows/ci.yml/badge.svg)](https://github.com/luminor-project/luminor-core-go-playground/actions/workflows/ci.yml)
[![Tests](https://github.com/luminor-project/luminor-core-go-playground/actions/workflows/tests.yml/badge.svg)](https://github.com/luminor-project/luminor-core-go-playground/actions/workflows/tests.yml)
[![Security](https://github.com/luminor-project/luminor-core-go-playground/actions/workflows/security.yml/badge.svg)](https://github.com/luminor-project/luminor-core-go-playground/actions/workflows/security.yml)

An opinionated Go web application foundation built around vertical slice architecture with strict boundary enforcement, comprehensive testing, and excellent DX.

## Quick Start

**Prerequisites:** Docker + [mise](https://mise.jdx.dev). That's it — no Go, no Node on the host.

```bash
mise run setup
```

This will:

1. Build and start Docker containers (app + PostgreSQL)
2. Download Go dependencies inside the container
3. Generate templ templates
4. Build Tailwind CSS
5. Run migrations
6. Run quality checks and tests
7. Attempt to open the app in your browser (best effort)
8. Print the app URL

Everything runs inside Docker containers. The app container has Go 1.26, templ, air, and Node.js pre-installed.

## Development

```bash
mise run dev          # Hot reload with Air inside the container
mise run build        # Full build (templ + tailwind + go build)
mise run db           # Open a PostgreSQL shell (psql) in the db container
mise run quality      # Go + frontend/docs/config linting and format checks
mise run security     # govulncheck + npm audit checks
mise run tests        # Unit tests
mise run all-checks   # Everything
mise run browser      # Open app URL in your browser (best effort)
```

## Tech Stack

| Concern       | Technology                                       |
| ------------- | ------------------------------------------------ |
| Language      | Go 1.26+                                         |
| Router        | `net/http` stdlib (method routing + path params) |
| Templates     | [templ](https://templ.guide) (compile-time safe) |
| Database      | PostgreSQL 17 + raw SQL                          |
| Interactivity | htmx + Alpine.js                                 |
| CSS           | Tailwind CSS (standalone CLI)                    |
| Sessions      | gorilla/sessions                                 |
| Auth          | bcrypt + sessions + middleware                   |
| Migrations    | golang-migrate/migrate                           |
| Config        | caarlos0/env (twelve-factor)                     |
| Logging       | log/slog (stdlib)                                |
| DI            | Manual wiring in main.go                         |

## Architecture

The project follows **vertical slice architecture** with strict boundary enforcement:

```
internal/
├── account/        # User auth & profile
├── organization/   # Multi-tenancy & teams
├── content/        # Public pages
├── common/         # Shared UI, layouts
├── shared/         # Value objects
└── platform/       # Infrastructure (config, db, auth, events)
```

Each vertical has:

- `domain/` — Entities, services, business logic
- `facade/` — Cross-vertical value contracts (DTOs/events) and wiring helpers
- `infra/` — PostgreSQL repositories
- `web/` — HTTP handlers + templ templates
- `subscriber/` — Event handlers

**Key rule:** Verticals can only import each other's `facade/` package and should collaborate through consumer-owned interfaces. Never import another vertical's `domain/`, `infra/`, `web/`, or `subscriber/`. `tools/archtest/` enforces both import boundaries and type-aware checks against foreign concrete symbol usage, with explicit allowlists for cross-vertical value types (DTOs/events).

## Cross-Vertical Communication

Verticals communicate through:

1. **Consumer-owned interfaces** — Stable contracts defined by the consuming package
2. **Events** — Synchronous in-process event bus for state changes

Example event chain on user registration:

1. Account registers → dispatches `AccountCreatedEvent`
2. Organization subscriber creates default org → dispatches `ActiveOrgChangedEvent`
3. Account subscriber sets `currentlyActiveOrganizationID`

## Frontend Formatting and Linting

```bash
npm run lint:frontend
npm run format:check
npm run format:write
```

## Documentation

- [archbook.md](docs/archbook.md) — Architecture rules and patterns
- [techbook.md](docs/techbook.md) — Tech stack decisions
- [devbook.md](docs/devbook.md) — Development workflows
- [frontendbook.md](docs/frontendbook.md) — htmx + Alpine.js + templ patterns
- [setupbook.md](docs/setupbook.md) — Prerequisites and setup
- [orgbook.md](docs/orgbook.md) — Organization domain model
- [runbook.md](docs/runbook.md) — Production operations

## Docker

Development uses 2 containers (app + PostgreSQL) via Docker Compose. The app container runs Go with the project mounted as a volume.

```bash
docker compose build           # Build containers
docker compose up -d           # Start containers
docker compose down            # Stop all
```

Production build (multi-stage, alpine runtime):

```bash
docker build -t lcgp-app -f docker/prod/Dockerfile .
docker run -p 8090:8090 -e DATABASE_URL=... lcgp-app
```




