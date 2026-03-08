# Setup Book

## Prerequisites

You only need two tools on your host machine:

- **Docker** (Docker Desktop or Docker Engine) — Everything runs inside containers (Go, Node, PostgreSQL)
- **mise** — Task runner ([install](https://mise.jdx.dev/getting-started.html))

That's it. No Go, no Node, no other toolchain needed on the host.

On Linux hosts without Docker Desktop, project `mise` tasks automatically use `sudo docker compose` when the Docker socket requires elevated access.

## Quick Start

```bash
git clone https://github.com/luminor-project/luminor-core-go-playground.git
cd luminor-core-go-playground
mise run setup
```

This single command will:

1. Build and start Docker containers (app + PostgreSQL)
2. Download Go dependencies inside the container
3. Generate templ templates
4. Build Tailwind CSS
5. Copy JS assets
6. Run database migrations
7. Run quality checks and tests
8. Attempt to open the app in your browser (best effort)

The app will be available at http://localhost:8090.
If the environment is headless/non-GUI, `mise run browser` prints the URL instead of failing.

## How It Works

All development commands run **inside the Docker container** via `mise run in-app-container`. The app container has Go 1.26, templ, air (hot reload), and Node.js pre-installed. The project directory is mounted as a volume, so code changes on the host are immediately reflected inside the container.

```bash
# This runs `go test` inside the container, not on the host:
mise run tests

# You can also run arbitrary commands in the container:
mise run in-app-container go version
mise run in-app-container templ generate
```

## Manual Setup

If you prefer step-by-step:

```bash
# 1. Build and start containers
docker compose build
docker compose up -d

# 2. Install Go dependencies
mise run in-app-container go mod download

# 3. Generate templ templates
mise run in-app-container templ generate

# 4. Build CSS + copy vendored JS assets
mise run prepare-assets

# 5. Run migrations
mise run migrate-db:business
mise run migrate-db:rag

# 6. Start development server (hot reload)
mise run dev

# Optional: open app URL in your browser
mise run browser
```

## Environment Variables

Environment variables are set in `docker-compose.yml` for the app container:

| Variable       | Default                                | Description                                  |
| -------------- | -------------------------------------- | -------------------------------------------- |
| `APP_ENV`      | `development`                          | Environment (development/production)         |
| `PORT`         | `8090`                                 | HTTP server port                             |
| `DATABASE_URL` | `postgres://...@postgres:5432/luminor` | PostgreSQL connection (container networking) |
| `SESSION_KEY`  | (dev default)                          | 32-byte session encryption key               |
| `BASE_URL`     | `http://localhost:8090`                | Application base URL                         |

## Troubleshooting

### Container won't start

```bash
docker compose logs app     # Check app container logs
docker compose logs postgres # Check database logs
```

### Rebuild after Dockerfile changes

```bash
docker compose build
docker compose up -d
```

### Port already in use

Change the port mapping in `docker-compose.yml` or stop the conflicting process.
