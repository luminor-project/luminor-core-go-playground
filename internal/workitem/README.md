# WorkItem Vertical

Event-sourced aggregate for managing work items — cases, tickets, or any entity with a timeline of actions. This is a **framework capability vertical**: it provides the event-sourced infrastructure that domain-specific verticals (like `app_casehandling`) compose into business workflows.

## Architecture

```
internal/workitem/
├── domain/
│   ├── events.go          # Event type constants + payload structs
│   ├── workitem.go        # WorkItem aggregate (Apply + command methods)
│   └── serialization.go   # JSON deserialization for event payloads
├── facade/
│   ├── dto.go             # DTOs for facade methods
│   ├── events.go          # Published event types (consumed via eventbus)
│   └── impl.go            # Facade: load-execute-save-publish cycle
└── testharness/
    └── factory.go         # Test fixtures with golden-path defaults
```

## Domain Model

The WorkItem aggregate is a **pure state machine** reconstituted from events. It has zero infrastructure imports — only `time` and `errors`.

### State

| Field              | Type            | Purpose                                                         |
| ------------------ | --------------- | --------------------------------------------------------------- |
| ID                 | string          | Work item identifier                                            |
| Status             | string          | `new`, `in_progress`, `pending_confirmation`, `resolved`        |
| Version            | int             | Event count (used for optimistic concurrency)                   |
| PartyIDs           | []string        | Linked party identifiers                                        |
| SubjectID          | string          | Linked subject identifier                                       |
| Created            | bool            | Idempotency guard for intake                                    |
| HasPendingDraft    | bool            | Whether an AI draft awaits confirmation                         |
| Confirmed          | bool            | Whether outbound message was confirmed                          |
| TimelineEntryCount | int             | Number of timeline-producing events (for note index validation) |
| NoteIDs            | map[string]bool | Note existence and deletion tracking (noteID → deleted?)        |

### Events (10 types)

**Lifecycle:** `WorkItemCreated`, `PartyLinkedToWorkItem`, `SubjectLinkedToWorkItem`, `WorkItemStatusChanged`

**Timeline:** `InboundMessageRecorded`, `AssistantActionRecorded`, `OutboundMessageRecorded`

**Notes:** `NoteAddedToTimelineEntry`, `NoteEditedOnTimelineEntry`, `NoteDeletedFromTimelineEntry`

All events are versioned (e.g., `workitem.WorkItemCreated.v1`) and stored as JSONB in the event store.

### Commands

| Command                | Produces                                         | Validation                                 |
| ---------------------- | ------------------------------------------------ | ------------------------------------------ |
| IntakeInboundMessage   | Created + links + InboundMessage + StatusChanged | Must not already be created                |
| RecordAssistantAction  | AssistantAction (+ StatusChanged if draft)       | Must exist, not resolved                   |
| ConfirmOutboundMessage | OutboundMessage + StatusChanged(resolved)        | Must have pending draft, not yet confirmed |
| AddNote                | NoteAdded                                        | Must exist, entry index in range           |
| EditNote               | NoteEdited                                       | Note must exist and not be deleted         |
| DeleteNote             | NoteDeleted                                      | Note must exist and not be deleted         |

### Key Design Decisions

**Commands return events, never mutate state.** The aggregate's command methods (`AddNote`, `EditNote`, etc.) validate against current state and return `[]DomainEvent`. They never modify aggregate fields directly. Only `Apply()` mutates state.

**DomainEvent is domain-local.** The `DomainEvent` struct (`EventType string, Payload any`) lives in the domain package. The facade converts these to `eventstore.UncommittedEvent` before appending. This keeps the domain at zero platform imports.

**ID generation lives in the facade.** The facade generates UUIDs for work items and notes, then passes them into commands. The domain never touches `uuid` or any other infrastructure package.

**TimelineEntryCount enables note validation.** The aggregate tracks how many timeline entries exist (inbound messages, assistant actions, outbound messages). `AddNote` validates that the `EntryIndex` is within this range. This is an example of the aggregate maintaining just enough state for command validation.

## Persistence

The workitem vertical uses the **platform event store** (`internal/platform/eventstore/`), not a repository. There is no workitem table — all state is derived from the event stream `workitem-{id}`.

The facade's load-execute-save-publish cycle:

1. **Load:** `eventstore.LoadStream("workitem-{id}")` → `[]StoredEvent`
2. **Replay:** Deserialize each event, call `aggregate.Apply()` to reconstitute state
3. **Execute:** Call command method → `[]DomainEvent` (validated against current state)
4. **Save:** Convert to `[]UncommittedEvent`, call `eventstore.Append()` with expected version
5. **Publish:** Deserialize stored events, convert to facade event types, publish to eventbus

## Testing

```bash
# Domain unit tests (pure, no DB)
mise run in-app-container go test ./internal/workitem/domain/ -v

# Full golden path tested via app_casehandling integration
mise run in-app-container go test ./internal/app_casehandling/... -v
```
