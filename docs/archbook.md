# Architecture Book

## Vertical Slice Architecture

The application is organized into self-contained verticals. Each vertical owns its complete stack from domain logic to HTTP handlers and templates.

### Verticals

**CRUD-based:**

- **account** — User authentication, profile management, session handling
- **organization** — Multi-tenancy, teams, groups, invitations, access rights
- **content** — Public pages (homepage, about, living styleguide)

**Event-sourced:**

- **workitem** — Event-sourced WorkItem aggregate (cases, timeline, notes). No repository — domain is a pure state machine reconstituted from events.
- **party** — Party identity facade (human and AI actors). Demo data for V1.
- **subject** — Subject info facade (properties, units). Demo data for V1.

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

A facade method must justify its existence by one of two criteria:

1. **Cross-vertical public API** — The method is called by another vertical (or by a composition root like `cmd/server` or `cmd/seed`).
2. **Multi-vertical orchestration** — The method coordinates calls across two or more verticals or infrastructure ports (e.g., workitem + agent workload + subject lookup).

A method that simply delegates to a single dependency **does not belong on the facade**. If the only caller is the vertical's own handler, the handler should consume the dependency's facade directly via its own narrow interface.

**Example: app_casehandling**

`HandleInboundInquiry` orchestrates workitem intake + agent lookup + agent draft + two assistant action recordings across three dependencies. This is genuine multi-vertical orchestration — it belongs on the facade.

`ConfirmOutboundMessage`, `AddNote`, `EditNote`, `DeleteNote` — each of these would only delegate to the workitem facade with no additional logic. They do not belong on the app_casehandling facade. The handler consumes the workitem facade directly:

```go
// Handler uses workitem facade directly for single-vertical operations
type workitemCommands interface {
    ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
    AddNote(ctx context.Context, workItemID string, dto workitemfacade.AddNoteDTO) (string, error)
}
```

This keeps facades lean, avoids pure-delegation bloat, and makes it immediately clear which methods involve genuine orchestration.

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

Events table: `events (id, stream_id, stream_version, event_type, payload JSONB, recorded_at)` with a unique constraint on `(stream_id, stream_version)`.

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

### Load-Execute-Save-Publish Cycle

The facade orchestrates the full cycle:

```
1. Load:    event store → LoadStream(streamID) → []StoredEvent
2. Replay:  for each stored event → DeserializeEvent → aggregate.Apply()
3. Execute: aggregate.CommandMethod(cmd) → []DomainEvent (or error)
4. Save:    DomainEvent → UncommittedEvent → event store.Append(streamID, version, events)
5. Publish: StoredEvent → deserialize → facade event type → eventbus.Publish()
```

This cycle is identical for every command — `IntakeInboundMessage`, `RecordAssistantAction`, `ConfirmOutboundMessage`, `AddNote`, `EditNote`, `DeleteNote` all follow the same five steps. Adding a new command means: define events, add Apply cases, write the command method, add the facade method (same plumbing), and extend publishAll.

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

## Archtest Modes

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

## Error Handling

Application/domain paths are Go-idiomatic: return `error` values and use sentinel errors with `errors.Is()`.  
Current startup/bootstrap code still contains a few fail-fast panics in platform initialization (config/database/context helpers); treat those as bootstrap guards, not request-path behavior.
