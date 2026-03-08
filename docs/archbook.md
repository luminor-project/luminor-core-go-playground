# Architecture Book

## Vertical Slice Architecture

The application is organized into self-contained verticals. Each vertical owns its complete stack from domain logic to HTTP handlers and templates.

### Verticals

- **account** — User authentication, profile management, session handling
- **organization** — Multi-tenancy, teams, groups, invitations, access rights
- **content** — Public pages (homepage, about)

### Shared Packages

- **common** — Cross-cutting UI concerns (layouts, navigation components, flash messages)
- **shared** — Value objects used across verticals (EmailAddress)
- **platform** — Infrastructure (config, database, event bus, sessions, auth, CSRF, flash, render)

## Boundary Rules

1. **Vertical X must NOT import vertical Y's internal packages** (`domain/`, `infra/`, `web/`, `subscriber/`, `testharness/`)
2. **Cross-vertical imports are allowed ONLY through `facade/` packages**
3. **Cross-vertical concrete symbol usage is forbidden** (deep calls/constructors/foreign concrete members), except explicit DTO/event allowlisted value symbols
4. **Cross-vertical collaboration must be interface-oriented** (interfaces are defined by the consumer package)
5. **DTO/event value types are allowed only via explicit symbol allowlist**
6. These rules are enforced by `tools/archtest/` using both import checks and type-aware symbol analysis (`go/packages` + `go/types`)
7. `common/`, `shared/`, and `platform/` are exempt — any vertical can import them

## Facade Pattern

Each vertical's `facade/` package exports:

- **DTO structs** for data transfer (no domain entities leak)
- **Event types** that this vertical publishes
- Optional façade implementation/wiring helpers

```go
// consumer package
type accountReader interface {
    GetAccountInfoByID(ctx context.Context, id string) (accountfacade.AccountInfoDTO, error)
}
```

Interfaces are defined where they are consumed, not in the provider package.
The provider may expose concrete facade implementations for wiring, but consumers collaborate through their own narrow interfaces.

## Event Bus

A synchronous in-process event bus in `internal/platform/eventbus/` replaces Symfony's EventDispatcher. Events are plain structs defined in each vertical's `facade/events.go`. Subscribers are registered during wiring in `cmd/server/main.go`.

### Event Chain: User Registration

1. Account registration creates account → publishes `AccountCreatedEvent`
2. Organization subscriber receives event → creates default org → publishes `ActiveOrgChangedEvent`
3. Account subscriber receives event → sets `currentlyActiveOrganizationID`

## Archtest Modes

- **Enforce mode (default):** `go run ./tools/archtest` fails on violations.
- **Report-only mode:** set `ARCHTEST_REPORT_ONLY=1` to print violations without failing (useful during rollout/cleanup).

## Dependency Injection

No container. All dependencies wired explicitly in `cmd/server/main.go`. Dependencies flow downward: main.go → facades → services → repositories.

## Database Rules

- **No cross-vertical foreign keys** — Cross-vertical references stored as indexed UUID columns
- **Within-vertical FKs with CASCADE** — Groups and invitations reference their parent organization
- **Join tables** belong to the vertical that semantically owns the relationship

## Error Handling

Application/domain paths are Go-idiomatic: return `error` values and use sentinel errors with `errors.Is()`.  
Current startup/bootstrap code still contains a few fail-fast panics in platform initialization (config/database/context helpers); treat those as bootstrap guards, not request-path behavior.
