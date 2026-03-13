package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

func TestGenerateMagicLinkToken(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clk := clock.NewFixed(now)

	token, rawToken, err := domain.GenerateMagicLinkToken("account-123", clk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.ID == "" {
		t.Error("expected non-empty token ID")
	}
	if token.AccountID != "account-123" {
		t.Errorf("expected account ID 'account-123', got %q", token.AccountID)
	}
	if token.TokenHash == "" {
		t.Error("expected non-empty token hash")
	}
	if !token.ExpiresAt.Equal(now.Add(domain.MagicLinkTokenLifetime)) {
		t.Errorf("expected expires at %v, got %v", now.Add(domain.MagicLinkTokenLifetime), token.ExpiresAt)
	}
	if !token.CreatedAt.Equal(now) {
		t.Errorf("expected created at %v, got %v", now, token.CreatedAt)
	}
	if rawToken == "" {
		t.Error("expected non-empty raw token")
	}

	// Verify token hash is derived from raw token
	expectedHash := domain.HashToken(rawToken)
	if token.TokenHash != expectedHash {
		t.Error("token hash does not match hash of raw token")
	}
}

func TestMagicLinkToken_IsValid(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		token     domain.MagicLinkToken
		wantValid bool
	}{
		{
			name: "valid unused token",
			token: domain.MagicLinkToken{
				ExpiresAt: now.Add(10 * time.Minute),
				UsedAt:    nil,
			},
			wantValid: true,
		},
		{
			name: "expired token",
			token: domain.MagicLinkToken{
				ExpiresAt: now.Add(-5 * time.Minute),
				UsedAt:    nil,
			},
			wantValid: false,
		},
		{
			name: "already used token",
			token: domain.MagicLinkToken{
				ExpiresAt: now.Add(10 * time.Minute),
				UsedAt:    func() *time.Time { t := now.Add(-5 * time.Minute); return &t }(),
			},
			wantValid: false,
		},
		{
			name: "expired and used token",
			token: domain.MagicLinkToken{
				ExpiresAt: now.Add(-5 * time.Minute),
				UsedAt:    func() *time.Time { t := now.Add(-10 * time.Minute); return &t }(),
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.IsValid(now)
			if got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestAccountService_RequestMagicLink(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &mockRepoWithMagicLink{accounts: make(map[string]domain.AccountCore), tokens: make(map[string]domain.MagicLinkToken)}
	clk := clock.NewFixed(now)
	svc := domain.NewAccountService(repo, clk)

	// Create an account first
	account, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	t.Run("successful request", func(t *testing.T) {
		token, rawToken, err := svc.RequestMagicLink(context.Background(), account.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.ID == "" {
			t.Error("expected non-empty token ID")
		}
		if rawToken == "" {
			t.Error("expected non-empty raw token")
		}
		if token.AccountID != account.ID {
			t.Errorf("expected account ID %q, got %q", account.ID, token.AccountID)
		}
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		// The first subtest already created 1 token, so we can only make 2 more before hitting the limit
		for i := 0; i < 2; i++ {
			_, _, err := svc.RequestMagicLink(context.Background(), account.ID)
			if err != nil {
				t.Fatalf("request %d failed: %v", i+1, err)
			}
		}

		// 4th request (1 from first subtest + 2 above + this one) should fail with rate limit
		_, _, err := svc.RequestMagicLink(context.Background(), account.ID)
		if !errors.Is(err, domain.ErrMagicLinkRateLimitExceeded) {
			t.Errorf("expected ErrMagicLinkRateLimitExceeded, got %v", err)
		}
	})
}

func TestAccountService_VerifyMagicLink(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &mockRepoWithMagicLink{accounts: make(map[string]domain.AccountCore), tokens: make(map[string]domain.MagicLinkToken)}
	clk := clock.NewFixed(now)
	svc := domain.NewAccountService(repo, clk)

	// Create an account first
	account, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		token, rawToken, err := svc.RequestMagicLink(context.Background(), account.ID)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}
		repo.tokens[token.TokenHash] = token // Store the token

		accountID, err := svc.VerifyMagicLink(context.Background(), rawToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if accountID != account.ID {
			t.Errorf("expected account ID %q, got %q", account.ID, accountID)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.VerifyMagicLink(context.Background(), "invalid-token")
		if !errors.Is(err, domain.ErrMagicLinkTokenExpired) {
			t.Errorf("expected ErrMagicLinkTokenExpired, got %v", err)
		}
	})

	t.Run("already used token", func(t *testing.T) {
		token, rawToken, err := svc.RequestMagicLink(context.Background(), account.ID)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}
		repo.tokens[token.TokenHash] = token

		// First use should succeed
		_, err = svc.VerifyMagicLink(context.Background(), rawToken)
		if err != nil {
			t.Fatalf("first use should succeed: %v", err)
		}

		// Mark token as used
		now := clk.Now()
		token.UsedAt = &now
		repo.tokens[token.TokenHash] = token

		// Second use should fail
		_, err = svc.VerifyMagicLink(context.Background(), rawToken)
		if !errors.Is(err, domain.ErrMagicLinkTokenExpired) {
			t.Errorf("expected ErrMagicLinkTokenExpired for reused token, got %v", err)
		}
	})
}

// mockRepoWithMagicLink extends mockRepository with magic link support
type mockRepoWithMagicLink struct {
	accounts map[string]domain.AccountCore
	tokens   map[string]domain.MagicLinkToken // keyed by token hash
}

func (m *mockRepoWithMagicLink) FindByID(_ context.Context, id string) (domain.AccountCore, error) {
	a, ok := m.accounts[id]
	if !ok {
		return domain.AccountCore{}, domain.ErrAccountNotFound
	}
	return a, nil
}

func (m *mockRepoWithMagicLink) FindByEmail(_ context.Context, email string) (domain.AccountCore, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return a, nil
		}
	}
	return domain.AccountCore{}, domain.ErrAccountNotFound
}

func (m *mockRepoWithMagicLink) Create(_ context.Context, account domain.AccountCore) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockRepoWithMagicLink) Update(_ context.Context, account domain.AccountCore) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockRepoWithMagicLink) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepoWithMagicLink) ExistsByID(_ context.Context, id string) (bool, error) {
	_, ok := m.accounts[id]
	return ok, nil
}

func (m *mockRepoWithMagicLink) FindByIDs(_ context.Context, ids []string) ([]domain.AccountCore, error) {
	var result []domain.AccountCore
	for _, id := range ids {
		if a, ok := m.accounts[id]; ok {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepoWithMagicLink) ExecuteInTx(_ context.Context, fn func(repo domain.Repository) error) error {
	return fn(m)
}

func (m *mockRepoWithMagicLink) CreatePartyMembership(_ context.Context, membership domain.PartyMembership) error {
	return nil
}

func (m *mockRepoWithMagicLink) FindPartyMembershipsByAccountAndOrg(_ context.Context, accountID, orgID string) ([]domain.PartyMembership, error) {
	return nil, nil
}

func (m *mockRepoWithMagicLink) ExistsPartyMembership(_ context.Context, accountID, partyID string) (bool, error) {
	return false, nil
}

func (m *mockRepoWithMagicLink) FindAccountIDsByPartyID(_ context.Context, partyID string) ([]string, error) {
	return nil, nil
}

func (m *mockRepoWithMagicLink) CreatePendingPartyLink(_ context.Context, link domain.PendingPartyLink) error {
	return nil
}

func (m *mockRepoWithMagicLink) FindPendingPartyLinkByInvitationID(_ context.Context, invitationID string) (domain.PendingPartyLink, error) {
	return domain.PendingPartyLink{}, domain.ErrPendingLinkNotFound
}

func (m *mockRepoWithMagicLink) DeletePendingPartyLink(_ context.Context, id string) error {
	return nil
}

func (m *mockRepoWithMagicLink) CreateMagicLinkToken(_ context.Context, token domain.MagicLinkToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockRepoWithMagicLink) FindMagicLinkTokenByHash(_ context.Context, tokenHash string) (domain.MagicLinkToken, error) {
	token, ok := m.tokens[tokenHash]
	if !ok {
		return domain.MagicLinkToken{}, domain.ErrMagicLinkTokenNotFound
	}
	return token, nil
}

func (m *mockRepoWithMagicLink) MarkMagicLinkTokenUsed(_ context.Context, tokenID string, usedAt time.Time) error {
	for hash, token := range m.tokens {
		if token.ID == tokenID {
			if token.UsedAt != nil {
				return domain.ErrMagicLinkTokenAlreadyUsed
			}
			token.UsedAt = &usedAt
			m.tokens[hash] = token
			return nil
		}
	}
	return domain.ErrMagicLinkTokenNotFound
}

func (m *mockRepoWithMagicLink) CountRecentMagicLinkTokensForAccount(_ context.Context, accountID string, since time.Time) (int, error) {
	count := 0
	for _, token := range m.tokens {
		if token.AccountID == accountID && token.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}
