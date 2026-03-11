package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSubjectNotFound = errors.New("subject not found")
	ErrEmptyName       = errors.New("subject name must not be empty")
)

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Subject is a tracked object under management (e.g., a property).
type Subject struct {
	ID                   string
	Name                 string
	Detail               string
	OwningOrganizationID string
	CreatedByAccountID   string
	CreatedAt            time.Time
}

// NewSubject creates a new subject with a generated UUID and validated fields.
func NewSubject(name, detail, orgID, createdByAccountID string, now time.Time) (Subject, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Subject{}, ErrEmptyName
	}

	return Subject{
		ID:                   uuid.New().String(),
		Name:                 trimmed,
		Detail:               strings.TrimSpace(detail),
		OwningOrganizationID: orgID,
		CreatedByAccountID:   createdByAccountID,
		CreatedAt:            now,
	}, nil
}
