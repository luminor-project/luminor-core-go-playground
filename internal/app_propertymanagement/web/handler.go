package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_propertymanagement/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/app_propertymanagement/web/templates"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
)

type pmUseCases interface {
	CreateProperty(ctx context.Context, dto facade.CreatePropertyDTO) (string, error)
	CreateTenant(ctx context.Context, dto facade.CreateTenantDTO) (string, error)
	AssignTenantToProperty(ctx context.Context, dto facade.AssignTenantDTO) (string, error)
	InviteTenant(ctx context.Context, dto facade.InviteTenantDTO) error
	CreatePropertyOwner(ctx context.Context, dto facade.CreatePropertyOwnerDTO) (string, error)
	InvitePropertyOwner(ctx context.Context, dto facade.InvitePropertyOwnerDTO) error
}

type partyLister interface {
	ListPartiesByOrgAndKind(ctx context.Context, orgID string, kind partyfacade.PartyKind) ([]partyfacade.PartyInfoDTO, error)
}

type subjectLister interface {
	ListSubjectsByOrgAndKind(ctx context.Context, orgID string, kind subjectfacade.SubjectKind) ([]subjectfacade.SubjectInfoDTO, error)
}

type rentalLister interface {
	ListRentalsByOrg(ctx context.Context, orgID string) ([]rentalfacade.RentalInfoDTO, error)
}

type activeOrgProvider interface {
	GetActiveOrgID(ctx context.Context, accountID string) (string, error)
}

// Handler handles property management HTTP requests.
type Handler struct {
	pm       pmUseCases
	parties  partyLister
	subjects subjectLister
	rentals  rentalLister
	accounts activeOrgProvider
}

// NewHandler creates a new property management handler.
func NewHandler(pm pmUseCases, parties partyLister, subjects subjectLister, rentals rentalLister, accounts activeOrgProvider) *Handler {
	return &Handler{
		pm:       pm,
		parties:  parties,
		subjects: subjects,
		rentals:  rentals,
		accounts: accounts,
	}
}

func (h *Handler) getActiveOrgID(r *http.Request) (string, error) {
	user := auth.MustUserFromContext(r.Context())
	return h.accounts.GetActiveOrgID(r.Context(), user.ID)
}

// ShowDashboard renders the property management dashboard.
func (h *Handler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID, err := h.accounts.GetActiveOrgID(r.Context(), user.ID)
	if err != nil || orgID == "" {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	properties, _ := h.subjects.ListSubjectsByOrgAndKind(r.Context(), orgID, subjectfacade.SubjectKindDwelling)
	tenants, _ := h.parties.ListPartiesByOrgAndKind(r.Context(), orgID, partyfacade.PartyKindTenant)
	propertyOwners, _ := h.parties.ListPartiesByOrgAndKind(r.Context(), orgID, partyfacade.PartyKindPropertyOwner)
	rentals, _ := h.rentals.ListRentalsByOrg(r.Context(), orgID)

	render.Page(w, r, templates.Dashboard(templates.DashboardData{
		Properties:     properties,
		Tenants:        tenants,
		PropertyOwners: propertyOwners,
		Rentals:        rentals,
	}))
}

// HandleCreateProperty handles POST /property-management/properties.
func (h *Handler) HandleCreateProperty(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID, err := h.getActiveOrgID(r)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	_, err = h.pm.CreateProperty(r.Context(), facade.CreatePropertyDTO{
		Name:               r.FormValue("name"),
		Detail:             r.FormValue("detail"),
		OrgID:              orgID,
		CreatedByAccountID: user.ID,
	})
	if err != nil {
		slog.Error("create property failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/property-management"), http.StatusSeeOther)
}

// HandleCreateTenant handles POST /property-management/tenants.
func (h *Handler) HandleCreateTenant(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID, err := h.getActiveOrgID(r)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	_, err = h.pm.CreateTenant(r.Context(), facade.CreateTenantDTO{
		Name:               r.FormValue("name"),
		OrgID:              orgID,
		CreatedByAccountID: user.ID,
	})
	if err != nil {
		slog.Error("create tenant failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/property-management"), http.StatusSeeOther)
}

// HandleAssignTenant handles POST /property-management/rentals.
func (h *Handler) HandleAssignTenant(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID, err := h.getActiveOrgID(r)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	_, err = h.pm.AssignTenantToProperty(r.Context(), facade.AssignTenantDTO{
		SubjectID:          r.FormValue("subject_id"),
		TenantPartyID:      r.FormValue("tenant_party_id"),
		OrgID:              orgID,
		CreatedByAccountID: user.ID,
	})
	if err != nil {
		slog.Error("assign tenant failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/property-management"), http.StatusSeeOther)
}

// HandleInviteTenant handles POST /property-management/tenants/{tenantId}/invite.
func (h *Handler) HandleInviteTenant(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	tenantPartyID := r.PathValue("tenantId")
	orgID, err := h.getActiveOrgID(r)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	err = h.pm.InviteTenant(r.Context(), facade.InviteTenantDTO{
		TenantPartyID:  tenantPartyID,
		Email:          r.FormValue("email"),
		OrgID:          orgID,
		ActorAccountID: user.ID,
	})
	if err != nil {
		slog.Error("invite tenant failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/property-management"), http.StatusSeeOther)
}

// HandleCreatePropertyOwner handles POST /property-management/property-owners.
func (h *Handler) HandleCreatePropertyOwner(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID, err := h.getActiveOrgID(r)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	_, err = h.pm.CreatePropertyOwner(r.Context(), facade.CreatePropertyOwnerDTO{
		Name:               r.FormValue("name"),
		OrgID:              orgID,
		CreatedByAccountID: user.ID,
	})
	if err != nil {
		slog.Error("create property owner failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/property-management"), http.StatusSeeOther)
}

// HandleInvitePropertyOwner handles POST /property-management/property-owners/{ownerId}/invite.
func (h *Handler) HandleInvitePropertyOwner(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	propertyOwnerPartyID := r.PathValue("ownerId")
	orgID, err := h.getActiveOrgID(r)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	err = h.pm.InvitePropertyOwner(r.Context(), facade.InvitePropertyOwnerDTO{
		PropertyOwnerPartyID: propertyOwnerPartyID,
		Email:                r.FormValue("email"),
		OrgID:                orgID,
		ActorAccountID:       user.ID,
	})
	if err != nil {
		slog.Error("invite property owner failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/property-management"), http.StatusSeeOther)
}
