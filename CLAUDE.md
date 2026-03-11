# luminor-core-go-playground

Module: `github.com/luminor-project/luminor-core-go-playground` — Go/htmx vertical-slice application.

## How to Run Things

All Go, Node, and templ execution happens inside Docker. Only `mise` and Docker are required on the host.
Never run `go`, `templ`, `npm`, or `air` directly — always use `mise run in-app-container <cmd>`.

| Task                              | What it does                                       |
| --------------------------------- | -------------------------------------------------- |
| `mise run setup`                  | Bootstrap everything (build, start, migrate, test) |
| `mise run dev`                    | Hot-reload server (air + tailwind watch)           |
| `mise run build`                  | Generate templ + build CSS + compile Go binary     |
| `mise run quality`                | Lint, archtest, format checks                      |
| `mise run tests`                  | Unit tests with coverage threshold                 |
| `mise run tests-integration`      | Integration tests                                  |
| `mise run tests-e2e`              | End-to-end tests                                   |
| `mise run all-checks`             | build + quality + security + all test suites       |
| `mise run seed`                   | Seed demo data                                     |
| `mise run migrate-db:<db>`        | Run migrations (e.g. `migrate-db:business`)        |
| `mise run in-app-container <cmd>` | Run any command inside the app container           |

When adding a new Go entry point under `cmd/`, always create a corresponding `.mise/tasks/` wrapper script.

## Quality Gates

Before you're done, `mise run all-checks` must be green. This is the single validation command.

- `mise run quality` runs: eslint, prettier, go vet, golangci-lint, archtest, gofmt
- Architecture rules live in `tools/archtest/` — don't memorize them, just keep checks green
- Unit test coverage threshold: 60% (override: `MIN_COVERAGE=N mise run tests`)
- Always run `mise run quality` before committing

## Where to Learn

Read the docs before making architectural decisions. Each book answers different questions:

- `docs/archbook.md` — architecture rules, boundary enforcement, event sourcing, facade purity
- `docs/devbook.md` — development workflow, available tasks, coverage scope and exclusions
- `docs/techbook.md` — technology choices and key dependencies
- `docs/frontendbook.md` — template architecture, htmx/Alpine.js, design system
- `docs/orgbook.md` — team and project organization
- `docs/runbook.md` — operational procedures
- `docs/setupbook.md` — environment setup details
- `tools/archtest/` — architecture rules as executable code (`policy.go` for the vertical registry)
- `../luminor-planning/` — sibling repo with ADRs, use cases, field notes

## Key Behavioral Rules

- Always go through `facade/` for cross-vertical imports — archtest enforces this
- Define interfaces where they are consumed, not in the provider package
- Never call `time.Now()` in business logic — inject a `Clock`
- Use typed string constants for domain concepts, never bare strings
- Domain packages have zero infrastructure imports
- New `internal/` verticals must be registered in `tools/archtest/policy.go`
- Wiring happens in `cmd/server/main.go` — no DI container
- Entry points live in `cmd/` (server, migrate, seed, worker)
- Read the docs before making architectural decisions

## Testing Conventions

- Domain: pure unit tests with injected `clock.NewFixed()`
- Handlers: fakes + `httptest`
- Fixtures: `testharness/` packages within each vertical
- See `docs/devbook.md` for coverage exclusions, thresholds, and scope
