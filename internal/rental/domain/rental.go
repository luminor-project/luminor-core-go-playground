package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRentalNotFound  = errors.New("rental not found")
	ErrDuplicateRental = errors.New("rental already exists for this subject and tenant")
)

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Rental links a tenant party to a subject (property) they rent.
type Rental struct {
	ID                 string
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
	CreatedAt          time.Time
}

// NewRental creates a new rental with a generated UUID.
func NewRental(subjectID, tenantPartyID, orgID, createdByAccountID string, now time.Time) Rental {
	return Rental{
		ID:                 uuid.New().String(),
		SubjectID:          subjectID,
		TenantPartyID:      tenantPartyID,
		OrgID:              orgID,
		CreatedByAccountID: createdByAccountID,
		CreatedAt:          now,
	}
}
