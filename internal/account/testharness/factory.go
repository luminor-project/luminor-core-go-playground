package testharness

import (
	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"golang.org/x/crypto/bcrypt"
)

// MakeAccount creates a test account with sensible defaults.
func MakeAccount(email, password string) domain.AccountCore {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return domain.NewAccountCore(email, string(hash))
}
