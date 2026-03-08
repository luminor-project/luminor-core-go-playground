FROM golang:1.26

# Install mise for tool management inside the container
RUN curl https://mise.run | sh
ENV PATH="/root/.local/bin:/root/.local/share/mise/shims:${PATH}"

# Install templ and air as Go tools
RUN go install github.com/a-h/templ/cmd/templ@latest && \
    go install github.com/air-verse/air@latest

# Install Node.js via mise (for Playwright E2E tests only)
RUN mise use -g node@24

WORKDIR /app

# Keep the container running (dev commands run via docker compose exec)
CMD ["tail", "-f", "/dev/null"]
