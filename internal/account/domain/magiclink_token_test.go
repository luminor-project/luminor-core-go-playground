package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

var magicLinkTestClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

func TestNewMagicLinkToken_Success(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, plaintext, err := domain.NewMagicLinkToken(accountID, expiresAt, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.ID == "" {
		t.Error("expected non-empty token ID")
	}
	if token.AccountID != accountID {
		t.Errorf("expected accountID %q, got %q", accountID, token.AccountID)
	}
	if token.TokenHash == "" {
		t.Error("expected non-empty token hash")
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected expiresAt %v, got %v", expiresAt, token.ExpiresAt)
	}
	if !token.CreatedAt.Equal(now) {
		t.Errorf("expected createdAt %v, got %v", now, token.CreatedAt)
	}
	if token.UsedAt != nil {
		t.Error("expected UsedAt to be nil for new token")
	}
	if plaintext == "" {
		t.Error("expected non-empty plaintext token")
	}
	if len(plaintext) < 32 {
		t.Errorf("expected plaintext token to be at least 32 chars, got %d", len(plaintext))
	}
}

func TestNewMagicLinkToken_GeneratesUniqueTokens(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		_, plaintext, err := domain.NewMagicLinkToken(accountID, expiresAt, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[plaintext] {
			t.Fatalf("duplicate token generated: %s", plaintext)
		}
		seen[plaintext] = true
	}
}

func TestMagicLinkToken_IsExpired(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, _, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	tests := []struct {
		name     string
		checkAt  time.Time
		expected bool
	}{
		{
			name:     "not expired at creation time",
			checkAt:  now,
			expected: false,
		},
		{
			name:     "not expired 1 minute before expiry",
			checkAt:  expiresAt.Add(-1 * time.Minute),
			expected: false,
		},
		{
			name:     "expired at expiry time (boundary)",
			checkAt:  expiresAt,
			expected: false, // After, not At
		},
		{
			name:     "expired 1 second after expiry",
			checkAt:  expiresAt.Add(1 * time.Second),
			expected: true,
		},
		{
			name:     "expired 1 minute after expiry",
			checkAt:  expiresAt.Add(1 * time.Minute),
			expected: true,
		},
		{
			name:     "expired long after expiry",
			checkAt:  now.Add(1 * time.Hour),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := token.IsExpired(tt.checkAt)
			if got != tt.expected {
				t.Errorf("IsExpired(%v) = %v, want %v", tt.checkAt, got, tt.expected)
			}
		})
	}
}

func TestMagicLinkToken_IsUsed(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, _, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	if token.IsUsed() {
		t.Error("new token should not be marked as used")
	}

	usedAt := now.Add(1 * time.Minute)
	token.MarkUsed(usedAt)

	if !token.IsUsed() {
		t.Error("token should be marked as used after MarkUsed()")
	}
	if token.UsedAt == nil {
		t.Error("UsedAt should not be nil after MarkUsed()")
	}
	if !token.UsedAt.Equal(usedAt) {
		t.Errorf("UsedAt = %v, want %v", *token.UsedAt, usedAt)
	}
}

func TestMagicLinkToken_ValidateToken_Success(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, plaintext, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	if !token.ValidateToken(plaintext) {
		t.Error("ValidateToken should return true for correct plaintext")
	}
}

func TestMagicLinkToken_ValidateToken_Failure(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, plaintext, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	tests := []struct {
		name          string
		inputToken    string
		shouldBeValid bool
	}{
		{
			name:          "completely different token",
			inputToken:    "totally-different-token-string-here",
			shouldBeValid: false,
		},
		{
			name:          "empty token",
			inputToken:    "",
			shouldBeValid: false,
		},
		{
			name:          "token with one char different",
			inputToken:    plaintext[:len(plaintext)-1] + "X",
			shouldBeValid: false,
		},
		{
			name:          "token with extra char",
			inputToken:    plaintext + "X",
			shouldBeValid: false,
		},
		{
			name:          "token missing last char",
			inputToken:    plaintext[:len(plaintext)-1],
			shouldBeValid: false,
		},
		{
			name:          "similar but different token",
			inputToken:    plaintext[:5] + "X" + plaintext[6:],
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := token.ValidateToken(tt.inputToken)
			if got != tt.shouldBeValid {
				t.Errorf("ValidateToken(%q) = %v, want %v", tt.inputToken, got, tt.shouldBeValid)
			}
		})
	}
}

func TestMagicLinkToken_ValidateToken_SameHashForSameInput(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token1, plaintext, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	// Create another token with the same plaintext - should have same hash
	token2, _, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)
	token2.TokenHash = token1.TokenHash // Copy the hash

	// Both should validate the same plaintext
	if !token1.ValidateToken(plaintext) {
		t.Error("token1 should validate the plaintext")
	}
	if !token2.ValidateToken(plaintext) {
		t.Error("token2 should also validate the same plaintext")
	}
}

func TestMagicLinkToken_StoredHashNotReversible(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, plaintext, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	// The stored hash should not contain or resemble the plaintext
	if strings.Contains(token.TokenHash, plaintext) {
		t.Error("token hash should not contain the plaintext token")
	}
	if len(token.TokenHash) != 44 {
		// SHA-256 produces 32 bytes, base64 encoded is 44 chars with padding
		t.Errorf("expected hash length of 44 (base64 encoded SHA-256), got %d", len(token.TokenHash))
	}
}

func TestMagicLinkToken_DifferentAccountsHaveDifferentHashes(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)

	// Probability of collision is astronomically low, but let's verify
	account1Hashes := make(map[string]bool)
	account2Hashes := make(map[string]bool)

	for i := 0; i < 10; i++ {
		token1, _, _ := domain.NewMagicLinkToken("acct-1", expiresAt, now)
		token2, _, _ := domain.NewMagicLinkToken("acct-2", expiresAt, now)

		account1Hashes[token1.TokenHash] = true
		account2Hashes[token2.TokenHash] = true
	}

	// No overlap between accounts
	for hash := range account1Hashes {
		if account2Hashes[hash] {
			t.Error("found hash collision between different accounts")
		}
	}
}

func TestMagicLinkToken_MarkUsed_DoesNotAffectValidation(t *testing.T) {
	t.Parallel()

	now := magicLinkTestClock.Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, plaintext, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	// Mark as used
	usedAt := now.Add(1 * time.Minute)
	token.MarkUsed(usedAt)

	// Validation should still work (business logic handles single-use enforcement)
	if !token.ValidateToken(plaintext) {
		t.Error("ValidateToken should still work even after token is marked as used")
	}
}
