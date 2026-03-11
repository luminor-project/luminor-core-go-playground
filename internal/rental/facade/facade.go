package facade

import (
	"fmt"

	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

// ErrRentalNotFound is returned when a rental is not found.
var ErrRentalNotFound = fmt.Errorf("rental not found")

// ErrDuplicateRental is the domain error for duplicate subject+tenant rentals.
var ErrDuplicateRental = domain.ErrDuplicateRental

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
