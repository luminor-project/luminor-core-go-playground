# Tech Book

## Tech Stack Decisions

### Router: `net/http` stdlib

Go 1.22+ added method routing and path parameters to the standard library router. No framework needed.

```go
mux.Handle("GET /sign-in", handler)
mux.Handle("POST /organization/switch/{organizationId}", handler)
```

### Templates: templ

Compile-time type-safe HTML templates with component model. Templates are `.templ` files that generate Go code. IDE support via templ LSP.

### Database: PostgreSQL + raw SQL

PostgreSQL provides native UUIDs, JSONB for flexible data, and TEXT[] for access rights. Raw SQL queries via pgx for full control and performance.

### Frontend: htmx + Alpine.js

- **htmx** — Server-driven interactions (form submissions, partial updates, SSE)
- **Alpine.js** — Client-only state (dropdowns, modals, dark mode)
- Both served as vendored `.min.js` files — no bundler

### CSS: Tailwind standalone CLI

Tailwind CSS is built via the Tailwind CLI in project scripts/tasks. In this repo, Node tooling is used for asset and quality workflows (inside the dev container), while output remains plain static CSS served without a bundler.

### Sessions & Auth

Cookie-based sessions via `gorilla/sessions`. Authentication uses bcrypt password hashing with custom middleware. CSRF protection uses Go stdlib `net/http.CrossOriginProtection` (origin/fetch-metadata checks for unsafe methods).

### Migrations: golang-migrate

SQL-based migrations in `migrations/business/` and `migrations/rag/` directories. CLI runner in `cmd/migrate/` takes database name as first argument (e.g., `go run ./cmd/migrate business up`).

### Config: caarlos0/env

Twelve-factor configuration via environment variables. Config struct with `env` struct tags.

### Event Bus: Custom synchronous

~40 lines of Go using generics. Type-safe event publishing and subscription.

### Background Jobs: Outbox worker (current) + queue evolution (future)

Outbox events are persisted in PostgreSQL and dispatched by `cmd/worker`. This provides retryable asynchronous publishing for integration events; queue specialization can be introduced later if needed.

## Key Dependencies

```
github.com/a-h/templ           — Type-safe HTML templates
github.com/google/uuid          — UUID generation
github.com/gorilla/sessions     — Session management
github.com/jackc/pgx/v5         — PostgreSQL driver
github.com/golang-migrate/migrate/v4 — Database migrations
github.com/caarlos0/env/v11     — Environment config
golang.org/x/crypto             — bcrypt password hashing
```
