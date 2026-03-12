// Merged with docker-compose.yml when COMPOSE_BAKE=true.
// Target name "app" matches the Compose service name.

target "app" {
  context    = "."
  dockerfile = "docker/dev/Dockerfile"
  cache-from = ["type=gha"]
  cache-to   = ["type=gha,mode=max"]
}
