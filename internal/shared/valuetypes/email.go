package valuetypes

import (
	"fmt"
	"net/mail"
	"strings"
)

// EmailAddress is an immutable value type representing a validated email address.
type EmailAddress struct {
	value string
}

// NewEmailAddress creates a validated EmailAddress from a string.
// The email is trimmed and lowercased.
func NewEmailAddress(email string) (EmailAddress, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))

	if normalized == "" {
		return EmailAddress{}, fmt.Errorf("email address cannot be empty")
	}

	if _, err := mail.ParseAddress(normalized); err != nil {
		return EmailAddress{}, fmt.Errorf("invalid email address: %q", email)
	}

	return EmailAddress{value: normalized}, nil
}

// MustNewEmailAddress creates an EmailAddress or panics. Use only in tests.
func MustNewEmailAddress(email string) EmailAddress {
	e, err := NewEmailAddress(email)
	if err != nil {
		panic(err)
	}
	return e
}

// String returns the normalized email address string.
func (e EmailAddress) String() string {
	return e.value
}

// Equal returns true if two email addresses are equal.
func (e EmailAddress) Equal(other EmailAddress) bool {
	return e.value == other.value
}

// IsZero returns true if the email address is unset.
func (e EmailAddress) IsZero() bool {
	return e.value == ""
}
