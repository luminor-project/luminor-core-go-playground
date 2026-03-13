package facade

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
)

type fakeService struct {
	setActivePartyFunc          func(ctx context.Context, accountID, partyID string) error
	linkPartyToAccountFunc      func(ctx context.Context, accountID, partyID, orgID string) error
	getPartyMembershipsFunc     func(ctx context.Context, accountID, orgID string) ([]domain.PartyMembership, error)
	getAccountIDsForPartyFunc   func(ctx context.Context, partyID string) ([]string, error)
	createPendingPartyLinkFunc  func(ctx context.Context, invitationID, partyID, orgID string) (domain.PendingPartyLink, error)
	resolvePendingPartyLinkFunc func(ctx context.Context, invitationID, accountID string) error

	// Unused stubs required by accountService interface.
	registerFunc     func(ctx context.Context, email, pw string) (domain.AccountCore, error)
	authenticateFunc func(ctx context.Context, email, pw string) (domain.AccountCore, error)
	findByEmailFunc  func(ctx context.Context, email string) (domain.AccountCore, error)
	findByIDFunc     func(ctx context.Context, id string) (domain.AccountCore, error)
	findByIDsFunc    func(ctx context.Context, ids []string) ([]domain.AccountCore, error)
	setActiveOrgFunc func(ctx context.Context, accountID, orgID string) error
	setPasswordFunc  func(ctx context.Context, accountID, pw string) error
}

func (f *fakeService) Register(ctx context.Context, email, pw string) (domain.AccountCore, error) {
	if f.registerFunc != nil {
		return f.registerFunc(ctx, email, pw)
	}
	return domain.AccountCore{}, nil
}

func (f *fakeService) Authenticate(ctx context.Context, email, pw string) (domain.AccountCore, error) {
	if f.authenticateFunc != nil {
		return f.authenticateFunc(ctx, email, pw)
	}
	return domain.AccountCore{}, nil
}

func (f *fakeService) FindByEmail(ctx context.Context, email string) (domain.AccountCore, error) {
	if f.findByEmailFunc != nil {
		return f.findByEmailFunc(ctx, email)
	}
	return domain.AccountCore{}, nil
}

func (f *fakeService) FindByID(ctx context.Context, id string) (domain.AccountCore, error) {
	if f.findByIDFunc != nil {
		return f.findByIDFunc(ctx, id)
	}
	return domain.AccountCore{}, nil
}

func (f *fakeService) FindByIDs(ctx context.Context, ids []string) ([]domain.AccountCore, error) {
	if f.findByIDsFunc != nil {
		return f.findByIDsFunc(ctx, ids)
	}
	return nil, nil
}

func (f *fakeService) SetActiveOrganization(ctx context.Context, accountID, orgID string) error {
	if f.setActiveOrgFunc != nil {
		return f.setActiveOrgFunc(ctx, accountID, orgID)
	}
	return nil
}

func (f *fakeService) SetPassword(ctx context.Context, accountID, pw string) error {
	if f.setPasswordFunc != nil {
		return f.setPasswordFunc(ctx, accountID, pw)
	}
	return nil
}

func (f *fakeService) SetActiveParty(ctx context.Context, accountID, partyID string) error {
	if f.setActivePartyFunc != nil {
		return f.setActivePartyFunc(ctx, accountID, partyID)
	}
	return nil
}

func (f *fakeService) LinkPartyToAccount(ctx context.Context, accountID, partyID, orgID string) error {
	if f.linkPartyToAccountFunc != nil {
		return f.linkPartyToAccountFunc(ctx, accountID, partyID, orgID)
	}
	return nil
}

func (f *fakeService) GetPartyMembershipsForAccount(ctx context.Context, accountID, orgID string) ([]domain.PartyMembership, error) {
	if f.getPartyMembershipsFunc != nil {
		return f.getPartyMembershipsFunc(ctx, accountID, orgID)
	}
	return nil, nil
}

func (f *fakeService) GetAccountIDsForParty(ctx context.Context, partyID string) ([]string, error) {
	if f.getAccountIDsForPartyFunc != nil {
		return f.getAccountIDsForPartyFunc(ctx, partyID)
	}
	return nil, nil
}

func (f *fakeService) CreatePendingPartyLink(ctx context.Context, invitationID, partyID, orgID string) (domain.PendingPartyLink, error) {
	if f.createPendingPartyLinkFunc != nil {
		return f.createPendingPartyLinkFunc(ctx, invitationID, partyID, orgID)
	}
	return domain.PendingPartyLink{}, nil
}

func (f *fakeService) ResolvePendingPartyLink(ctx context.Context, invitationID, accountID string) error {
	if f.resolvePendingPartyLinkFunc != nil {
		return f.resolvePendingPartyLinkFunc(ctx, invitationID, accountID)
	}
	return nil
}

func (f *fakeService) CreatePasswordResetToken(ctx context.Context, accountID string) (domain.PasswordResetToken, string, error) {
	return domain.PasswordResetToken{}, "", nil
}

func (f *fakeService) FindValidPasswordResetToken(ctx context.Context, accountID, rawToken string) (domain.PasswordResetToken, error) {
	return domain.PasswordResetToken{}, nil
}

func (f *fakeService) MarkPasswordResetTokenUsed(ctx context.Context, tokenID string) error {
	return nil
}

func TestSetActiveParty_DelegatesToService(t *testing.T) {
	t.Parallel()

	var calledAccountID, calledPartyID string
	svc := &fakeService{
		setActivePartyFunc: func(_ context.Context, accountID, partyID string) error {
			calledAccountID = accountID
			calledPartyID = partyID
			return nil
		},
	}

	fac := New(svc, nil, nil)
	err := fac.SetActiveParty(context.Background(), "acct-1", "party-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledAccountID != "acct-1" {
		t.Errorf("expected accountID 'acct-1', got %q", calledAccountID)
	}
	if calledPartyID != "party-1" {
		t.Errorf("expected partyID 'party-1', got %q", calledPartyID)
	}
}

func TestLinkPartyToAccount_DelegatesToService(t *testing.T) {
	t.Parallel()

	var calledAccountID, calledPartyID, calledOrgID string
	svc := &fakeService{
		linkPartyToAccountFunc: func(_ context.Context, accountID, partyID, orgID string) error {
			calledAccountID = accountID
			calledPartyID = partyID
			calledOrgID = orgID
			return nil
		},
	}

	fac := New(svc, nil, nil)
	err := fac.LinkPartyToAccount(context.Background(), "acct-1", "party-1", "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledAccountID != "acct-1" || calledPartyID != "party-1" || calledOrgID != "org-1" {
		t.Errorf("wrong args: %q %q %q", calledAccountID, calledPartyID, calledOrgID)
	}
}

func TestLinkPartyToAccount_AlreadyLinked(t *testing.T) {
	t.Parallel()

	svc := &fakeService{
		linkPartyToAccountFunc: func(_ context.Context, _, _, _ string) error {
			return domain.ErrAlreadyLinked
		},
	}

	fac := New(svc, nil, nil)
	err := fac.LinkPartyToAccount(context.Background(), "acct-1", "party-1", "org-1")
	if !errors.Is(err, ErrAlreadyLinked) {
		t.Errorf("expected ErrAlreadyLinked, got %v", err)
	}
}

func TestGetPartyMemberships_MapsCorrectly(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	svc := &fakeService{
		getPartyMembershipsFunc: func(_ context.Context, _, _ string) ([]domain.PartyMembership, error) {
			return []domain.PartyMembership{
				{AccountID: "acct-1", PartyID: "party-1", OrgID: "org-1", CreatedAt: now},
			}, nil
		},
	}

	fac := New(svc, nil, nil)
	memberships, err := fac.GetPartyMembershipsForAccount(context.Background(), "acct-1", "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if memberships[0].PartyID != "party-1" {
		t.Errorf("expected party-1, got %q", memberships[0].PartyID)
	}
	if memberships[0].AccountID != "acct-1" {
		t.Errorf("expected acct-1, got %q", memberships[0].AccountID)
	}
}

func TestCreatePendingPartyLink_DelegatesToService(t *testing.T) {
	t.Parallel()

	svc := &fakeService{
		createPendingPartyLinkFunc: func(_ context.Context, invID, partyID, orgID string) (domain.PendingPartyLink, error) {
			return domain.PendingPartyLink{
				ID:           "link-1",
				InvitationID: invID,
				PartyID:      partyID,
				OrgID:        orgID,
			}, nil
		},
	}

	fac := New(svc, nil, nil)
	linkID, err := fac.CreatePendingPartyLink(context.Background(), "inv-1", "party-1", "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if linkID != "link-1" {
		t.Errorf("expected link ID 'link-1', got %q", linkID)
	}
}

func TestResolvePendingPartyLink_DelegatesToService(t *testing.T) {
	t.Parallel()

	var calledInvID, calledAcctID string
	svc := &fakeService{
		resolvePendingPartyLinkFunc: func(_ context.Context, invID, acctID string) error {
			calledInvID = invID
			calledAcctID = acctID
			return nil
		},
	}

	fac := New(svc, nil, nil)
	err := fac.ResolvePendingPartyLink(context.Background(), "inv-1", "acct-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledInvID != "inv-1" {
		t.Errorf("expected inv-1, got %q", calledInvID)
	}
	if calledAcctID != "acct-1" {
		t.Errorf("expected acct-1, got %q", calledAcctID)
	}
}
