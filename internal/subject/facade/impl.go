package facade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
)

type subjectReader interface {
	FindByID(ctx context.Context, id string) (domain.Subject, error)
	FindByIDs(ctx context.Context, ids []string) ([]domain.Subject, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]domain.Subject, error)
	FindByOrgAndKind(ctx context.Context, orgID string, kind domain.SubjectKind) ([]domain.Subject, error)
}

var _ interface {
	GetSubjectInfo(ctx context.Context, subjectID string) (SubjectInfoDTO, error)
	CreateSubject(ctx context.Context, dto CreateSubjectDTO) (string, error)
	ListSubjectsByOrg(ctx context.Context, orgID string) ([]SubjectInfoDTO, error)
	ListSubjectsByOrgAndKind(ctx context.Context, orgID string, kind SubjectKind) ([]SubjectInfoDTO, error)
	GetSubjectsByIDs(ctx context.Context, ids []string) ([]SubjectInfoDTO, error)
} = (*facadeImpl)(nil)

type facadeImpl struct {
	store     eventstore.Store
	bus       *eventbus.Bus
	clock     domain.Clock
	readModel subjectReader
}

func New(store eventstore.Store, bus *eventbus.Bus, clock domain.Clock, readModel subjectReader) *facadeImpl {
	return &facadeImpl{store: store, bus: bus, clock: clock, readModel: readModel}
}

func (f *facadeImpl) CreateSubject(ctx context.Context, dto CreateSubjectDTO) (string, error) {
	subjectID := uuid.New().String()
	streamID := "subject-" + subjectID

	s := domain.NewSubject(f.clock)
	domainEvents, err := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:          subjectID,
		SubjectKind:        domain.SubjectKind(dto.SubjectKind),
		Name:               dto.Name,
		Detail:             dto.Detail,
		OrgID:              dto.OwningOrgID,
		CreatedByAccountID: dto.CreatedByAccountID,
	})
	if err != nil {
		return "", fmt.Errorf("register subject: %w", err)
	}

	uncommitted := toUncommitted(domainEvents)
	stored, err := f.store.Append(ctx, streamID, 0, uncommitted)
	if err != nil {
		return "", fmt.Errorf("append events: %w", err)
	}

	f.publishAll(ctx, stored)
	return subjectID, nil
}

func (f *facadeImpl) GetSubjectInfo(ctx context.Context, subjectID string) (SubjectInfoDTO, error) {
	s, err := f.readModel.FindByID(ctx, subjectID)
	if err != nil {
		if errors.Is(err, domain.ErrSubjectNotFound) {
			return SubjectInfoDTO{}, ErrSubjectNotFound
		}
		return SubjectInfoDTO{}, err
	}
	return toDTO(s), nil
}

func (f *facadeImpl) ListSubjectsByOrg(ctx context.Context, orgID string) ([]SubjectInfoDTO, error) {
	subjects, err := f.readModel.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toDTOs(subjects), nil
}

func (f *facadeImpl) ListSubjectsByOrgAndKind(ctx context.Context, orgID string, kind SubjectKind) ([]SubjectInfoDTO, error) {
	subjects, err := f.readModel.FindByOrgAndKind(ctx, orgID, domain.SubjectKind(kind))
	if err != nil {
		return nil, err
	}
	return toDTOs(subjects), nil
}

func (f *facadeImpl) GetSubjectsByIDs(ctx context.Context, ids []string) ([]SubjectInfoDTO, error) {
	subjects, err := f.readModel.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toDTOs(subjects), nil
}

func toDTO(s domain.Subject) SubjectInfoDTO {
	return SubjectInfoDTO{
		ID:          s.ID,
		SubjectKind: SubjectKind(s.SubjectKind),
		Name:        s.Name,
		Detail:      s.Detail,
	}
}

func toDTOs(subjects []domain.Subject) []SubjectInfoDTO {
	result := make([]SubjectInfoDTO, len(subjects))
	for i, s := range subjects {
		result[i] = toDTO(s)
	}
	return result
}

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

func (f *facadeImpl) publishAll(ctx context.Context, stored []eventstore.StoredEvent) {
	for _, se := range stored {
		payload, err := domain.DeserializeEvent(se.EventType, se.Payload)
		if err != nil {
			slog.Error("failed to deserialize event for publishing", "event_type", se.EventType, "error", err)
			continue
		}

		var publishErr error
		switch se.EventType {
		case domain.EventSubjectRegistered:
			e := payload.(domain.SubjectRegistered)
			publishErr = eventbus.Publish(ctx, f.bus, SubjectRegisteredEvent{
				SubjectID:          e.SubjectID,
				SubjectKind:        SubjectKind(e.SubjectKind),
				Name:               e.Name,
				Detail:             e.Detail,
				OrgID:              e.OrgID,
				CreatedByAccountID: e.CreatedByAccountID,
				RegisteredAt:       e.RegisteredAt,
			})
		}

		if publishErr != nil {
			slog.Error("failed to publish event", "event_type", se.EventType, "error", publishErr)
		}
	}
}
