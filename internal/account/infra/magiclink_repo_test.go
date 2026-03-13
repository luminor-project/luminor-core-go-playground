//go:build integration

package infra_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/account/infra"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://luminor:luminor@localhost:5442/luminor?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func cleanupMagicLinkTokens(t *testing.T, pool *pgxpool.Pool, accountID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "DELETE FROM magic_link_tokens WHERE account_id = $1", accountID)
	if err != nil {
		t.Fatalf("cleanup magic link tokens: %v", err)
	}
}

func TestMagicLinkRepository_CreateAndFind(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-create-find"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	// Create a token
	token, plaintext, err := domain.NewMagicLinkToken(accountID, expiresAt, now)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Store it
	if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
		t.Fatalf("failed to create magic link token: %v", err)
	}

	// Find it by hash
	tokenHash := token.TokenHash
	found, err := repo.FindMagicLinkTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("failed to find token: %v", err)
	}

	if found.ID != token.ID {
		t.Errorf("expected ID %q, got %q", token.ID, found.ID)
	}
	if found.AccountID != accountID {
		t.Errorf("expected AccountID %q, got %q", accountID, found.AccountID)
	}
	if found.TokenHash != tokenHash {
		t.Errorf("expected TokenHash %q, got %q", tokenHash, found.TokenHash)
	}
	if found.UsedAt != nil {
		t.Error("expected UsedAt to be nil for new token")
	}
}

func TestMagicLinkRepository_Find_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	// Try to find a non-existent token
	_, err := repo.FindMagicLinkTokenByHash(ctx, "non-existent-hash")
	if err != domain.ErrMagicLinkNotFound {
		t.Errorf("expected ErrMagicLinkNotFound, got %v", err)
	}
}

func TestMagicLinkRepository_MarkUsed(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-mark-used"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	// Create and store a token
	token, _, err := domain.NewMagicLinkToken(accountID, expiresAt, now)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Mark it as used
	usedAt := now.Add(1 * time.Minute)
	if err := repo.MarkMagicLinkTokenUsed(ctx, token.ID, usedAt); err != nil {
		t.Fatalf("failed to mark token used: %v", err)
	}

	// Verify it was marked as used
	found, err := repo.FindMagicLinkTokenByHash(ctx, token.TokenHash)
	if err != nil {
		t.Fatalf("failed to find token after marking used: %v", err)
	}
	if found.UsedAt == nil {
		t.Error("expected UsedAt to be set")
	} else if !found.UsedAt.Equal(usedAt) {
		t.Errorf("expected UsedAt %v, got %v", usedAt, *found.UsedAt)
	}
}

func TestMagicLinkRepository_CountActive(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-count-active"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()
	clk := clock.NewFixed(now)

	// Initially should have 0 active tokens
	count, err := repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 active tokens, got %d", count)
	}

	// Create 3 active tokens
	for i := 0; i < 3; i++ {
		token, _, err := domain.NewMagicLinkToken(accountID, now.Add(15*time.Minute), now)
		if err != nil {
			t.Fatalf("failed to create token %d: %v", i+1, err)
		}
		if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
			t.Fatalf("failed to store token %d: %v", i+1, err)
		}
	}

	// Should now have 3 active tokens
	count, err = repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 active tokens, got %d", count)
	}
}

func TestMagicLinkRepository_CountActive_ExcludesUsed(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-count-excludes-used"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()

	// Create an active token
	token1, _, err := domain.NewMagicLinkToken(accountID, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := repo.CreateMagicLinkToken(ctx, token1); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}

	// Create and mark a token as used
	token2, _, err := domain.NewMagicLinkToken(accountID, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := repo.CreateMagicLinkToken(ctx, token2); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}
	if err := repo.MarkMagicLinkTokenUsed(ctx, token2.ID, now.Add(1*time.Minute)); err != nil {
		t.Fatalf("failed to mark token used: %v", err)
	}

	// Should only have 1 active token (excludes used)
	count, err := repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 active token (excludes used), got %d", count)
	}
}

func TestMagicLinkRepository_CountActive_ExcludesExpired(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-count-excludes-expired"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()

	// Create an active token with future expiry
	token1, _, err := domain.NewMagicLinkToken(accountID, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := repo.CreateMagicLinkToken(ctx, token1); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}

	// Create a token that is already expired
	token2, _, err := domain.NewMagicLinkToken(accountID, now.Add(-1*time.Minute), now.Add(-16*time.Minute))
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := repo.CreateMagicLinkToken(ctx, token2); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}

	// Should only have 1 active token (excludes expired)
	count, err := repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 active token (excludes expired), got %d", count)
	}
}

func TestMagicLinkRepository_InvalidateExisting(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-invalidate"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()

	// Create 3 active tokens
	for i := 0; i < 3; i++ {
		token, _, err := domain.NewMagicLinkToken(accountID, now.Add(15*time.Minute), now)
		if err != nil {
			t.Fatalf("failed to create token %d: %v", i+1, err)
		}
		if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
			t.Fatalf("failed to store token %d: %v", i+1, err)
		}
	}

	// Verify we have 3 active tokens
	count, err := repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 active tokens, got %d", count)
	}

	// Invalidate all existing tokens
	if err := repo.InvalidateExistingMagicLinkTokens(ctx, accountID); err != nil {
		t.Fatalf("failed to invalidate tokens: %v", err)
	}

	// Should now have 0 active tokens
	count, err = repo.CountActiveMagicLinkTokens(ctx, accountID)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 active tokens after invalidation, got %d", count)
	}
}

func TestMagicLinkRepository_InvalidateExisting_LeavesOtherAccounts(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID1 := "test-acct-invalidate-1"
	accountID2 := "test-acct-invalidate-2"
	t.Cleanup(func() {
		cleanupMagicLinkTokens(t, pool, accountID1)
		cleanupMagicLinkTokens(t, pool, accountID2)
	})

	now := time.Now()

	// Create 2 tokens for account 1
	for i := 0; i < 2; i++ {
		token, _, err := domain.NewMagicLinkToken(accountID1, now.Add(15*time.Minute), now)
		if err != nil {
			t.Fatalf("failed to create token for account 1: %v", err)
		}
		if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
			t.Fatalf("failed to store token: %v", err)
		}
	}

	// Create 2 tokens for account 2
	for i := 0; i < 2; i++ {
		token, _, err := domain.NewMagicLinkToken(accountID2, now.Add(15*time.Minute), now)
		if err != nil {
			t.Fatalf("failed to create token for account 2: %v", err)
		}
		if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
			t.Fatalf("failed to store token: %v", err)
		}
	}

	// Invalidate tokens for account 1 only
	if err := repo.InvalidateExistingMagicLinkTokens(ctx, accountID1); err != nil {
		t.Fatalf("failed to invalidate tokens: %v", err)
	}

	// Account 1 should have 0 active tokens
	count1, err := repo.CountActiveMagicLinkTokens(ctx, accountID1)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count1 != 0 {
		t.Errorf("account 1: expected 0 active tokens, got %d", count1)
	}

	// Account 2 should still have 2 active tokens
	count2, err := repo.CountActiveMagicLinkTokens(ctx, accountID2)
	if err != nil {
		t.Fatalf("failed to count active tokens: %v", err)
	}
	if count2 != 2 {
		t.Errorf("account 2: expected 2 active tokens, got %d", count2)
	}
}

func TestMagicLinkRepository_TokenUniqueness(t *testing.T) {
	pool := testPool(t)
	repo := infra.NewPostgresRepository(pool)
	ctx := context.Background()

	accountID := "test-acct-uniqueness"
	t.Cleanup(func() { cleanupMagicLinkTokens(t, pool, accountID) })

	now := time.Now()
	seenHashes := make(map[string]bool)

	// Create many tokens and verify all hashes are unique
	for i := 0; i < 100; i++ {
		token, _, err := domain.NewMagicLinkToken(accountID, now.Add(15*time.Minute), now)
		if err != nil {
			t.Fatalf("failed to create token %d: %v", i+1, err)
		}

		if seenHashes[token.TokenHash] {
			t.Fatalf("duplicate token hash found: %s", token.TokenHash)
		}
		seenHashes[token.TokenHash] = true

		if err := repo.CreateMagicLinkToken(ctx, token); err != nil {
			t.Fatalf("failed to store token %d: %v", i+1, err)
		}
	}
}
