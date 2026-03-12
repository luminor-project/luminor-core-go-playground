package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"golang.org/x/crypto/bcrypt"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

// mockRepository is an in-memory repository for testing.
type mockRepository struct {
	accounts     map[string]domain.AccountCore
	memberships  []domain.PartyMembership
	pendingLinks map[string]domain.PendingPartyLink   // keyed by ID
	resetTokens  map[string]domain.PasswordResetToken // keyed by token hash
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		accounts:     make(map[string]domain.AccountCore),
		memberships:  nil,
		pendingLinks: make(map[string]domain.PendingPartyLink),
		resetTokens:  make(map[string]domain.PasswordResetToken),
	}
}

func (m *mockRepository) FindByID(_ context.Context, id string) (domain.AccountCore, error) {
	a, ok := m.accounts[id]
	if !ok {
		return domain.AccountCore{}, domain.ErrAccountNotFound
	}
	return a, nil
}

func (m *mockRepository) FindByEmail(_ context.Context, email string) (domain.AccountCore, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return a, nil
		}
	}
	return domain.AccountCore{}, domain.ErrAccountNotFound
}

func (m *mockRepository) Create(_ context.Context, account domain.AccountCore) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockRepository) Update(_ context.Context, account domain.AccountCore) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) ExistsByID(_ context.Context, id string) (bool, error) {
	_, ok := m.accounts[id]
	return ok, nil
}

func (m *mockRepository) FindByIDs(_ context.Context, ids []string) ([]domain.AccountCore, error) {
	var result []domain.AccountCore
	for _, id := range ids {
		if a, ok := m.accounts[id]; ok {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) ExecuteInTx(_ context.Context, fn func(repo domain.Repository) error) error {
	return fn(m)
}

func (m *mockRepository) CreatePartyMembership(_ context.Context, membership domain.PartyMembership) error {
	for _, existing := range m.memberships {
		if existing.AccountID == membership.AccountID && existing.PartyID == membership.PartyID {
			return domain.ErrAlreadyLinked
		}
	}
	m.memberships = append(m.memberships, membership)
	return nil
}

func (m *mockRepository) FindPartyMembershipsByAccountAndOrg(_ context.Context, accountID, orgID string) ([]domain.PartyMembership, error) {
	var result []domain.PartyMembership
	for _, pm := range m.memberships {
		if pm.AccountID == accountID && pm.OrgID == orgID {
			result = append(result, pm)
		}
	}
	return result, nil
}

func (m *mockRepository) ExistsPartyMembership(_ context.Context, accountID, partyID string) (bool, error) {
	for _, pm := range m.memberships {
		if pm.AccountID == accountID && pm.PartyID == partyID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) FindAccountIDsByPartyID(_ context.Context, partyID string) ([]string, error) {
	var result []string
	for _, pm := range m.memberships {
		if pm.PartyID == partyID {
			result = append(result, pm.AccountID)
		}
	}
	return result, nil
}

func (m *mockRepository) CreatePendingPartyLink(_ context.Context, link domain.PendingPartyLink) error {
	m.pendingLinks[link.ID] = link
	return nil
}

func (m *mockRepository) FindPendingPartyLinkByInvitationID(_ context.Context, invitationID string) (domain.PendingPartyLink, error) {
	for _, link := range m.pendingLinks {
		if link.InvitationID == invitationID {
			return link, nil
		}
	}
	return domain.PendingPartyLink{}, domain.ErrPendingLinkNotFound
}

func (m *mockRepository) DeletePendingPartyLink(_ context.Context, id string) error {
	delete(m.pendingLinks, id)
	return nil
}

func (m *mockRepository) SavePasswordResetToken(_ context.Context, token domain.PasswordResetToken) error {
	m.resetTokens[token.TokenHash] = token
	return nil
}

func (m *mockRepository) FindPasswordResetToken(_ context.Context, tokenHash string) (domain.PasswordResetToken, error) {
	token, ok := m.resetTokens[tokenHash]
	if !ok {
		return domain.PasswordResetToken{}, domain.ErrPasswordResetTokenInvalid
	}
	return token, nil
}

func (m *mockRepository) DeletePasswordResetToken(_ context.Context, tokenHash string) error {
	delete(m.resetTokens, tokenHash)
	return nil
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", account.Email)
	}
	if account.ID == "" {
		t.Error("expected non-empty ID")
	}
	if !account.HasRole(domain.RoleUser) {
		t.Error("expected user role")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = svc.Register(context.Background(), "test@example.com", "password456")
	if !errors.Is(err, domain.ErrEmailAlreadyTaken) {
		t.Errorf("expected ErrEmailAlreadyTaken, got %v", err)
	}
}

func TestRegister_NormalizesEmail(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, err := svc.Register(context.Background(), "  Test@Example.COM  ", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected normalized email, got %q", account.Email)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	account, err := svc.Authenticate(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", account.Email)
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	_, err = svc.Authenticate(context.Background(), "test@example.com", "wrong")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticate_NonexistentUser(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Authenticate(context.Background(), "nonexistent@example.com", "password123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestSetPassword(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "old_password")

	err := svc.SetPassword(context.Background(), account.ID, "new_password")
	if err != nil {
		t.Fatalf("set password failed: %v", err)
	}

	// Old password should no longer work
	_, err = svc.Authenticate(context.Background(), "test@example.com", "old_password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Error("old password should no longer work")
	}

	// New password should work
	_, err = svc.Authenticate(context.Background(), "test@example.com", "new_password")
	if err != nil {
		t.Errorf("new password should work, got: %v", err)
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "short")
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestSetPassword_PasswordTooShort(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.SetPassword(context.Background(), account.ID, "short")
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestSetActiveOrganization(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.SetActiveOrganization(context.Background(), account.ID, "org-123")
	if err != nil {
		t.Fatalf("set active org failed: %v", err)
	}

	updated, _ := svc.FindByID(context.Background(), account.ID)
	if updated.CurrentlyActiveOrganizationID != "org-123" {
		t.Errorf("expected active org 'org-123', got %q", updated.CurrentlyActiveOrganizationID)
	}
}

func TestSetActiveParty_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.SetActiveParty(context.Background(), account.ID, "party-42")
	if err != nil {
		t.Fatalf("set active party failed: %v", err)
	}

	updated, _ := svc.FindByID(context.Background(), account.ID)
	if updated.CurrentlyActivePartyID != "party-42" {
		t.Errorf("expected active party 'party-42', got %q", updated.CurrentlyActivePartyID)
	}
}

func TestLinkPartyToAccount_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.LinkPartyToAccount(context.Background(), account.ID, "party-1", "org-1")
	if err != nil {
		t.Fatalf("link party failed: %v", err)
	}

	memberships, err := svc.GetPartyMembershipsForAccount(context.Background(), account.ID, "org-1")
	if err != nil {
		t.Fatalf("get memberships failed: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if memberships[0].PartyID != "party-1" {
		t.Errorf("expected party 'party-1', got %q", memberships[0].PartyID)
	}
	if memberships[0].AccountID != account.ID {
		t.Errorf("expected account %q, got %q", account.ID, memberships[0].AccountID)
	}
}

func TestLinkPartyToAccount_AlreadyLinked(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.LinkPartyToAccount(context.Background(), account.ID, "party-1", "org-1")
	if err != nil {
		t.Fatalf("first link failed: %v", err)
	}

	err = svc.LinkPartyToAccount(context.Background(), account.ID, "party-1", "org-1")
	if !errors.Is(err, domain.ErrAlreadyLinked) {
		t.Errorf("expected ErrAlreadyLinked, got %v", err)
	}
}

func TestGetPartyMembershipsForAccount_ScopedToOrg(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	_ = svc.LinkPartyToAccount(context.Background(), account.ID, "party-1", "org-1")
	_ = svc.LinkPartyToAccount(context.Background(), account.ID, "party-2", "org-2")

	memberships, err := svc.GetPartyMembershipsForAccount(context.Background(), account.ID, "org-1")
	if err != nil {
		t.Fatalf("get memberships failed: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership for org-1, got %d", len(memberships))
	}
	if memberships[0].PartyID != "party-1" {
		t.Errorf("expected party-1, got %q", memberships[0].PartyID)
	}
}

func TestGetAccountIDsForParty(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	acct1, _ := svc.Register(context.Background(), "a@example.com", "password123")
	acct2, _ := svc.Register(context.Background(), "b@example.com", "password123")

	_ = svc.LinkPartyToAccount(context.Background(), acct1.ID, "party-shared", "org-1")
	_ = svc.LinkPartyToAccount(context.Background(), acct2.ID, "party-shared", "org-1")

	ids, err := svc.GetAccountIDsForParty(context.Background(), "party-shared")
	if err != nil {
		t.Fatalf("get account IDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 account IDs, got %d", len(ids))
	}
}

func TestCreatePendingPartyLink(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	link, err := svc.CreatePendingPartyLink(context.Background(), "inv-1", "party-1", "org-1")
	if err != nil {
		t.Fatalf("create pending link failed: %v", err)
	}
	if link.ID == "" {
		t.Error("expected non-empty ID")
	}
	if link.InvitationID != "inv-1" {
		t.Errorf("expected invitation 'inv-1', got %q", link.InvitationID)
	}
	if link.PartyID != "party-1" {
		t.Errorf("expected party 'party-1', got %q", link.PartyID)
	}
}

func TestResolvePendingPartyLink_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	_, err := svc.CreatePendingPartyLink(context.Background(), "inv-1", "party-1", "org-1")
	if err != nil {
		t.Fatalf("create pending link failed: %v", err)
	}

	err = svc.ResolvePendingPartyLink(context.Background(), "inv-1", account.ID)
	if err != nil {
		t.Fatalf("resolve pending link failed: %v", err)
	}

	// The party should now be linked to the account.
	memberships, err := svc.GetPartyMembershipsForAccount(context.Background(), account.ID, "org-1")
	if err != nil {
		t.Fatalf("get memberships failed: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership after resolve, got %d", len(memberships))
	}
	if memberships[0].PartyID != "party-1" {
		t.Errorf("expected party-1, got %q", memberships[0].PartyID)
	}
}

func TestResolvePendingPartyLink_NotFound(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.ResolvePendingPartyLink(context.Background(), "nonexistent-inv", account.ID)
	if !errors.Is(err, domain.ErrPendingLinkNotFound) {
		t.Errorf("expected ErrPendingLinkNotFound, got %v", err)
	}
}
