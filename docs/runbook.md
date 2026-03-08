# Runbook

## Production Deployment

### Docker Build

```bash
docker build -t lcgp-app -f docker/prod/Dockerfile .
```

This creates a multi-stage build:

1. **Builder stage** — Compiles Go binary with templ generation
2. **Runtime stage** — Distroless container with only the binary and static assets

### Environment Variables

Set these in production:

- `APP_ENV=production`
- `DATABASE_URL` — PostgreSQL connection string with SSL
- `SESSION_KEY` — Random 32-byte key (generate with `openssl rand -hex 16`)
- `BASE_URL` — Public URL of the application
- `PORT` — HTTP port (default 8090)

### CSRF Protection

The app uses Go stdlib `net/http.CrossOriginProtection` for CSRF protection.

- It blocks cross-origin unsafe requests (`POST`, `PUT`, `PATCH`, `DELETE`, ...).
- Safe methods (`GET`, `HEAD`, `OPTIONS`) are always allowed and must remain side-effect free.
- Protection relies on modern browser headers (`Sec-Fetch-Site` and/or `Origin`).

### Database Migrations

Run before deploying new versions (from a Go-capable environment/CI runner with repo source):

```bash
DATABASE_URL=postgres://... go run ./cmd/migrate business up
RAG_DATABASE_URL=postgres://... go run ./cmd/migrate rag up
```

### Health Check

The server responds to all routes. Use `/` as a basic health check endpoint.

## Monitoring

### Structured Logging

The application uses `log/slog` with text output (`slog.NewTextHandler`). Key log events:

- Server start/stop
- Database connection
- Authentication attempts
- Event bus dispatches
- Error conditions

### Database

Monitor PostgreSQL connection pool metrics and query performance.

## Troubleshooting

### Application won't start

1. Check DATABASE_URL is correct and PostgreSQL is reachable
2. Check SESSION_KEY is set (32 bytes)
3. Check port is not already in use

### Migrations fail

1. Check database connectivity
2. Check migration files exist in `migrations/business/` or `migrations/rag/`
3. Check for dirty migration state: `DATABASE_URL=... go run ./cmd/migrate business version`
