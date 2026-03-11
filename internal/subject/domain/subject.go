package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrSubjectNotFound    = errors.New("subject not found")
	ErrInvalidSubjectKind = errors.New("invalid subject kind")
	ErrEmptyName          = errors.New("subject name must not be empty")
	ErrAlreadyRegistered  = errors.New("subject already registered")
)

type SubjectKind string

const (
	SubjectKindDwelling SubjectKind = "dwelling"
)

func ValidSubjectKinds() []SubjectKind {
	return []SubjectKind{SubjectKindDwelling}
}

func IsValidSubjectKind(k SubjectKind) bool {
	for _, v := range ValidSubjectKinds() {
		if v == k {
			return true
		}
	}
	return false
}

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Subject is the event-sourced aggregate for tracked objects under management.
type Subject struct {
	ID                   string
	SubjectKind          SubjectKind
	Name                 string
	Detail               string
	OwningOrganizationID string
	CreatedByAccountID   string
	CreatedAt            time.Time
	Registered           bool
	Version              int
	clock                Clock
}

// NewSubject creates a Subject aggregate with the given clock.
func NewSubject(clock Clock) *Subject {
	return &Subject{clock: clock}
}

// Apply reconstitutes state from a single event payload.
func (s *Subject) Apply(eventType string, payload any) {
	switch eventType {
	case EventSubjectRegistered:
		e := payload.(SubjectRegistered)
		s.ID = e.SubjectID
		s.SubjectKind = e.SubjectKind
		s.Name = e.Name
		s.Detail = e.Detail
		s.OwningOrganizationID = e.OrgID
		s.CreatedByAccountID = e.CreatedByAccountID
		s.CreatedAt = e.RegisteredAt
		s.Registered = true
	default:
		panic("subject.Apply: unknown event type: " + eventType)
	}
	s.Version++
}

// RegisterSubjectCmd holds the data needed to register a new subject.
type RegisterSubjectCmd struct {
	SubjectID          string
	SubjectKind        SubjectKind
	Name               string
	Detail             string
	OrgID              string
	CreatedByAccountID string
}

// RegisterSubject registers a new subject entity.
func (s *Subject) RegisterSubject(cmd RegisterSubjectCmd) ([]DomainEvent, error) {
	if s.Registered {
		return nil, ErrAlreadyRegistered
	}
	if !IsValidSubjectKind(cmd.SubjectKind) {
		return nil, ErrInvalidSubjectKind
	}
	trimmed := strings.TrimSpace(cmd.Name)
	if trimmed == "" {
		return nil, ErrEmptyName
	}

	return []DomainEvent{
		{EventType: EventSubjectRegistered, Payload: SubjectRegistered{
			SubjectID:          cmd.SubjectID,
			SubjectKind:        cmd.SubjectKind,
			Name:               trimmed,
			Detail:             strings.TrimSpace(cmd.Detail),
			OrgID:              cmd.OrgID,
			CreatedByAccountID: cmd.CreatedByAccountID,
			RegisteredAt:       s.clock.Now(),
		}},
	}, nil
}
