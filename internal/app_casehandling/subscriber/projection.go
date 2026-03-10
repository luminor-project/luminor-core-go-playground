package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

type dashboardUpdater interface {
	Upsert(ctx context.Context, row infra.CaseDashboardRow) error
	AppendTimeline(ctx context.Context, workItemID string, entry infra.TimelineEntry) error
	UpdateStatus(ctx context.Context, workItemID, status string) error
	AddNoteToTimeline(ctx context.Context, workItemID string, entryIndex int, note infra.TimelineNote) error
	EditNoteOnTimeline(ctx context.Context, workItemID, noteID, body string, editedAt time.Time) error
	DeleteNoteOnTimeline(ctx context.Context, workItemID, noteID string) error
}

type partyLookup interface {
	GetPartyInfo(ctx context.Context, partyID string) (partyfacade.PartyInfoDTO, error)
}

type subjectLookupIface interface {
	GetSubjectInfo(ctx context.Context, subjectID string) (subjectfacade.SubjectInfoDTO, error)
}

// RegisterProjectionSubscribers registers eventbus subscribers that project workitem events
// into the case dashboard read model.
func RegisterProjectionSubscribers(
	bus *eventbus.Bus,
	store dashboardUpdater,
	parties partyLookup,
	subjects subjectLookupIface,
) {
	subscribeCreationEvents(bus, store, parties, subjects)
	subscribeTimelineEvents(bus, store, parties)
	subscribeNoteEvents(bus, store, parties)
}

func subscribeCreationEvents(
	bus *eventbus.Bus,
	store dashboardUpdater,
	parties partyLookup,
	subjects subjectLookupIface,
) {
	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.WorkItemCreatedEvent) error {
		slog.Info("projecting WorkItemCreatedEvent", "work_item_id", e.WorkItemID)
		return store.Upsert(ctx, infra.CaseDashboardRow{
			WorkItemID: e.WorkItemID,
			Status:     string(workitemfacade.StatusNew),
			CreatedAt:  e.CreatedAt,
		})
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.PartyLinkedEvent) error {
		if e.Role != workitemfacade.PartyRoleSender {
			return nil
		}
		slog.Info("projecting PartyLinkedEvent (sender)", "work_item_id", e.WorkItemID, "party_id", e.PartyID)
		info, err := parties.GetPartyInfo(ctx, e.PartyID)
		if err != nil {
			return fmt.Errorf("lookup party %s: %w", e.PartyID, err)
		}
		return store.Upsert(ctx, infra.CaseDashboardRow{
			WorkItemID:     e.WorkItemID,
			PartyName:      info.Name,
			PartyActorKind: string(info.ActorKind),
		})
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.SubjectLinkedEvent) error {
		slog.Info("projecting SubjectLinkedEvent", "work_item_id", e.WorkItemID, "subject_id", e.SubjectID)
		info, err := subjects.GetSubjectInfo(ctx, e.SubjectID)
		if err != nil {
			return fmt.Errorf("lookup subject %s: %w", e.SubjectID, err)
		}
		return store.Upsert(ctx, infra.CaseDashboardRow{
			WorkItemID:    e.WorkItemID,
			SubjectName:   info.Name,
			SubjectDetail: info.Detail,
		})
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.WorkItemStatusChangedEvent) error {
		slog.Info("projecting WorkItemStatusChangedEvent", "work_item_id", e.WorkItemID, "new_status", e.NewStatus)
		return store.UpdateStatus(ctx, e.WorkItemID, string(e.NewStatus))
	})
}

func subscribeTimelineEvents(
	bus *eventbus.Bus,
	store dashboardUpdater,
	parties partyLookup,
) {
	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.InboundMessageRecordedEvent) error {
		slog.Info("projecting InboundMessageRecordedEvent", "work_item_id", e.WorkItemID)
		actorName, actorKind := resolveParty(ctx, parties, e.SenderID)
		return store.AppendTimeline(ctx, e.WorkItemID, infra.TimelineEntry{
			EventType:  "inbound_message",
			ActorName:  actorName,
			ActorKind:  actorKind,
			Content:    e.Body,
			RecordedAt: e.RecordedAt,
		})
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.AssistantActionRecordedEvent) error {
		slog.Info("projecting AssistantActionRecordedEvent", "work_item_id", e.WorkItemID, "action_kind", e.ActionKind)
		actorName, actorKind := resolveParty(ctx, parties, e.ActorID)
		return store.AppendTimeline(ctx, e.WorkItemID, infra.TimelineEntry{
			EventType:   "assistant_action_" + string(e.ActionKind),
			ActorName:   actorName,
			ActorKind:   actorKind,
			Content:     e.Output,
			DraftStatus: string(e.DraftStatus),
			RecordedAt:  e.RecordedAt,
		})
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.OutboundMessageRecordedEvent) error {
		slog.Info("projecting OutboundMessageRecordedEvent", "work_item_id", e.WorkItemID)
		actorName, actorKind := resolveParty(ctx, parties, e.ConfirmedBy)
		if err := store.AppendTimeline(ctx, e.WorkItemID, infra.TimelineEntry{
			EventType:  "outbound_message",
			ActorName:  actorName,
			ActorKind:  actorKind,
			Content:    e.Body,
			RecordedAt: e.RecordedAt,
		}); err != nil {
			return err
		}
		return store.UpdateStatus(ctx, e.WorkItemID, string(workitemfacade.StatusResolved))
	})
}

func subscribeNoteEvents(
	bus *eventbus.Bus,
	store dashboardUpdater,
	parties partyLookup,
) {
	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.NoteAddedToTimelineEntryEvent) error {
		slog.Info("projecting NoteAddedToTimelineEntryEvent", "work_item_id", e.WorkItemID, "note_id", e.NoteID, "entry_index", e.EntryIndex)
		authorName, _ := resolveParty(ctx, parties, e.AuthorID)
		return store.AddNoteToTimeline(ctx, e.WorkItemID, e.EntryIndex, infra.TimelineNote{
			NoteID:     e.NoteID,
			AuthorID:   e.AuthorID,
			AuthorName: authorName,
			Body:       e.Body,
			CreatedAt:  e.CreatedAt,
		})
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.NoteEditedOnTimelineEntryEvent) error {
		slog.Info("projecting NoteEditedOnTimelineEntryEvent", "work_item_id", e.WorkItemID, "note_id", e.NoteID)
		return store.EditNoteOnTimeline(ctx, e.WorkItemID, e.NoteID, e.Body, e.EditedAt)
	})

	eventbus.Subscribe(bus, func(ctx context.Context, e workitemfacade.NoteDeletedFromTimelineEntryEvent) error {
		slog.Info("projecting NoteDeletedFromTimelineEntryEvent", "work_item_id", e.WorkItemID, "note_id", e.NoteID)
		return store.DeleteNoteOnTimeline(ctx, e.WorkItemID, e.NoteID)
	})
}

func resolveParty(ctx context.Context, parties partyLookup, partyID string) (string, string) {
	info, err := parties.GetPartyInfo(ctx, partyID)
	if err != nil {
		slog.Warn("party lookup failed for timeline", "party_id", partyID, "error", err)
		return partyID, "unknown"
	}
	return info.Name, string(info.ActorKind)
}
