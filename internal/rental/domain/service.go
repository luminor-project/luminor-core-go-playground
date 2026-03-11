package domain

import (
	"context"
	"fmt"
)

// DuplicateChecker checks whether a rental already exists for a subject+tenant pair.
type DuplicateChecker interface {
	ExistsBySubjectAndTenant(ctx context.Context, subjectID, tenantPartyID string) (bool, error)
}

// EstablishNewRental encapsulates the full business rule for creating a rental:
// a property may have at most one active rental per tenant.
func EstablishNewRental(ctx context.Context, checker DuplicateChecker, clock Clock, cmd EstablishRentalCmd) ([]DomainEvent, error) {
	exists, err := checker.ExistsBySubjectAndTenant(ctx, cmd.SubjectID, cmd.TenantPartyID)
	if err != nil {
		return nil, fmt.Errorf("check duplicate rental: %w", err)
	}
	if exists {
		return nil, ErrDuplicateRental
	}

	r := NewRental(clock)
	return r.EstablishRental(cmd)
}
