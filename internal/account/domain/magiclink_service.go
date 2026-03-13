package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// MagicLinkTokenDuration is the lifetime of a magic link token (15 minutes).
const MagicLinkTokenDuration = 15 * time.Minute

// MaxActiveMagicLinks is the maximum number of active (unused, non-expired) tokens per account.
const MaxActiveMagicLinks = 3

// MagicLinkService handles magic link token generation and validation.
type MagicLinkService struct {
	repo  Repository
	clock Clock
}

// NewMagicLinkService creates a new MagicLinkService.
func NewMagicLinkService(repo Repository, clock Clock) *MagicLinkService {
	return &MagicLinkService{repo: repo, clock: clock}
}

// GenerateToken creates a new magic link token for the given account.
// Returns the plaintext token that should be sent to the user via email.
// If there are too many active tokens for this account, old ones are invalidated.
func (s *MagicLinkService) GenerateToken(ctx context.Context, accountID string) (string, error) {
	// Check active token count
	activeCount, err := s.repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("count active tokens: %w", err)
	}

	// If at or over limit, invalidate existing tokens
	if activeCount >= MaxActiveMagicLinks {
		if err := s.repo.InvalidateExistingMagicLinkTokens(ctx, accountID); err != nil {
			return "", fmt.Errorf("invalidate existing tokens: %w", err)
		}
	}

	now := s.clock.Now()
	expiresAt := now.Add(MagicLinkTokenDuration)

	token, plaintextToken, err := NewMagicLinkToken(accountID, expiresAt, now)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	if err := s.repo.CreateMagicLinkToken(ctx, token); err != nil {
		return "", fmt.Errorf("save token: %w", err)
	}

	return plaintextToken, nil
}

// ValidateToken validates a plaintext magic link token and returns the associated account ID.
// Returns an error if the token is invalid, expired, or already used.
func (s *MagicLinkService) ValidateToken(ctx context.Context, plaintextToken string) (string, error) {
	tokenHash := hashToken(plaintextToken)

	token, err := s.repo.FindMagicLinkTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrMagicLinkNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("find token: %w", err)
	}

	now := s.clock.Now()

	// Check if already used
	if token.IsUsed() {
		return "", ErrMagicLinkUsed
	}

	// Check if expired
	if token.IsExpired(now) {
		return "", ErrMagicLinkExpired
	}

	// Mark as used
	token.MarkUsed(now)
	if err := s.repo.MarkMagicLinkTokenUsed(ctx, token.ID, now); err != nil {
		return "", fmt.Errorf("mark token used: %w", err)
	}

	return token.AccountID, nil
}
