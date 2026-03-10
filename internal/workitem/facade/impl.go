package facade

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	"github.com/luminor-project/luminor-core-go-playground/internal/workitem/domain"
)

// Compile-time interface assertion.
var _ interface {
	IntakeInboundMessage(ctx context.Context, dto IntakeInboundMessageDTO) (string, error)
	RecordAssistantAction(ctx context.Context, workItemID string, dto RecordAssistantActionDTO) error
	ConfirmOutboundMessage(ctx context.Context, workItemID string, dto ConfirmOutboundMessageDTO) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	store eventstore.Store
	bus   *eventbus.Bus
}

// New creates a new workitem facade.
func New(store eventstore.Store, bus *eventbus.Bus) *facadeImpl {
	return &facadeImpl{
		store: store,
		bus:   bus,
	}
}

// IntakeInboundMessage creates a new work item and records the initial inbound message.
func (f *facadeImpl) IntakeInboundMessage(ctx context.Context, dto IntakeInboundMessageDTO) (string, error) {
	workItemID := uuid.New().String()
	streamID := "workitem-" + workItemID

	wi := &domain.WorkItem{}
	domainEvents, err := wi.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     workItemID,
		SenderPartyID:  dto.SenderPartyID,
		SubjectID:      dto.SubjectID,
		Body:           dto.Body,
		HandlerPartyID: dto.HandlerPartyID,
		AgentPartyID:   dto.AgentPartyID,
	})
	if err != nil {
		return "", fmt.Errorf("intake inbound message: %w", err)
	}

	uncommitted := toUncommitted(domainEvents)
	stored, err := f.store.Append(ctx, streamID, 0, uncommitted)
	if err != nil {
		return "", fmt.Errorf("append events: %w", err)
	}

	f.publishAll(ctx, stored)

	return workItemID, nil
}

// RecordAssistantAction records an AI assistant action on an existing work item.
func (f *facadeImpl) RecordAssistantAction(ctx context.Context, workItemID string, dto RecordAssistantActionDTO) error {
	streamID := "workitem-" + workItemID
	wi, version, err := f.loadAggregate(ctx, streamID)
	if err != nil {
		return err
	}

	domainEvents, err := wi.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID:  workItemID,
		ActorID:     dto.ActorID,
		ActionKind:  dto.ActionKind,
		Output:      dto.Output,
		DraftStatus: dto.DraftStatus,
	})
	if err != nil {
		return fmt.Errorf("record assistant action: %w", err)
	}

	uncommitted := toUncommitted(domainEvents)
	stored, err := f.store.Append(ctx, streamID, version, uncommitted)
	if err != nil {
		return fmt.Errorf("append events: %w", err)
	}

	f.publishAll(ctx, stored)

	return nil
}

// ConfirmOutboundMessage confirms an outbound message on an existing work item.
func (f *facadeImpl) ConfirmOutboundMessage(ctx context.Context, workItemID string, dto ConfirmOutboundMessageDTO) error {
	streamID := "workitem-" + workItemID
	wi, version, err := f.loadAggregate(ctx, streamID)
	if err != nil {
		return err
	}

	domainEvents, err := wi.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID:  workItemID,
		ConfirmedBy: dto.ConfirmedByPartyID,
		Body:        dto.Body,
	})
	if err != nil {
		return fmt.Errorf("confirm outbound message: %w", err)
	}

	uncommitted := toUncommitted(domainEvents)
	stored, err := f.store.Append(ctx, streamID, version, uncommitted)
	if err != nil {
		return fmt.Errorf("append events: %w", err)
	}

	f.publishAll(ctx, stored)

	return nil
}

// loadAggregate loads a work item aggregate from its event stream.
func (f *facadeImpl) loadAggregate(ctx context.Context, streamID string) (*domain.WorkItem, int, error) {
	storedEvents, err := f.store.LoadStream(ctx, streamID)
	if err != nil {
		return nil, 0, fmt.Errorf("load stream %s: %w", streamID, err)
	}

	wi := &domain.WorkItem{}
	for _, se := range storedEvents {
		payload, err := domain.DeserializeEvent(se.EventType, se.Payload)
		if err != nil {
			return nil, 0, fmt.Errorf("deserialize event %s: %w", se.EventType, err)
		}
		wi.Apply(se.EventType, payload)
	}

	return wi, len(storedEvents), nil
}

// toUncommitted converts domain events to uncommitted events for the event store.
func toUncommitted(domainEvents []domain.DomainEvent) []eventstore.UncommittedEvent {
	uncommitted := make([]eventstore.UncommittedEvent, len(domainEvents))
	for i, de := range domainEvents {
		uncommitted[i] = eventstore.UncommittedEvent{
			EventType: de.EventType,
			Payload:   de.Payload,
		}
	}
	return uncommitted
}

// publishAll publishes stored events to the eventbus as facade event types.
func (f *facadeImpl) publishAll(ctx context.Context, stored []eventstore.StoredEvent) {
	for _, se := range stored {
		payload, err := domain.DeserializeEvent(se.EventType, se.Payload)
		if err != nil {
			slog.Error("failed to deserialize event for publishing", "event_type", se.EventType, "error", err)
			continue
		}

		var publishErr error
		switch se.EventType {
		case domain.EventWorkItemCreated:
			e := payload.(domain.WorkItemCreated)
			publishErr = eventbus.Publish(ctx, f.bus, WorkItemCreatedEvent{
				WorkItemID: e.WorkItemID,
				CreatedAt:  e.CreatedAt,
			})
		case domain.EventPartyLinked:
			e := payload.(domain.PartyLinkedToWorkItem)
			publishErr = eventbus.Publish(ctx, f.bus, PartyLinkedEvent{
				WorkItemID: e.WorkItemID,
				PartyID:    e.PartyID,
				Role:       e.Role,
			})
		case domain.EventSubjectLinked:
			e := payload.(domain.SubjectLinkedToWorkItem)
			publishErr = eventbus.Publish(ctx, f.bus, SubjectLinkedEvent{
				WorkItemID: e.WorkItemID,
				SubjectID:  e.SubjectID,
			})
		case domain.EventInboundMessageRecorded:
			e := payload.(domain.InboundMessageRecorded)
			publishErr = eventbus.Publish(ctx, f.bus, InboundMessageRecordedEvent{
				WorkItemID: e.WorkItemID,
				SenderID:   e.SenderID,
				Body:       e.Body,
				RecordedAt: e.RecordedAt,
			})
		case domain.EventAssistantActionRecorded:
			e := payload.(domain.AssistantActionRecorded)
			publishErr = eventbus.Publish(ctx, f.bus, AssistantActionRecordedEvent{
				WorkItemID:  e.WorkItemID,
				ActorID:     e.ActorID,
				ActionKind:  e.ActionKind,
				Output:      e.Output,
				DraftStatus: e.DraftStatus,
				RecordedAt:  e.RecordedAt,
			})
		case domain.EventOutboundMessageRecorded:
			e := payload.(domain.OutboundMessageRecorded)
			publishErr = eventbus.Publish(ctx, f.bus, OutboundMessageRecordedEvent{
				WorkItemID:  e.WorkItemID,
				ConfirmedBy: e.ConfirmedBy,
				Body:        e.Body,
				RecordedAt:  e.RecordedAt,
			})
		case domain.EventWorkItemStatusChanged:
			e := payload.(domain.WorkItemStatusChanged)
			publishErr = eventbus.Publish(ctx, f.bus, WorkItemStatusChangedEvent{
				WorkItemID: e.WorkItemID,
				OldStatus:  e.OldStatus,
				NewStatus:  e.NewStatus,
			})
		}

		if publishErr != nil {
			slog.Error("failed to publish event", "event_type", se.EventType, "error", publishErr)
		}
	}
}
