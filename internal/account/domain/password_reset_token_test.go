package domain

import (
	"testing"
	"time"
)

func TestPasswordResetToken_IsValid_NotUsed_NotExpired(t *testing.T) {
	token := PasswordResetToken{
		ExpiresAt: time.Now().Add(time.Hour),
		UsedAt:    nil,
	}

	if !token.IsValid(time.Now()) {
		t.Error("expected token to be valid when not used and not expired")
	}
}

func TestPasswordResetToken_IsValid_Used(t *testing.T) {
	now := time.Now()
	usedAt := now.Add(-time.Hour)
	token := PasswordResetToken{
		ExpiresAt: now.Add(time.Hour),
		UsedAt:    &usedAt,
	}

	if token.IsValid(now) {
		t.Error("expected token to be invalid when already used")
	}
}

func TestPasswordResetToken_IsValid_Expired(t *testing.T) {
	now := time.Now()
	token := PasswordResetToken{
		ExpiresAt: now.Add(-time.Hour),
		UsedAt:    nil,
	}

	if token.IsValid(now) {
		t.Error("expected token to be invalid when expired")
	}
}

func TestPasswordResetToken_MarkAsUsed(t *testing.T) {
	token := PasswordResetToken{}
	now := time.Now()

	token.MarkAsUsed(now)

	if token.UsedAt == nil {
		t.Fatal("expected UsedAt to be set")
	}
	if !token.UsedAt.Equal(now) {
		t.Errorf("expected UsedAt to be %v, got %v", now, *token.UsedAt)
	}
}

func TestGenerateResetToken(t *testing.T) {
	plainToken, hashedToken, err := GenerateResetToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plainToken == "" {
		t.Error("expected plain token to be non-empty")
	}
	if hashedToken == "" {
		t.Error("expected hashed token to be non-empty")
	}
	if plainToken == hashedToken {
		t.Error("expected plain and hashed tokens to be different")
	}

	// Verify that hashing the plain token produces the same hash
	hash := HashToken(plainToken)
	if hash != hashedToken {
		t.Error("expected HashToken(plainToken) to equal hashedToken")
	}
}

func TestGenerateResetToken_Unique(t *testing.T) {
	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		plainToken, _, err := GenerateResetToken()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tokens[plainToken] {
			t.Error("generated duplicate token")
		}
		tokens[plainToken] = true
	}
}

func TestHashToken(t *testing.T) {
	// Test that HashToken is deterministic
	token := "test-token-123"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("expected HashToken to be deterministic")
	}

	// Test that different tokens produce different hashes
	differentHash := HashToken("different-token")
	if hash1 == differentHash {
		t.Error("expected different tokens to produce different hashes")
	}
}
