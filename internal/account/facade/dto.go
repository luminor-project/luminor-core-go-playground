package facade

import "time"

// AccountInfoDTO holds account data for cross-vertical communication.
type AccountInfoDTO struct {
	ID                            string
	Email                         string
	Roles                         []string
	CreatedAt                     time.Time
	CurrentlyActiveOrganizationID string
}

// DisplayName returns a display-friendly name.
func (d AccountInfoDTO) DisplayName() string {
	return d.Email
}

// RegistrationDTO holds data for account registration.
type RegistrationDTO struct {
	Email           string
	PlainPassword   string
	MustSetPassword bool
}
