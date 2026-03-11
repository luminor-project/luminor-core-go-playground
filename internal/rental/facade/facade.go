package facade

import "fmt"

// ErrRentalNotFound is returned when a rental is not found.
var ErrRentalNotFound = fmt.Errorf("rental not found")

// ErrDuplicateRental is returned when a rental already exists for the subject+tenant pair.
var ErrDuplicateRental = fmt.Errorf("duplicate rental")

// RentalInfoDTO holds rental data for cross-vertical communication.
type RentalInfoDTO struct {
	ID            string
	SubjectID     string
	TenantPartyID string
	OrgID         string
}

// CreateRentalDTO holds data for creating a new rental.
type CreateRentalDTO struct {
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
}
