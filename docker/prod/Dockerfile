# Production multi-stage build
FROM golang:1.26 AS builder

WORKDIR /app

# Download dependencies first (cacheable layer)
COPY go.mod go.sum ./
RUN go mod download

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy source code
COPY . .

# Generate templ templates
RUN templ generate

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server

# Build the migrate binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/migrate ./cmd/migrate

# Runtime stage — alpine (shell needed for docker compose exec)
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/static /app/static
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/server"]
