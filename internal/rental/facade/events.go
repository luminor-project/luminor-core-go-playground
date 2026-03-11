package facade

import "time"

type RentalEstablishedEvent struct {
	RentalID           string
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
	EstablishedAt      time.Time
}
