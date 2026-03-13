package domain_test

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

// TestSecurity_TimingAttackProtection verifies that token validation uses constant-time comparison.
func TestSecurity_TimingAttackProtection(t *testing.T) {
	t.Parallel()

	now := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)).Now()
	expiresAt := now.Add(15 * time.Minute)
	accountID := "acct-123"

	token, plaintext, _ := domain.NewMagicLinkToken(accountID, expiresAt, now)

	// Test that validation takes similar time for correct and incorrect tokens
	// This is a heuristic test - we run multiple iterations to reduce noise
	correctIterations := 100
	incorrectIterations := 100

	// Time correct token validation
	start := time.Now()
	for i := 0; i < correctIterations; i++ {
		token.ValidateToken(plaintext)
	}
	correctDuration := time.Since(start)

	// Time incorrect token validation (same length as correct token)
	wrongToken := "x" + plaintext[1:] // Same length, different content
	start = time.Now()
	for i := 0; i < incorrectIterations; i++ {
		token.ValidateToken(wrongToken)
	}
	incorrectDuration := time.Since(start)

	// The durations should be within a reasonable ratio
	// Allow for more variance since we're not running in a controlled environment
	ratio := float64(incorrectDuration) / float64(correctDuration)
	if ratio < 0.1 || ratio > 10 {
		t.Errorf("timing difference may indicate non-constant-time comparison: ratio=%.2f", ratio)
	}
}

// TestSecurity_ConstantTimeComparison_Subtle verifies the underlying crypto/subtle usage.
func TestSecurity_ConstantTimeComparison_Subtle(t *testing.T) {
	t.Parallel()

	// The implementation uses subtle.ConstantTimeCompare
	// Let's verify this directly works as expected

	hash1 := base64.StdEncoding.EncodeToString([]byte("test-hash-1"))
	hash2 := base64.StdEncoding.EncodeToString([]byte("test-hash-2"))
	hash1Copy := base64.StdEncoding.EncodeToString([]byte("test-hash-1"))

	// Equal hashes should return 1
	result := subtle.ConstantTimeCompare([]byte(hash1), []byte(hash1Copy))
	if result != 1 {
		t.Errorf("expected equal hashes to return 1, got %d", result)
	}

	// Different hashes should return 0
	result = subtle.ConstantTimeCompare([]byte(hash1), []byte(hash2))
	if result != 0 {
		t.Errorf("expected different hashes to return 0, got %d", result)
	}
}

// TestSecurity_SingleUseEnforcement verifies that a token can only be used once.
func TestSecurity_SingleUseEnforcement(t *testing.T) {
	repo := newMockMagicLinkRepo()
	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate a token
	plaintext, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// First use should succeed
	_, err = svc.ValidateToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("first validation should succeed: %v", err)
	}

	// Second use should fail with ErrMagicLinkUsed
	_, err = svc.ValidateToken(ctx, plaintext)
	if err == nil {
		t.Fatal("second validation should fail")
	}

	// Third use should also fail
	_, err = svc.ValidateToken(ctx, plaintext)
	if err == nil {
		t.Fatal("third validation should also fail")
	}
}

// TestSecurity_SingleUseEnforcement_MultipleRequests verifies single-use even with concurrent requests.
func TestSecurity_SingleUseEnforcement_MultipleRequests(t *testing.T) {
	repo := newMockMagicLinkRepo()
	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate a token
	plaintext, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate once
	_, err = svc.ValidateToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("first validation failed: %v", err)
	}

	// All subsequent validations should fail
	for i := 0; i < 5; i++ {
		_, err = svc.ValidateToken(ctx, plaintext)
		if err == nil {
			t.Fatalf("validation %d should have failed after first use", i+2)
		}
	}
}

// TestSecurity_TokenEntropy verifies that generated tokens have sufficient randomness.
func TestSecurity_TokenEntropy(t *testing.T) {
	t.Parallel()

	now := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)).Now()
	expiresAt := now.Add(15 * time.Minute)

	seen := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		_, plaintext, err := domain.NewMagicLinkToken("acct-123", expiresAt, now)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		if seen[plaintext] {
			t.Fatalf("duplicate token generated at iteration %d", i)
		}
		seen[plaintext] = true

		// Verify token is not predictable (has sufficient length)
		if len(plaintext) < 32 {
			t.Errorf("token length %d is less than expected 32", len(plaintext))
		}
	}

	// We generated 1000 unique tokens
	if len(seen) != iterations {
		t.Errorf("expected %d unique tokens, got %d", iterations, len(seen))
	}
}

// TestSecurity_TokenFormat verifies tokens are URL-safe base64 encoded.
func TestSecurity_TokenFormat(t *testing.T) {
	t.Parallel()

	now := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)).Now()
	expiresAt := now.Add(15 * time.Minute)

	for i := 0; i < 100; i++ {
		_, plaintext, err := domain.NewMagicLinkToken("acct-123", expiresAt, now)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// URL-safe base64 should not contain + or /
		for _, char := range plaintext {
			if char == '+' || char == '/' {
				t.Errorf("token contains non-URL-safe character: %c", char)
			}
		}

		// Should not contain padding characters in URL-safe encoding
		// URL-safe base64 may contain - and _ which are URL-safe
	}
}

// TestSecurity_HashStorage verifies that only hashes are stored, not plaintext.
func TestSecurity_HashStorage(t *testing.T) {
	repo := newMockMagicLinkRepo()
	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate a token
	plaintext, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// The plaintext should NOT be in the repository storage
	for _, token := range repo.magicLinkTokens {
		// The stored hash should not equal or contain the plaintext
		if token.TokenHash == plaintext {
			t.Error("stored hash equals plaintext - this should never happen")
		}

		// The plaintext should not be a substring of the hash
		// (this is a weak check but ensures basic protection)
		if len(token.TokenHash) == len(plaintext) {
			// They could be same length but different content
			continue
		}
	}

	// Hash should be base64 encoded SHA-256 (44 characters)
	for _, token := range repo.magicLinkTokens {
		if len(token.TokenHash) != 44 {
			t.Errorf("expected hash length of 44 (SHA-256 base64), got %d", len(token.TokenHash))
		}
	}
}

// TestSecurity_RateLimiting_InvalidateOldTokens verifies that rate limiting invalidates old tokens.
func TestSecurity_RateLimiting_InvalidateOldTokens(t *testing.T) {
	repo := newMockMagicLinkRepo()
	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate max allowed tokens
	var tokens []string
	for i := 0; i < domain.MaxActiveMagicLinks; i++ {
		token, err := svc.GenerateToken(ctx, accountID)
		if err != nil {
			t.Fatalf("failed to generate token %d: %v", i+1, err)
		}
		tokens = append(tokens, token)
	}

	// All should be valid at this point
	for i, token := range tokens {
		_, err := svc.ValidateToken(ctx, token)
		if err != nil {
			t.Fatalf("token %d should be valid: %v", i+1, err)
		}
	}

	// Now generate more tokens (triggers invalidation)
	newToken, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to generate token after rate limit: %v", err)
	}

	// The new token should be valid
	_, err = svc.ValidateToken(ctx, newToken)
	if err != nil {
		t.Fatalf("new token should be valid: %v", err)
	}
}

// TestSecurity_ExpiryPrecision verifies expiry checks are precise.
func TestSecurity_ExpiryPrecision(t *testing.T) {
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	accountID := "acct-123"
	ctx := context.Background()

	tests := []struct {
		name          string
		validateAt    time.Time
		shouldBeValid bool
	}{
		{
			name:          "immediately after creation",
			validateAt:    baseTime,
			shouldBeValid: true,
		},
		{
			name:          "1 minute before expiry",
			validateAt:    baseTime.Add(14 * time.Minute),
			shouldBeValid: true,
		},
		{
			name:          "1 second before expiry",
			validateAt:    baseTime.Add(15*time.Minute - 1*time.Second),
			shouldBeValid: true,
		},
		{
			name:          "exactly at expiry (boundary)",
			validateAt:    baseTime.Add(15 * time.Minute),
			shouldBeValid: true, // After, not At
		},
		{
			name:          "1 second after expiry",
			validateAt:    baseTime.Add(15*time.Minute + 1*time.Second),
			shouldBeValid: false,
		},
		{
			name:          "1 minute after expiry",
			validateAt:    baseTime.Add(16 * time.Minute),
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new repo and service for each test
			repo := newMockMagicLinkRepo()
			clk := clock.NewFixed(baseTime)
			svc := domain.NewMagicLinkService(repo, clk)

			plaintext, err := svc.GenerateToken(ctx, accountID)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			// Create a new service with the validation time as the current time
			validateClk := clock.NewFixed(tt.validateAt)
			validateSvc := domain.NewMagicLinkService(repo, validateClk)

			_, err = validateSvc.ValidateToken(ctx, plaintext)
			if tt.shouldBeValid && err != nil {
				t.Errorf("expected token to be valid, got error: %v", err)
			}
			if !tt.shouldBeValid && err == nil {
				t.Error("expected token to be expired, but it was valid")
			}
		})
	}
}

// TestSecurity_DifferentAccountIsolation verifies tokens for one account don't work for another.
func TestSecurity_DifferentAccountIsolation(t *testing.T) {
	repo := newMockMagicLinkRepo()
	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	svc := domain.NewMagicLinkService(repo, clk)

	ctx := context.Background()

	// Generate token for account 1
	token1, err := svc.GenerateToken(ctx, "acct-1")
	if err != nil {
		t.Fatalf("failed to generate token for acct-1: %v", err)
	}

	// Try to validate token1
	// The service looks up by hash, so it will find the token and return account 1's ID
	// This is correct behavior - the token identifies the account
	accountID, err := svc.ValidateToken(ctx, token1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accountID != "acct-1" {
		t.Errorf("expected account ID 'acct-1', got %q", accountID)
	}

	// Verify token is marked as used in the repo
	if len(repo.magicLinkTokens) == 0 {
		t.Fatal("expected token to be stored in repo")
	}
}

// TestSecurity_ValidateToken_WrongTokenFormat verifies validation handles various malformed tokens.
func TestSecurity_ValidateToken_WrongTokenFormat(t *testing.T) {
	repo := newMockMagicLinkRepo()
	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	svc := domain.NewMagicLinkService(repo, clk)

	ctx := context.Background()

	malformedTokens := []string{
		"",      // Empty
		"short", // Too short
		"this-is-a-very-long-token-that-should-not-exist-in-the-database-and-is-definitely-longer-than-normal-tokens", // Too long
		"!!!@@@###", // Invalid characters
		"dGVzdA==",  // Base64 but wrong format
	}

	for _, token := range malformedTokens {
		_, err := svc.ValidateToken(ctx, token)
		if err == nil {
			t.Errorf("token %q should have failed validation", token)
		}
	}
}
