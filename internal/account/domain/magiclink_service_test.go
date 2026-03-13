package domain_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

// mockMagicLinkRepository wraps mockRepository and adds magic link storage.
type mockMagicLinkRepository struct {
	*mockRepository
	magicLinkTokens map[string]domain.MagicLinkToken // keyed by token ID
	mu              sync.RWMutex
	clock           domain.Clock
}

func newMockMagicLinkRepo() *mockMagicLinkRepository {
	return &mockMagicLinkRepository{
		mockRepository:  newMockRepo(),
		magicLinkTokens: make(map[string]domain.MagicLinkToken),
	}
}

func newMockMagicLinkRepoWithClock(clk domain.Clock) *mockMagicLinkRepository {
	return &mockMagicLinkRepository{
		mockRepository:  newMockRepo(),
		magicLinkTokens: make(map[string]domain.MagicLinkToken),
		clock:           clk,
	}
}

func (m *mockMagicLinkRepository) now() time.Time {
	if m.clock != nil {
		return m.clock.Now()
	}
	return time.Now()
}

func (m *mockMagicLinkRepository) CreateMagicLinkToken(_ context.Context, token domain.MagicLinkToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.magicLinkTokens[token.ID] = token
	return nil
}

func (m *mockMagicLinkRepository) FindMagicLinkTokenByHash(_ context.Context, tokenHash string) (domain.MagicLinkToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, token := range m.magicLinkTokens {
		if token.TokenHash == tokenHash {
			return token, nil
		}
	}
	return domain.MagicLinkToken{}, domain.ErrMagicLinkNotFound
}

func (m *mockMagicLinkRepository) MarkMagicLinkTokenUsed(_ context.Context, tokenID string, usedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	token, ok := m.magicLinkTokens[tokenID]
	if !ok {
		return errors.New("token not found")
	}
	token.MarkUsed(usedAt)
	m.magicLinkTokens[tokenID] = token
	return nil
}

func (m *mockMagicLinkRepository) CountActiveMagicLinkTokens(_ context.Context, accountID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	now := m.now()
	for _, token := range m.magicLinkTokens {
		if token.AccountID == accountID && !token.IsUsed() && !token.IsExpired(now) {
			count++
		}
	}
	return count, nil
}

func (m *mockMagicLinkRepository) InvalidateExistingMagicLinkTokens(_ context.Context, accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	for id, token := range m.magicLinkTokens {
		if token.AccountID == accountID && !token.IsUsed() {
			token.MarkUsed(now)
			m.magicLinkTokens[id] = token
		}
	}
	return nil
}

// Test MagicLinkService.GenerateToken
func TestMagicLinkService_GenerateToken_Success(t *testing.T) {
	t.Parallel()

	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	plaintext, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plaintext == "" {
		t.Error("expected non-empty plaintext token")
	}

	// Verify the token was stored by finding it
	// We need to find the token by iterating since we can't compute the hash directly
	var foundToken domain.MagicLinkToken
	for _, t := range repo.magicLinkTokens {
		if t.AccountID == accountID {
			foundToken = t
			break
		}
	}
	if foundToken.ID == "" {
		t.Fatal("token was not stored in repository")
	}
	if foundToken.AccountID != accountID {
		t.Errorf("expected accountID %q, got %q", accountID, foundToken.AccountID)
	}
}

func TestMagicLinkService_GenerateToken_MultipleTokensAllowed(t *testing.T) {
	t.Parallel()

	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate up to MaxActiveMagicLinks tokens
	for i := 0; i < domain.MaxActiveMagicLinks; i++ {
		_, err := svc.GenerateToken(ctx, accountID)
		if err != nil {
			t.Fatalf("failed to generate token %d: %v", i+1, err)
		}
	}

	// Count should be exactly MaxActiveMagicLinks
	count, err := repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count tokens: %v", err)
	}
	if count != domain.MaxActiveMagicLinks {
		t.Errorf("expected %d active tokens, got %d", domain.MaxActiveMagicLinks, count)
	}
}

func TestMagicLinkService_GenerateToken_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate MaxActiveMagicLinks tokens
	for i := 0; i < domain.MaxActiveMagicLinks; i++ {
		_, err := svc.GenerateToken(ctx, accountID)
		if err != nil {
			t.Fatalf("failed to generate token %d: %v", i+1, err)
		}
	}

	// Generate one more - should invalidate old ones first
	_, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to generate token after rate limit: %v", err)
	}

	// Old tokens should be invalidated (marked as used), so we should have 1 active
	count, err := repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count tokens: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 active token after rate limit, got %d", count)
	}
}

func TestMagicLinkService_GenerateToken_DifferentAccountsIndependent(t *testing.T) {
	t.Parallel()

	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	ctx := context.Background()

	// Generate max tokens for account 1
	for i := 0; i < domain.MaxActiveMagicLinks; i++ {
		_, err := svc.GenerateToken(ctx, "acct-1")
		if err != nil {
			t.Fatalf("failed to generate token for acct-1: %v", err)
		}
	}

	// Generate max tokens for account 2 - should work independently
	for i := 0; i < domain.MaxActiveMagicLinks; i++ {
		_, err := svc.GenerateToken(ctx, "acct-2")
		if err != nil {
			t.Fatalf("failed to generate token for acct-2: %v", err)
		}
	}

	// Both accounts should have MaxActiveMagicLinks active tokens
	count1, _ := repo.CountActiveMagicLinkTokens(ctx, "acct-1")
	count2, _ := repo.CountActiveMagicLinkTokens(ctx, "acct-2")

	if count1 != domain.MaxActiveMagicLinks {
		t.Errorf("acct-1: expected %d active tokens, got %d", domain.MaxActiveMagicLinks, count1)
	}
	if count2 != domain.MaxActiveMagicLinks {
		t.Errorf("acct-2: expected %d active tokens, got %d", domain.MaxActiveMagicLinks, count2)
	}
}

// Test MagicLinkService.ValidateToken
func TestMagicLinkService_ValidateToken_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewFixed(now)
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate a token
	plaintext, err := svc.GenerateToken(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate it immediately
	validatedAccountID, err := svc.ValidateToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validatedAccountID != accountID {
		t.Errorf("expected accountID %q, got %q", accountID, validatedAccountID)
	}
}

func TestMagicLinkService_ValidateToken_MarksAsUsed(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewFixed(now)
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate and validate a token
	plaintext, _ := svc.GenerateToken(ctx, accountID)
	_, err := svc.ValidateToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("first validation failed: %v", err)
	}

	// Try to validate again - should fail because it's used
	_, err = svc.ValidateToken(ctx, plaintext)
	if !errors.Is(err, domain.ErrMagicLinkUsed) {
		t.Errorf("expected ErrMagicLinkUsed, got %v", err)
	}
}

func TestMagicLinkService_ValidateToken_Expired(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewFixed(now)
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate a token
	plaintext, _ := svc.GenerateToken(ctx, accountID)

	// Advance time past expiration (15 minutes + 1 second)
	expiredClk := clock.NewFixed(now.Add(domain.MagicLinkTokenDuration + 1*time.Second))
	expiredSvc := domain.NewMagicLinkService(repo, expiredClk)

	// Try to validate - should fail because it's expired
	_, err := expiredSvc.ValidateToken(ctx, plaintext)
	if !errors.Is(err, domain.ErrMagicLinkExpired) {
		t.Errorf("expected ErrMagicLinkExpired, got %v", err)
	}
}

func TestMagicLinkService_ValidateToken_InvalidToken(t *testing.T) {
	t.Parallel()

	clk := clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	ctx := context.Background()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "random token",
			token: "this-is-a-random-token-that-does-not-exist",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "valid format but non-existent",
			token: "aHR0cHM6Ly9leGFtcGxlLmNvbS9tYWdpYy1saW5r", // base64 URL encoded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateToken(ctx, tt.token)
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("expected ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

func TestMagicLinkService_ValidateToken_BoundaryExpiry(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	clk := clock.NewFixed(now)
	repo := newMockMagicLinkRepoWithClock(clk)
	svc := domain.NewMagicLinkService(repo, clk)

	accountID := "acct-123"
	ctx := context.Background()

	// Generate a token
	plaintext, _ := svc.GenerateToken(ctx, accountID)

	// Validate at exactly 15 minutes (not expired yet due to IsExpired using After)
	exactExpiryClk := clock.NewFixed(now.Add(domain.MagicLinkTokenDuration))
	exactExpirySvc := domain.NewMagicLinkService(repo, exactExpiryClk)

	_, err := exactExpirySvc.ValidateToken(ctx, plaintext)
	// Should work at exact expiry time (boundary condition)
	if err != nil {
		t.Errorf("expected success at exact expiry time, got %v", err)
	}
}

// Verify mockMagicLinkRepository implements the full Repository interface.
var _ domain.Repository = (*mockMagicLinkRepository)(nil)
