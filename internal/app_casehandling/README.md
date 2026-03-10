# Case Handling Vertical (app_casehandling)

Synthesis vertical that orchestrates workitem, party, subject, and agentic workload into UC-01: Inbound Message to Managed Response. Owns the case dashboard read model, projection subscribers, and the workbench UI.

## Architecture

```
internal/app_casehandling/
├── facade/
│   ├── dto.go             # InquiryDTO
│   └── impl.go            # Orchestration: HandleInboundInquiry, ConfirmAndSend, note delegation
├── infra/
│   └── dashboard_store.go # Read model: CaseDashboardRow, TimelineEntry, TimelineNote
├── subscriber/
│   ├── projection.go      # Eventbus subscribers projecting into read model
│   └── projection_test.go # Tests with fake store and party/subject lookups
├── testharness/
│   └── golden_path.go     # SeedGoldenPath for demo data
└── web/
    ├── handler.go          # HTTP handlers (workbench, detail partial, notes CRUD)
    ├── routes.go           # Route registration
    └── templates/
        ├── case_workbench.templ  # 2-to-3-pane workbench layout
        ├── case_detail.templ     # Detail pane with timeline
        ├── case_list.templ       # Shared helpers and icons
        └── case_notes.templ      # Notes pane (third column)
```

## How It Composes Verticals

This vertical imports **only facade packages** from other verticals:

| Dependency    | Import                   | Purpose                                                  |
| ------------- | ------------------------ | -------------------------------------------------------- |
| workitem      | `workitem/facade`        | Event-sourced commands (intake, actions, confirm, notes) |
| party         | `party/facade`           | Resolve party names and actor kinds for timeline display |
| subject       | `subject/facade`         | Resolve subject names and details for case headers       |
| agentworkload | `platform/agentworkload` | Execute AI lookup and draft actions                      |

The facade orchestrates the UC-01 flow:

```
HandleInboundInquiry:
  1. workitems.IntakeInboundMessage → creates work item + records inbound message
  2. agent.Execute(lookup)          → AI retrieves relevant data
  3. workitems.RecordAssistantAction(lookup)
  4. agent.Execute(draft)           → AI drafts response
  5. workitems.RecordAssistantAction(draft, pending)

ConfirmAndSend:
  1. workitems.ConfirmOutboundMessage → records outbound message, resolves case
```

## Read Model (Projection)

The case dashboard is a **disposable projection** maintained by eventbus subscribers. It denormalizes data from multiple event types into a single query-friendly table.

### Schema

```sql
case_dashboard (
    work_item_id TEXT PRIMARY KEY,
    status TEXT, party_name TEXT, party_actor_kind TEXT,
    subject_name TEXT, subject_detail TEXT,
    timeline_json JSONB,  -- array of TimelineEntry, each with optional Notes
    created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
)
```

### Projection Flow

| Event                   | Projection Action                               |
| ----------------------- | ----------------------------------------------- |
| WorkItemCreated         | Upsert new row with status "new"                |
| PartyLinked (sender)    | Lookup party name, update row                   |
| SubjectLinked           | Lookup subject name/detail, update row          |
| InboundMessageRecorded  | Append to timeline_json                         |
| AssistantActionRecorded | Append to timeline_json                         |
| OutboundMessageRecorded | Append to timeline_json + set status "resolved" |
| WorkItemStatusChanged   | Update status                                   |
| NoteAdded               | Add note to specific timeline entry in JSONB    |
| NoteEdited              | Update note body in JSONB                       |
| NoteDeleted             | Mark note as deleted in JSONB                   |

### Consistency Model

The projection runs synchronously within the same HTTP request. If the projection fails, the event is already stored in the workitem event stream. Recovery: delete all rows from `case_dashboard` and replay events through the projection logic.

The event store and read model are **deliberately not transactionally coupled**. See `docs/archbook.md` for the rationale.

## Web UI

The case workbench uses a **2-to-3-pane layout**:

- **Pane 1 (list):** All cases with status indicators, subject, party, timestamps, AI action counts
- **Pane 2 (detail):** Selected case header + AI summary panel + timeline with connectors
- **Pane 3 (notes, on demand):** Opens when clicking "Notes" on a timeline entry. Shows existing notes, add/edit/delete forms.

**Key techniques:**

- htmx swaps detail pane content (`hx-get`, `hx-target="#case-detail-pane"`)
- Alpine.js manages selection state and notes pane visibility (`x-data`, `x-bind:class`, `x-show`)
- Negative margins (`-my-10 -mx-4`) break out of AppShell padding for full-viewport content
- CSS grid classes toggle between 2-col and 3-col layouts (`lmn-workbench-grid` vs `lmn-workbench-grid-with-notes`)

See `assets/css/design-system/contexts/workbench.css` for all component classes, and `/living-styleguide/workbench` for a self-contained interactive demo.

## Routes

| Method | Path                                            | Handler               | Purpose                               |
| ------ | ----------------------------------------------- | --------------------- | ------------------------------------- |
| GET    | /cases                                          | ShowCaseWorkbench     | Full workbench (list + detail)        |
| GET    | /cases/{id}                                     | ShowCaseWorkbench     | Workbench with specific case selected |
| GET    | /cases/{id}/partial                             | ShowCaseDetailPartial | htmx partial for detail pane          |
| POST   | /cases/{id}/confirm                             | HandleConfirm         | Confirm and send draft                |
| GET    | /cases/{id}/entries/{entryIndex}/notes          | ShowNotesPartial      | Notes pane for timeline entry         |
| POST   | /cases/{id}/entries/{entryIndex}/notes          | HandleAddNote         | Add note                              |
| PUT    | /cases/{id}/entries/{entryIndex}/notes/{noteId} | HandleEditNote        | Edit note                             |
| DELETE | /cases/{id}/entries/{entryIndex}/notes/{noteId} | HandleDeleteNote      | Soft-delete note                      |

## Testing

```bash
# Projection subscriber tests
mise run in-app-container go test ./internal/app_casehandling/subscriber/ -v

# Full test suite
mise run in-app-container go test ./internal/app_casehandling/... -v
```
