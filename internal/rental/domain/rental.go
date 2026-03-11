package domain

import (
	"errors"
	"time"
)

var (
	ErrRentalNotFound     = errors.New("rental not found")
	ErrDuplicateRental    = errors.New("rental already exists for this subject and tenant")
	ErrAlreadyEstablished = errors.New("rental already established")
)

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Rental is the event-sourced aggregate linking a tenant party to a subject.
type Rental struct {
	ID                 string
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
	Established        bool
	Version            int
	clock              Clock
}

// NewRental creates a Rental aggregate with the given clock.
func NewRental(clock Clock) *Rental {
	return &Rental{clock: clock}
}

// Apply reconstitutes state from a single event payload.
func (r *Rental) Apply(eventType string, payload any) {
	switch eventType {
	case EventRentalEstablished:
		e := payload.(RentalEstablished)
		r.ID = e.RentalID
		r.SubjectID = e.SubjectID
		r.TenantPartyID = e.TenantPartyID
		r.OrgID = e.OrgID
		r.CreatedByAccountID = e.CreatedByAccountID
		r.Established = true
	default:
		panic("rental.Apply: unknown event type: " + eventType)
	}
	r.Version++
}

// EstablishRentalCmd holds the data needed to establish a new rental.
type EstablishRentalCmd struct {
	RentalID           string
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
}

// EstablishRental creates a new rental relationship.
func (r *Rental) EstablishRental(cmd EstablishRentalCmd) ([]DomainEvent, error) {
	if r.Established {
		return nil, ErrAlreadyEstablished
	}

	return []DomainEvent{
		{EventType: EventRentalEstablished, Payload: RentalEstablished{
			RentalID:           cmd.RentalID,
			SubjectID:          cmd.SubjectID,
			TenantPartyID:      cmd.TenantPartyID,
			OrgID:              cmd.OrgID,
			CreatedByAccountID: cmd.CreatedByAccountID,
			EstablishedAt:      r.clock.Now(),
		}},
	}, nil
}
