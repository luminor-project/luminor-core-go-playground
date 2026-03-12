# Preview build — includes both server and migrate binaries.
# Multi-stage: Node (CSS/JS assets) → Go (templ + binaries) → runtime.

# --- Stage 1: Build CSS and prepare static JS assets ---
FROM node:24-slim AS assets

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY tailwind.config.js ./
COPY assets/ ./assets/
COPY internal/ ./internal/

# Build Tailwind CSS using v4 CLI
RUN mkdir -p static/css static/js
RUN npx @tailwindcss/cli -i assets/css/app.css -o static/css/app.css --minify

# Copy app JS
RUN cp assets/js/app.js static/js/app.js

# Copy vendor JS from node_modules
RUN cp node_modules/htmx.org/dist/htmx.min.js static/js/htmx.min.js
RUN cp node_modules/alpinejs/dist/cdn.min.js static/js/alpine.min.js

# --- Stage 2: Build Go binaries ---
FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .

# Bring in the built static assets from the Node stage
COPY --from=assets /app/static ./static

RUN templ generate

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/migrate ./cmd/migrate

# --- Stage 3: Runtime (Alpine for shell access) ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates postgresql-client

WORKDIR /app

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/static /app/static
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/server"]
