package web

import (
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

// RegisterRoutes registers all organization-related routes on the mux.
func RegisterRoutes(
	mux *http.ServeMux,
	orgService orgUseCases,
	orgFacade orgNameProvider,
	acctFacade accountOrgUseCases,
	sessionStore *sessions.CookieStore,
) {
	h := NewHandler(orgService, orgFacade, acctFacade, sessionStore)

	// All routes require authentication
	mux.Handle("GET /organization", auth.RequireAuth(http.HandlerFunc(h.ShowDashboard)))
	mux.Handle("POST /organization/create", auth.RequireAuth(http.HandlerFunc(h.HandleCreate)))
	mux.Handle("POST /organization/rename", auth.RequireAuth(http.HandlerFunc(h.HandleRename)))
	mux.Handle("POST /organization/switch/{organizationId}", auth.RequireAuth(http.HandlerFunc(h.HandleSwitch)))
	mux.Handle("POST /organization/invite", auth.RequireAuth(http.HandlerFunc(h.HandleInvite)))
	mux.Handle("POST /organization/invitation/{invitationId}/resend", auth.RequireAuth(http.HandlerFunc(h.HandleResendInvitation)))
	mux.Handle("GET /organization/invitation/{invitationId}", http.HandlerFunc(h.ShowAcceptInvitation))
	mux.Handle("POST /organization/invitation/{invitationId}", auth.RequireAuth(http.HandlerFunc(h.HandleAcceptInvitation)))
	mux.Handle("POST /organization/group/{groupId}/add-member", auth.RequireAuth(http.HandlerFunc(h.HandleAddMemberToGroup)))
	mux.Handle("POST /organization/group/{groupId}/remove-member", auth.RequireAuth(http.HandlerFunc(h.HandleRemoveMemberFromGroup)))
}
