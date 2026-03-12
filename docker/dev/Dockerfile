FROM golang:1.26

# Install mise for tool management inside the container
RUN curl https://mise.run | sh
ENV PATH="/root/.local/bin:/root/.local/share/mise/shims:${PATH}"

# Install Go tools (templ, air, golangci-lint, govulncheck).
# Baked into the image so they're cached in the Docker layer.
RUN go install github.com/a-h/templ/cmd/templ@latest && \
    go install github.com/air-verse/air@latest && \
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest && \
    go install golang.org/x/vuln/cmd/govulncheck@latest

# Install Node.js via mise (for Playwright E2E tests only)
RUN mise use -g node@24

WORKDIR /app

# Pre-download Go module dependencies (cacheable layer).
# Only invalidated when go.mod/go.sum change, not on source changes.
COPY go.mod go.sum ./
RUN go mod download

# Keep the container running (dev commands run via docker compose exec)
CMD ["tail", "-f", "/dev/null"]
