package valuetypes_test

import (
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/shared/valuetypes"
)

func TestNewEmailAddress_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "user@example.com"},
		{"  User@Example.COM  ", "user@example.com"},
		{"ADMIN@TEST.ORG", "admin@test.org"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			email, err := valuetypes.NewEmailAddress(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if email.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, email.String())
			}
		})
	}
}

func TestNewEmailAddress_Invalid(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"not-an-email",
		"@example.com",
		"user@",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := valuetypes.NewEmailAddress(input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestEmailAddress_Equal(t *testing.T) {
	a := valuetypes.MustNewEmailAddress("user@example.com")
	b := valuetypes.MustNewEmailAddress("USER@EXAMPLE.COM")
	c := valuetypes.MustNewEmailAddress("other@example.com")

	if !a.Equal(b) {
		t.Error("expected a and b to be equal")
	}
	if a.Equal(c) {
		t.Error("expected a and c to not be equal")
	}
}

func TestEmailAddress_IsZero(t *testing.T) {
	var zero valuetypes.EmailAddress
	if !zero.IsZero() {
		t.Error("expected zero value to be zero")
	}

	email := valuetypes.MustNewEmailAddress("user@example.com")
	if email.IsZero() {
		t.Error("expected non-zero email to not be zero")
	}
}
