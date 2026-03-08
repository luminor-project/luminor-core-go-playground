package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

func TestShowDashboard_ReturnsInternalServerErrorWhenActiveOrgLookupFails(t *testing.T) {
	t.Parallel()

	acct := &fakeAccountFacade{
		getActiveOrgIDFunc: func(_ context.Context, accountID string) (string, error) {
			if accountID != "user-1" {
				t.Fatalf("unexpected account id: %s", accountID)
			}
			return "", errors.New("boom")
		},
	}

	h := NewHandler(
		&fakeOrgService{},
		&fakeOrgFacade{},
		acct,
		newTestSessionStore(),
	)

	req := httptest.NewRequest(http.MethodGet, "/organization", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	h.ShowDashboard(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	if acct.getActiveOrgIDCalls != 1 {
		t.Fatalf("expected GetActiveOrgID called once, got %d", acct.getActiveOrgIDCalls)
	}
	if !strings.Contains(rr.Body.String(), "error.internal") {
		t.Fatalf("expected translated fallback error key in body, got %q", rr.Body.String())
	}
}

func TestHandleInvite_RedirectsWhenNoActiveOrg(t *testing.T) {
	t.Parallel()

	orgService := &fakeOrgService{}
	acct := &fakeAccountFacade{
		getActiveOrgIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", nil
		},
	}
	h := NewHandler(
		orgService,
		&fakeOrgFacade{},
		acct,
		newTestSessionStore(),
	)

	form := url.Values{}
	form.Set("email", "teammate@example.com")
	req := httptest.NewRequest(http.MethodPost, "/organization/invite", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "user-1"}))
	rr := httptest.NewRecorder()

	h.HandleInvite(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/en/organization" {
		t.Fatalf("expected localized redirect to /en/organization, got %q", got)
	}
	if orgService.createInvitationAsActorCalls != 0 {
		t.Fatalf("expected no invitation creation, got %d calls", orgService.createInvitationAsActorCalls)
	}
}

func TestShowAcceptInvitation_ReturnsNotFoundWhenInvitationMissing(t *testing.T) {
	t.Parallel()

	orgService := &fakeOrgService{
		findInvitationByIDFunc: func(_ context.Context, _ string) (domain.Invitation, error) {
			return domain.Invitation{}, domain.ErrInvitationNotFound
		},
	}

	h := NewHandler(
		orgService,
		&fakeOrgFacade{},
		&fakeAccountFacade{},
		newTestSessionStore(),
	)

	req := httptest.NewRequest(http.MethodGet, "/organization/accept/inv-1", nil)
	req.SetPathValue("invitationId", "inv-1")
	rr := httptest.NewRecorder()

	h.ShowAcceptInvitation(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "organization.invitation.notFound") {
		t.Fatalf("expected not-found translation key in body, got %q", rr.Body.String())
	}
	if len(orgService.findInvitationByIDInputs) != 1 || orgService.findInvitationByIDInputs[0] != "inv-1" {
		t.Fatalf("expected FindInvitationByID called with inv-1, got %v", orgService.findInvitationByIDInputs)
	}
}

func newTestSessionStore() *sessions.CookieStore {
	return sessions.NewCookieStore([]byte("test-session-secret-12345678901234567890"))
}

type fakeAccountFacade struct {
	getActiveOrgIDFunc  func(ctx context.Context, accountID string) (string, error)
	getActiveOrgIDCalls int
}

func (f *fakeAccountFacade) GetActiveOrgID(ctx context.Context, accountID string) (string, error) {
	f.getActiveOrgIDCalls++
	if f.getActiveOrgIDFunc == nil {
		return "", nil
	}
	return f.getActiveOrgIDFunc(ctx, accountID)
}

func (f *fakeAccountFacade) GetAccountEmailByID(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (f *fakeAccountFacade) GetAccountInfoByIDs(_ context.Context, _ []string) ([]accountfacade.AccountInfoDTO, error) {
	return nil, nil
}

func (f *fakeAccountFacade) SetActiveOrganization(_ context.Context, _, _ string) error {
	return nil
}

type fakeOrgFacade struct {
	getOrganizationNameByIDFunc func(ctx context.Context, orgID string) (string, error)
}

func (f *fakeOrgFacade) GetOrganizationNameByID(ctx context.Context, orgID string) (string, error) {
	if f.getOrganizationNameByIDFunc == nil {
		return "", nil
	}
	return f.getOrganizationNameByIDFunc(ctx, orgID)
}

type fakeOrgService struct {
	findInvitationByIDFunc       func(ctx context.Context, id string) (domain.Invitation, error)
	findInvitationByIDInputs     []string
	createInvitationAsActorCalls int
}

func (f *fakeOrgService) GetAllOrganizationsForUser(_ context.Context, _ string) ([]domain.Organization, error) {
	return nil, nil
}

func (f *fakeOrgService) GetOrganizationByID(_ context.Context, _ string) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (f *fakeOrgService) IsOwner(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (f *fakeOrgService) UserHasAccessRight(_ context.Context, _, _ string, _ domain.AccessRight) (bool, error) {
	return false, nil
}

func (f *fakeOrgService) GetMemberIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (f *fakeOrgService) GetGroups(_ context.Context, _ string) ([]domain.Group, error) {
	return nil, nil
}

func (f *fakeOrgService) GetGroupMemberIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (f *fakeOrgService) GetPendingInvitations(_ context.Context, _ string) ([]domain.Invitation, error) {
	return nil, nil
}

func (f *fakeOrgService) CreateOrganization(_ context.Context, _, _ string) (domain.Organization, error) {
	return domain.Organization{}, nil
}

func (f *fakeOrgService) RenameOrganizationAsActor(_ context.Context, _, _, _ string) error {
	return nil
}

func (f *fakeOrgService) CanAccessOrganization(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (f *fakeOrgService) CreateInvitationAsActor(_ context.Context, _, _, _ string) (domain.Invitation, error) {
	f.createInvitationAsActorCalls++
	return domain.Invitation{}, nil
}

func (f *fakeOrgService) FindInvitationByID(ctx context.Context, id string) (domain.Invitation, error) {
	f.findInvitationByIDInputs = append(f.findInvitationByIDInputs, id)
	if f.findInvitationByIDFunc == nil {
		return domain.Invitation{}, nil
	}
	return f.findInvitationByIDFunc(ctx, id)
}

func (f *fakeOrgService) AcceptInvitation(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func (f *fakeOrgService) AddUserToGroupAsActor(_ context.Context, _, _, _ string) error {
	return nil
}

func (f *fakeOrgService) RemoveUserFromGroupAsActor(_ context.Context, _, _, _ string) error {
	return nil
}
