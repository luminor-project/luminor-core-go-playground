# Architecture Book

## Vertical Slice Architecture

The application is organized into self-contained verticals. Each vertical owns its complete stack from domain logic to HTTP handlers and templates.

### Verticals

**CRUD-based:**

- **account** — User authentication, profile management, session handling
- **organization** — Multi-tenancy, teams, groups, invitations, access rights
- **content** — Public pages (homepage, about, living styleguide)
- **rag** — Retrieval-augmented generation with pgvector embeddings

**Event-sourced:**

- **workitem** — Event-sourced WorkItem aggregate (cases, timeline, notes). No repository — domain is a pure state machine reconstituted from events.
- **party** — Party identity aggregate (human and AI actors).
- **subject** — Subject info aggregate (properties, units).
- **rental** — Rental aggregate linking tenants to subjects. Domain service enforces cross-aggregate uniqueness.

**Synthesis:**

- **app_casehandling** — Orchestrates workitem + party + subject + agentic workload into UC-01 (Inbound Message to Managed Response). Owns the case dashboard read model and projection subscribers.

### Shared Packages

- **common** — Cross-cutting UI concerns (layouts, navigation components, flash messages)
- **shared** — Value objects used across verticals (EmailAddress)
- **platform** — Infrastructure (config, database, event bus, event store, sessions, auth, CSRF, flash, render, agentic workload port)

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

### Facade Purity Principle

Facades serve different roles depending on the vertical type. The purity rules reflect this.

#### Synthesis facades (e.g. app_casehandling)

A synthesis facade coordinates multiple verticals. Every method must justify itself by **multi-dependency orchestration** — coordinating calls across two or more verticals or infrastructure ports.

A method that delegates to a single dependency **does not belong on the synthesis facade**. The handler should consume that dependency's facade directly via its own narrow interface.

**Example:** `HandleInboundInquiry` orchestrates workitem intake + agent lookup + agent draft across three dependencies. This is genuine orchestration — it belongs on the facade. `ConfirmOutboundMessage`, `AddNote` — each would only delegate to the workitem facade. They do not belong on app_casehandling's facade:

```go
// Handler uses workitem facade directly for single-vertical operations
type workitemCommands interface {
    ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
    AddNote(ctx context.Context, workItemID string, dto workitemfacade.AddNoteDTO) (string, error)
}
```

#### CRUD facades (e.g. account, organization, rag)

A CRUD facade is the vertical's **public API boundary**. It hides domain internals, converts domain types to DTOs, and publishes domain events. Even same-vertical consumers (web handlers) go through the facade for consistency.

Every CRUD facade method must justify itself by at least one of:

1. **Cross-vertical consumer exists** — another vertical or composition root calls it
2. **DTO conversion / domain type hiding** — translates domain entities to public DTOs
3. **Event publishing** — publishes domain events after a write operation
4. **Semantic transformation** — changes the API shape (e.g. `FindByID` → `GetActiveOrgID` extracting one field)

Pure passthroughs (e.g. `SetActiveOrganization` forwarding to the service) are tolerable when they have cross-vertical consumers or maintain the invariant that all access flows through the facade. Methods with zero callers are dead code and must be removed.

#### Event-sourced facades

An event-sourced facade is **pure infrastructure wiring**. Its write methods follow the Load-Execute-Save-Publish cycle and contain zero business decisions. Specifically:

- **Business invariants** (validation, uniqueness checks, state machine guards) live in the domain — either in aggregate command methods or in domain service functions.
- **The facade's write method** does only: generate ID → call domain service or aggregate → convert to uncommitted events → append to event store → publish facade events.
- **Cross-aggregate invariant checks** (e.g. duplicate detection via `ExistsBySubjectAndTenant`) must not appear inline in facade methods. They must be expressed as domain service functions with injected interfaces.
- **The facade's query methods** (List\*, Get\*) may use a query model interface directly — read-path operations are not business logic.

The archtest enforces this structurally: facade-local interfaces in event-sourced verticals must not declare existence-check methods (`Exists*`, `Has*`, `Check*`, `IsDuplicate*`). Such methods indicate a write-path invariant that belongs in a domain interface.

## Event Bus

A synchronous in-process event bus in `internal/platform/eventbus/` replaces Symfony's EventDispatcher. Events are plain structs defined in each vertical's `facade/events.go`. Subscribers are registered during wiring in `cmd/server/main.go`.

### Subscriber Kinds: Projections vs Reactive Commands

Not every event subscriber is a projection. The codebase uses subscribers for two distinct purposes:

| Purpose              | What it does                                        | Example                                                                                    |
| -------------------- | --------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| **Projection**       | Transforms events into a query-optimized read model | `app_casehandling/subscriber/projection.go` builds the `case_dashboard` table              |
| **Reactive command** | Triggers a side effect (a new command) in response  | `organization/subscriber/account_created.go` creates a default org when an account is made |

A **projection** answers the question "how should this event stream look for a specific query?" — it denormalizes, pre-aggregates, or reshapes data into a read model table. The metaphor is geometric: projecting a rich event stream onto a flat shape optimized for one viewpoint. Nothing is created in the write model; the read model is disposable and rebuildable from events.

A **reactive command** answers the question "what should happen next?" — it executes a new write operation in reaction to an event. The organization subscriber doesn't build a read model; it calls `orgFacade.CreateDefaultOrg()`, which is a full write with its own validation, persistence, and potentially its own events.

The distinction matters because projections are idempotent and rebuildable (replay events → same read model), while reactive commands are not inherently idempotent (replaying could create duplicate side effects without explicit guards).

### Event Chain: User Registration

1. Account registration creates account → publishes `AccountCreatedEvent`
2. Organization subscriber receives event → creates default org → publishes `ActiveOrgChangedEvent` _(reactive command)_
3. Account subscriber receives event → sets `currentlyActiveOrganizationID` _(reactive command)_

## Event-Sourced Verticals

While CRUD verticals (account, organization) use repositories for persistence, the workitem vertical uses **event sourcing**. This section documents the pattern as implemented.

### Two Persistence Models

The codebase intentionally supports both CRUD and event-sourced verticals side by side:

| Aspect           | CRUD vertical (account)         | Event-sourced vertical (workitem)   |
| ---------------- | ------------------------------- | ----------------------------------- |
| Write model      | Repository with INSERT/UPDATE   | Event store with append-only INSERT |
| Read model       | Same table as write model       | Separate projection table(s)        |
| State recovery   | Read current row                | Replay all events through Apply()   |
| Delete semantics | Soft-delete flag or hard-delete | Append a semantic deletion event    |
| Domain imports   | May import repository interface | Zero infrastructure imports (PIL-1) |

### Event Store

`internal/platform/eventstore/` provides an append-only PostgreSQL store:

- **Append(ctx, streamID, expectedVersion, events)** — Inserts events with sequential version numbers. Returns `ErrConcurrencyConflict` on version mismatch (optimistic concurrency).
- **LoadStream(ctx, streamID)** — Returns all events for a stream in version order.
- **No UPDATE or DELETE statements exist in this package.** The event store is strictly append-only.

Events table: `events (id, stream_id, stream_version, event_type, payload JSONB, causation_id, correlation_id, recorded_at)` with a unique constraint on `(stream_id, stream_version)`.

### Aggregate Reconstitution

The WorkItem aggregate is a pure state machine with zero infrastructure imports:

```go
// Domain command method — returns events, never mutates state
func (w *WorkItem) AddNote(cmd AddNoteCmd) ([]DomainEvent, error) {
    if cmd.EntryIndex < 0 || cmd.EntryIndex >= w.TimelineEntryCount {
        return nil, ErrInvalidEntryIndex
    }
    return []DomainEvent{
        {EventType: EventNoteAddedToTimelineEntry, Payload: NoteAddedToTimelineEntry{...}},
    }, nil
}

// Apply reconstitutes state from a single event (called during load)
func (w *WorkItem) Apply(eventType string, payload any) {
    switch eventType {
    case EventNoteAddedToTimelineEntry:
        e := payload.(NoteAddedToTimelineEntry)
        w.NoteIDs[e.NoteID] = false  // false = not deleted
    // ...
    }
    w.Version++
}
```

Key principles:

- Command methods validate against current state and return `[]DomainEvent` — they never mutate the aggregate
- `Apply()` reconstitutes state from events — it is the only method that mutates fields
- The domain package imports only `time` and `errors` — no database, no HTTP, no JSON (except in `serialization.go`)
- `DomainEvent` is a domain-local struct (`EventType string, Payload any`) — the domain never imports the platform event store

### Domain Services for Cross-Aggregate Invariants

Some business rules span beyond a single aggregate instance. For example, "a property may have at most one active rental per tenant" requires checking all existing rentals — not just the current aggregate's state. These **cross-aggregate invariants** belong in the domain layer, not the facade.

The pattern: define a narrow, consumer-owned interface in the domain package for the specific check needed, then write a domain service function that enforces the invariant and delegates to the aggregate:

```go
// domain/service.go — cross-aggregate invariant lives in the domain

type DuplicateChecker interface {
    ExistsBySubjectAndTenant(ctx context.Context, subjectID, tenantPartyID string) (bool, error)
}

func EstablishNewRental(ctx context.Context, checker DuplicateChecker, clock Clock, cmd EstablishRentalCmd) ([]DomainEvent, error) {
    exists, err := checker.ExistsBySubjectAndTenant(ctx, cmd.SubjectID, cmd.TenantPartyID)
    if err != nil { return nil, fmt.Errorf("check duplicate rental: %w", err) }
    if exists   { return nil, ErrDuplicateRental }

    r := NewRental(clock)
    return r.EstablishRental(cmd)
}
```

The facade injects the read model (or a narrower adapter) as the implementation. The domain never knows about databases — it depends only on the interface it defines.

**When to use a domain service vs aggregate-only:**

- **Single-aggregate invariant** (e.g. "can't establish twice"): aggregate command method handles it directly
- **Cross-aggregate invariant** (e.g. "no duplicate subject+tenant rental"): domain service with an injected checker interface

Reading the `domain/` package should reveal **all** business rules — both within-aggregate and across-aggregate. If a business rule is only visible in the facade, it belongs in the domain.

### Load-Execute-Save-Publish Cycle

The facade orchestrates the full cycle:

```
1. Load:    event store → LoadStream(streamID) → []StoredEvent
2. Replay:  for each stored event → DeserializeEvent → aggregate.Apply()
3. Check:   domain service evaluates cross-aggregate invariants (if any)
4. Execute: aggregate.CommandMethod(cmd) → []DomainEvent (or error)
           (or: domain service function handles both check + execute)
5. Save:    DomainEvent → UncommittedEvent → event store.Append(streamID, version, events)
6. Publish: StoredEvent → deserialize → facade event type → eventbus.Publish()
```

For commands that need cross-aggregate invariants, steps 3-4 are combined in a domain service function. For commands that only need single-aggregate validation, the facade calls the aggregate directly (steps 3 is skipped).

This cycle is identical for every command — `IntakeInboundMessage`, `RecordAssistantAction`, `ConfirmOutboundMessage`, `AddNote`, `EditNote`, `DeleteNote` all follow the same steps. Adding a new command means: define events, add Apply cases, write the command method (and optionally a domain service), add the facade method (same plumbing), and extend publishAll.

### Projections and Read Models

Projections are the subset of eventbus subscribers whose purpose is transforming events into query-optimized read model tables (see [Subscriber Kinds](#subscriber-kinds-projections-vs-reactive-commands) for the full distinction). They are registered in `cmd/server/main.go` alongside route registration.

**The event store and read model are deliberately not in the same transaction.** The consistency guarantee is replay, not transactional coupling:

- The event store is the **source of truth** — immutable, append-only, complete
- Read models are **disposable projections** — derived, denormalized, rebuildable
- If a projection fails, the event is already stored; the read model is stale but the truth is safe
- Recovery: replay all events through projection logic to rebuild the read model from scratch

This separation is fundamental to CQRS. The write side and read side evolve independently. You can have multiple read models projecting from the same event stream (dashboard table, search index, analytics aggregate) without any coordination between them.

**Current implementation (V1):** Projections run synchronously within the same HTTP request via the in-process eventbus. No projection checkpoints yet — if the server crashes mid-request, neither the event nor the projection is committed. Async projection with checkpoints is a planned evolution.

### Soft-Delete Pattern

"Delete" in event-sourced verticals means "append a deletion event":

```go
// Domain: records intent
func (w *WorkItem) DeleteNote(cmd DeleteNoteCmd) ([]DomainEvent, error) {
    if w.NoteIDs[cmd.NoteID] { return nil, ErrNoteAlreadyDeleted }  // already deleted
    return []DomainEvent{{EventType: EventNoteDeletedFromTimelineEntry, ...}}, nil
}

// Apply: tracks state
case EventNoteDeletedFromTimelineEntry:
    w.NoteIDs[e.NoteID] = true  // true = deleted

// Projection: marks in read model
store.DeleteNoteOnTimeline(ctx, workItemID, noteID)  // sets deleted=true in JSONB
```

The event stream retains the full lifecycle: created → edited → deleted. The read model filters out deleted items for display. Nothing is ever physically removed from the event store.

## Structural vs Semantic Coupling

Cross-vertical coupling comes in two forms. Understanding the distinction is key to evaluating whether the architecture delivers on PIL-2 (Strong Vertical Isolation).

### Command Path: Structural Coupling

When `app_casehandling` sends a command to the workitem vertical, the coupling is purely structural:

```go
// app_casehandling handler defines its own narrow interface
type workitemCommands interface {
    ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
    AddNote(ctx context.Context, workItemID string, dto workitemfacade.AddNoteDTO) (string, error)
}
```

The consumer passes DTOs in and gets IDs or errors back. It never sees the WorkItem aggregate, `Apply()`, event streams, versions, or any internal structure. If the workitem vertical switched from event sourcing to CRUD tomorrow, the consumer's code would not change. This is genuinely low coupling — the consumer depends only on the shape of the DTO and the method signature.

### Event Path: Semantic Coupling

When `app_casehandling` projects workitem events into its read model, the coupling is semantic:

```go
// projection subscriber handles specific event types and reads their fields
eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.InboundMessageRecordedEvent) {
    store.AppendTimeline(ctx, e.WorkItemID, TimelineEntry{
        EventType:  "inbound_message",
        Content:    e.Body,
        ActorName:  lookupPartyName(e.SenderID),
        RecordedAt: e.RecordedAt,
    })
})
```

The subscriber must understand the business meaning of each event: which fields exist, what they represent, how they map to the read model. The facade event types are the public contract — like an API schema. If an event's structure changes (field renamed, semantics altered), the projection breaks.

### Why This Is Correct

This is not a flaw — it is inherent to event-driven architecture and consistent with PIL-2:

| Dimension           | Command path                         | Event path                                 |
| ------------------- | ------------------------------------ | ------------------------------------------ |
| Coupling type       | Structural (method signatures, DTOs) | Semantic (event contracts, field meanings) |
| Consumer knowledge  | "I can ask for X"                    | "I understand what X means"                |
| Change propagation  | Compile-time (interface mismatch)    | Compile-time (struct field mismatch)       |
| Isolation mechanism | Consumer-defined interfaces          | Versioned event types (`.v1` suffix)       |
| Provider freedom    | Full (can change internals)          | Constrained (event schema is public API)   |

The structural coupling on the command path is minimized by design: consumer-defined interfaces, opaque DTOs, no shared state. The semantic coupling on the event path is managed through stable, versioned event contracts. Adding a new field to an event is additive (non-breaking). Changing or removing fields requires a new version (`v2`), and both versions can coexist during migration.

This dual nature — low structural coupling with meaningful semantic coupling — is the architectural cost of event-driven projections. It is paid once per event type and amortized across every consumer that projects from the same stream.

## Archtest

The architecture test (`tools/archtest/`) enforces boundary rules at two levels:

1. **Import boundaries** — Cross-vertical imports must go through `facade/` packages only. Importing `domain/`, `infra/`, `web/`, `subscriber/`, or `testharness/` from another vertical is a violation.
2. **Type boundaries** — Cross-vertical symbol usage must be value-oriented (types, constants, sentinel errors). Concrete functions (constructors) from foreign facades are blocked — they belong in composition roots (`cmd/`), which the archtest does not check.

### Auto-Discovery

The allowlist of cross-vertical symbols is **auto-discovered at runtime** by scanning all facade packages with `go/packages`. No manual allowlist is needed. The facade package _is_ the allowlist:

- Exported **types** (structs, type aliases) → auto-allowed
- Exported **constants** → auto-allowed
- Exported **variables** with `error` type (sentinel errors) → auto-allowed
- Exported **functions** (constructors) → blocked

For type aliases (e.g., `type Status = domain.Status`), the underlying domain type is also discovered so that Go's transparent alias resolution does not cause false positives.

This means adding a new DTO, event type, or constant to any `facade/` package automatically makes it available cross-vertically. No policy file update required.

### Structural Checks

Beyond import and type boundaries, archtest enforces several structural invariants:

- **Domain purity** — Domain packages must not import infrastructure (`internal/platform/`, `database/sql`, `net/http`, database drivers). Keeps domain logic as a pure state machine with zero infrastructure coupling.
- **Event store immutability** — No UPDATE or DELETE SQL may appear in `internal/platform/eventstore/`. The event store is strictly append-only; mutations would violate the foundational invariant that the event stream is the source of truth.
- **Vertical registry** — Every directory under `internal/` must be declared in the archtest policy as a vertical or shared package. A new undeclared directory fails the build immediately, preventing it from silently escaping all boundary enforcement.
- **Subpackage convention** — Verticals may only contain recognized subpackages: `domain/`, `facade/`, `infra/`, `web/`, `subscriber/`, `testharness/`. Prevents structural drift where code ends up in packages that bypass the facade boundary.
- **Consumer-defined interfaces** — Facade packages must not export interface types (interfaces belong at the consumer site). Facade-only verticals (`party`, `subject`) are exempt since the exported interface is their public contract.
- **No direct time.Now()** — Business logic packages (`domain/`, `facade/`, `infra/`, `subscriber/`) must not reference `time.Now`. Time is injected via a `Clock` interface for testability. Uses AST analysis to catch import aliases (`import t "time"` → `t.Now()`) and function value references (`var f = time.Now`).
- **Facade write-path purity** — In event-sourced verticals, facade-local interfaces must not declare existence-check methods (`Exists*`, `Has*`, `Check*`, `IsDuplicate*`). Such methods indicate a cross-aggregate invariant that belongs in a domain service interface, not in the facade's query model.

### Modes

- **Enforce mode (default):** `go run ./tools/archtest` fails on violations.
- **Report-only mode:** set `ARCHTEST_REPORT_ONLY=1` to print violations without failing (useful during rollout/cleanup).

## Dependency Injection

No container. All dependencies wired explicitly in `cmd/server/main.go`. Dependencies flow downward: main.go → facades → services → repositories.

## Database Rules

- **No cross-vertical foreign keys** — Cross-vertical references stored as indexed UUID columns
- **Within-vertical FKs with CASCADE** — Groups and invitations reference their parent organization
- **Join tables** belong to the vertical that semantically owns the relationship

## Typed String Constants for Domain Concepts

Domain concepts with a fixed set of valid values (status, action kind, actor kind, party role) must be expressed as typed string constants — never bare `string`. This applies everywhere such a value is created, passed, or compared.

```go
// Definition (in the package that owns the concept)
type ActionKind string

const (
    ActionKindLookup ActionKind = "lookup"
    ActionKindDraft  ActionKind = "draft"
)
```

This gives compile-time safety at function boundaries (a bare `string` won't be accepted where `ActionKind` is expected), trivial JSON serialization (the underlying `string` round-trips without custom marshalers), and readable values in the event store and logs.

### Where types live

A typed constant lives in the package that semantically owns the concept:

| Type                                               | Owner                    | Why                                            |
| -------------------------------------------------- | ------------------------ | ---------------------------------------------- |
| `Status`, `ActionKind`, `DraftStatus`, `PartyRole` | `workitem/domain`        | Domain validation and aggregate state          |
| `ActorKind`                                        | `party/facade`           | Party identity concept, part of the public API |
| `ActionKind`                                       | `platform/agentworkload` | Independent definition for the port interface  |

When a type crosses a vertical boundary, the facade re-exports it via a Go type alias — one source of truth, no redundancy:

```go
// workitem/facade/types.go
type Status = domain.Status
const StatusNew = domain.StatusNew
```

Consumers use `workitemfacade.Status` and `workitemfacade.StatusNew`. The alias means the domain type and the facade type are identical — no conversion needed within the vertical.

### Read-model exception

Read-model structs (e.g., `CaseDashboardRow`, `TimelineEntry`) use plain `string` for denormalized fields like `Status` and `ActorKind`. These are projection-level values read from the database; the type safety lives at the event/command boundary where values are created. Projection subscribers convert typed event fields to strings when populating the read model: `string(e.NewStatus)`.

## Clock Injection

Business logic must never call `time.Now()` directly. Time is provided through a `Clock` interface:

```go
type Clock interface {
    Now() time.Time
}
```

Each domain package defines its own `Clock` interface (consumer-owned pattern). `internal/platform/clock/` provides `RealClock` (production) and `FixedClock` (tests). Entity constructors receive `time.Time` directly; domain services and aggregates receive a `Clock` via constructor injection.

This enables deterministic tests — a stepping clock can verify that events within a single command share a timestamp and that timestamps across commands are causally ordered.

The archtest enforces this rule via AST analysis of `domain/`, `facade/`, `infra/`, and `subscriber/` packages. It catches direct calls, aliased imports, and function value references.

## Error Handling

Application/domain paths are Go-idiomatic: return `error` values and use sentinel errors with `errors.Is()`.  
Current startup/bootstrap code still contains a few fail-fast panics in platform initialization (config/database/context helpers); treat those as bootstrap guards, not request-path behavior.
