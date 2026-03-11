package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/organization/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	appCSRF "github.com/luminor-project/luminor-core-go-playground/internal/platform/csrf"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/flash"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
	appSession "github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

// Handler handles organization-related HTTP requests.
type Handler struct {
	orgService   orgUseCases
	orgFacade    orgNameProvider
	acctFacade   accountOrgUseCases
	sessionStore *sessions.CookieStore
	loader       *DashboardLoader
}

type orgUseCases interface {
	GetAllOrganizationsForUser(ctx context.Context, userID string) ([]domain.Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (domain.Organization, error)
	IsOwner(ctx context.Context, userID, orgID string) (bool, error)
	UserHasAccessRight(ctx context.Context, userID, orgID string, right domain.AccessRight) (bool, error)
	GetMemberIDs(ctx context.Context, orgID string) ([]string, error)
	GetGroups(ctx context.Context, orgID string) ([]domain.Group, error)
	GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error)
	GetPendingInvitations(ctx context.Context, orgID string) ([]domain.Invitation, error)
	CreateOrganization(ctx context.Context, ownerID, name string) (domain.Organization, error)
	RenameOrganizationAsActor(ctx context.Context, actorUserID, orgID, newName string) error
	CanAccessOrganization(ctx context.Context, userID, orgID string) (bool, error)
	CreateInvitationAsActor(ctx context.Context, actorUserID, orgID, email string) (domain.Invitation, error)
	FindInvitationByID(ctx context.Context, id string) (domain.Invitation, error)
	AcceptInvitation(ctx context.Context, invitationID, accountCoreID, accountEmail string) (string, error)
	AddUserToGroupAsActor(ctx context.Context, actorUserID, accountCoreID, groupID string) error
	RemoveUserFromGroupAsActor(ctx context.Context, actorUserID, accountCoreID, groupID string) error
}

type orgNameProvider interface {
	GetOrganizationNameByID(ctx context.Context, orgID string) (string, error)
}

type accountOrgUseCases interface {
	GetActiveOrgID(ctx context.Context, accountID string) (string, error)
	GetAccountInfoByIDs(ctx context.Context, ids []string) ([]accountfacade.AccountInfoDTO, error)
	GetAccountEmailByID(ctx context.Context, accountID string) (string, error)
	SetActiveOrganization(ctx context.Context, accountID, orgID string) error
}

// NewHandler creates a new organization handler.
func NewHandler(
	orgService orgUseCases,
	orgFacade orgNameProvider,
	acctFacade accountOrgUseCases,
	sessionStore *sessions.CookieStore,
) *Handler {
	return &Handler{
		orgService:   orgService,
		orgFacade:    orgFacade,
		acctFacade:   acctFacade,
		sessionStore: sessionStore,
		loader:       NewDashboardLoader(orgService, orgFacade, acctFacade),
	}
}

// ShowDashboard renders the organization dashboard.
func (h *Handler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	csrfToken := appCSRF.Token(r)

	data, err := h.loader.Load(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to load dashboard data", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	data.CSRFToken = csrfToken

	ctx := render.WithCSRFToken(r.Context(), csrfToken)
	render.Page(w, r.WithContext(ctx), templates.Dashboard(data))
}

// HandleCreate creates a new organization.
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")

	org, err := h.orgService.CreateOrganization(r.Context(), user.ID, name)
	if err != nil {
		slog.Error("create organization failed", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.createFailed")
		redirectWithLocale(w, r, "/organization")
		return
	}

	// Switch to the new org directly and fail fast on persistence errors.
	if err := h.acctFacade.SetActiveOrganization(r.Context(), user.ID, org.ID); err != nil {
		slog.Error("set active organization failed", "error", err, "user_id", user.ID, "org_id", org.ID)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.createFailed")
		redirectWithLocale(w, r, "/organization")
		return
	}

	h.updateSessionOrgName(w, r, org.ID)
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.created")
	redirectWithLocale(w, r, "/organization")
}

// HandleRename renames the active organization.
func (h *Handler) HandleRename(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")

	activeOrgID, err := h.acctFacade.GetActiveOrgID(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to get active organization", "error", err, "user_id", user.ID)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if activeOrgID == "" {
		redirectWithLocale(w, r, "/organization")
		return
	}

	if err := h.orgService.RenameOrganizationAsActor(r.Context(), user.ID, activeOrgID, name); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.renameFailed")
			redirectWithLocale(w, r, "/organization")
			return
		}
		slog.Error("rename organization failed", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.renameFailed")
	} else {
		h.updateSessionOrgName(w, r, activeOrgID)
		flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.renamed")
	}

	redirectWithLocale(w, r, "/organization")
}

// HandleSwitch switches the active organization.
func (h *Handler) HandleSwitch(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID := r.PathValue("organizationId")

	canAccess, err := h.orgService.CanAccessOrganization(r.Context(), user.ID, orgID)
	if err != nil {
		slog.Error("failed to check organization access", "error", err, "user_id", user.ID, "org_id", orgID)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if !canAccess {
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.switchFailed")
		redirectWithLocale(w, r, "/organization")
		return
	}
	if err := h.acctFacade.SetActiveOrganization(r.Context(), user.ID, orgID); err != nil {
		slog.Error("set active organization failed", "error", err, "user_id", user.ID, "org_id", orgID)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.switchFailed")
		redirectWithLocale(w, r, "/organization")
		return
	}

	h.updateSessionOrgName(w, r, orgID)
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.switched")
	redirectWithLocale(w, r, "/organization")
}

// HandleInvite sends an invitation.
func (h *Handler) HandleInvite(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")

	activeOrgID, err := h.acctFacade.GetActiveOrgID(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to get active organization", "error", err, "user_id", user.ID)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if activeOrgID == "" {
		redirectWithLocale(w, r, "/organization")
		return
	}

	_, err = h.orgService.CreateInvitationAsActor(r.Context(), user.ID, activeOrgID, email)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.inviteFailed")
			redirectWithLocale(w, r, "/organization")
			return
		}
		slog.Error("create invitation failed", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.inviteFailed")
	} else {
		flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.inviteSent", "email", email)
	}

	redirectWithLocale(w, r, "/organization")
}

// HandleResendInvitation resends an invitation.
func (h *Handler) HandleResendInvitation(w http.ResponseWriter, r *http.Request) {
	// In a full implementation, this would resend the invitation email
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.inviteResent")
	redirectWithLocale(w, r, "/organization")
}

// ShowAcceptInvitation renders the invitation acceptance page.
func (h *Handler) ShowAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID := r.PathValue("invitationId")
	csrfToken := appCSRF.Token(r)

	inv, err := h.orgService.FindInvitationByID(r.Context(), invitationID)
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "organization.invitation.notFound"), http.StatusNotFound)
		return
	}

	orgName, err := h.orgFacade.GetOrganizationNameByID(r.Context(), inv.OrganizationID)
	if err != nil {
		slog.Error("failed to get invitation organization name", "error", err, "org_id", inv.OrganizationID)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if orgName == "" {
		orgName = i18n.T(r.Context(), "organization.fallbackName")
	}
	org, err := h.orgService.GetOrganizationByID(r.Context(), inv.OrganizationID)
	if err != nil {
		slog.Error("failed to get invitation organization", "error", err, "org_id", inv.OrganizationID)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	ownerEmail, err := h.acctFacade.GetAccountEmailByID(r.Context(), org.OwningUsersID)
	if err != nil {
		slog.Error("failed to get owner email", "error", err, "owner_id", org.OwningUsersID)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	ctx := render.WithCSRFToken(r.Context(), csrfToken)
	render.Page(w, r.WithContext(ctx), templates.AcceptInvitation(invitationID, orgName, ownerEmail, csrfToken))
}

// HandleAcceptInvitation processes invitation acceptance.
func (h *Handler) HandleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID := r.PathValue("invitationId")
	user := auth.MustUserFromContext(r.Context())

	orgID, err := h.orgService.AcceptInvitation(r.Context(), invitationID, user.ID, user.Email)
	if err != nil {
		if errors.Is(err, domain.ErrInvitationEmailMismatch) || errors.Is(err, domain.ErrForbidden) {
			flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.acceptFailed")
			redirectWithLocale(w, r, "/")
			return
		}
		slog.Error("accept invitation failed", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.acceptFailed")
		redirectWithLocale(w, r, "/")
		return
	}

	// Switch to the joined org.
	if err := h.acctFacade.SetActiveOrganization(r.Context(), user.ID, orgID); err != nil {
		slog.Error("set active organization failed after invitation acceptance", "error", err, "user_id", user.ID, "org_id", orgID)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.acceptFailed")
		redirectWithLocale(w, r, "/")
		return
	}

	h.updateSessionOrgName(w, r, orgID)
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.joined")
	redirectWithLocale(w, r, "/organization")
}

// HandleAddMemberToGroup adds a member to a group.
func (h *Handler) HandleAddMemberToGroup(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}
	groupID := r.PathValue("groupId")
	memberID := r.FormValue("member_id")

	if err := h.orgService.AddUserToGroupAsActor(r.Context(), user.ID, memberID, groupID); err != nil {
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrCrossOrganizationGroupAssignment) {
			flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.addMemberFailed")
			redirectWithLocale(w, r, "/organization")
			return
		}
		slog.Error("add to group failed", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.addMemberFailed")
	} else {
		flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.memberAdded")
	}

	redirectWithLocale(w, r, "/organization")
}

// HandleRemoveMemberFromGroup removes a member from a group.
func (h *Handler) HandleRemoveMemberFromGroup(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}
	groupID := r.PathValue("groupId")
	memberID := r.FormValue("member_id")

	if err := h.orgService.RemoveUserFromGroupAsActor(r.Context(), user.ID, memberID, groupID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.removeMemberFailed")
			redirectWithLocale(w, r, "/organization")
			return
		}
		slog.Error("remove from group failed", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeError, "flash.organization.removeMemberFailed")
	} else {
		flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.organization.memberRemoved")
	}

	redirectWithLocale(w, r, "/organization")
}

func (h *Handler) updateSessionOrgName(w http.ResponseWriter, r *http.Request, orgID string) {
	name, err := h.orgFacade.GetOrganizationNameByID(r.Context(), orgID)
	if err != nil {
		slog.Warn("failed to load org name for session", "error", err, "org_id", orgID)
		return
	}
	sess, err := h.sessionStore.Get(r, appSession.SessionName)
	if err != nil {
		slog.Warn("failed to get session for org name update", "error", err)
		return
	}
	sess.Values[appSession.KeyOrgName] = name
	if err := sess.Save(r, w); err != nil {
		slog.Warn("failed to save session with org name", "error", err)
	}
}

func redirectWithLocale(w http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), path), http.StatusSeeOther)
}
