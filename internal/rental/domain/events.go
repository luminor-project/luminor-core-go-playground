package domain

import "time"

const (
	EventRentalEstablished = "rental.RentalEstablished.v1"
)

type DomainEvent struct {
	EventType string
	Payload   any
}

type RentalEstablished struct {
	RentalID           string    `json:"rental_id"`
	SubjectID          string    `json:"subject_id"`
	TenantPartyID      string    `json:"tenant_party_id"`
	OrgID              string    `json:"org_id"`
	CreatedByAccountID string    `json:"created_by_account_id"`
	EstablishedAt      time.Time `json:"established_at"`
}
