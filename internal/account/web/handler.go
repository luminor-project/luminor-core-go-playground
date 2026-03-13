package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/account/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	appCSRF "github.com/luminor-project/luminor-core-go-playground/internal/platform/csrf"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/flash"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
	appSession "github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

// Handler handles account-related HTTP requests.
type Handler struct {
	accounts     accountUseCases
	sessionStore *sessions.CookieStore
	enricher     sessionEnricher
}

type accountUseCases interface {
	Authenticate(ctx context.Context, email, password string) (facade.AccountInfoDTO, error)
	MustSetPassword(ctx context.Context, email string) (bool, error)
	Register(ctx context.Context, dto facade.RegistrationDTO) (string, error)
	GetAccountInfoByID(ctx context.Context, id string) (facade.AccountInfoDTO, error)
	SetPassword(ctx context.Context, accountID, newPassword string) error
	RequestMagicLink(ctx context.Context, dto facade.MagicLinkRequestDTO, baseURL string) error
	ValidateMagicLink(ctx context.Context, plaintextToken string) (facade.MagicLinkResultDTO, error)
}

type sessionEnricher interface {
	GetPartyNameAndKind(ctx context.Context, partyID string) (name string, kind string, err error)
	GetOrgName(ctx context.Context, orgID string) (string, error)
}

// NewHandler creates a new account handler.
func NewHandler(accounts accountUseCases, sessionStore *sessions.CookieStore, enricher sessionEnricher) *Handler {
	return &Handler{
		accounts:     accounts,
		sessionStore: sessionStore,
		enricher:     enricher,
	}
}

// ShowSignIn renders the sign-in page.
func (h *Handler) ShowSignIn(w http.ResponseWriter, r *http.Request) {
	ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
	render.Page(w, r.WithContext(ctx), templates.SignIn(render.CSRFTokenFromContext(ctx), "", ""))
}

// HandleSignIn processes the sign-in form submission.
func (h *Handler) HandleSignIn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	info, err := h.accounts.Authenticate(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
			render.Component(w, r.WithContext(ctx), http.StatusOK,
				templates.SignIn(render.CSRFTokenFromContext(ctx), email, i18n.T(ctx, "auth.validation.invalidCredentials")))
			return
		}
		slog.Error("sign in failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	// Check if user must set password
	mustSet, err := h.accounts.MustSetPassword(r.Context(), email)
	if err != nil {
		slog.Error("failed to check must-set-password", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if mustSet {
		h.setSession(w, r, info)
		redirectWithLocale(w, r, "/set-password")
		return
	}

	h.setSession(w, r, info)
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.auth.welcomeBack")
	redirectWithLocale(w, r, "/dashboard")
}

// ShowSignUp renders the sign-up page.
func (h *Handler) ShowSignUp(w http.ResponseWriter, r *http.Request) {
	ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
	render.Page(w, r.WithContext(ctx), templates.SignUp(render.CSRFTokenFromContext(ctx), "", ""))
}

// HandleSignUp processes the sign-up form submission.
func (h *Handler) HandleSignUp(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if password != passwordConfirm {
		ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
		render.Component(w, r.WithContext(ctx), http.StatusOK,
			templates.SignUp(render.CSRFTokenFromContext(ctx), email, i18n.T(ctx, "auth.validation.passwordsDoNotMatch")))
		return
	}

	userID, err := h.accounts.Register(r.Context(), facade.RegistrationDTO{
		Email:         email,
		PlainPassword: password,
	})
	if err != nil {
		var validationErr *domain.ValidationError
		if errors.As(err, &validationErr) {
			ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
			render.Component(w, r.WithContext(ctx), http.StatusOK,
				templates.SignUp(render.CSRFTokenFromContext(ctx), email, i18n.T(ctx, validationErr.Key)))
			return
		}
		slog.Error("registration failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	// Auto sign-in after registration
	info, err := h.accounts.GetAccountInfoByID(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get account after registration", "error", err)
		flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.auth.accountCreatedSignIn")
		redirectWithLocale(w, r, "/sign-in")
		return
	}

	h.setSession(w, r, info)
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.auth.accountCreatedWelcome")
	redirectWithLocale(w, r, "/dashboard")
}

// HandleSignOut processes sign-out.
func (h *Handler) HandleSignOut(w http.ResponseWriter, r *http.Request) {
	sess, err := h.sessionStore.Get(r, appSession.SessionName)
	if err == nil {
		sess.Options.MaxAge = -1
		if saveErr := sess.Save(r, w); saveErr != nil {
			slog.Error("failed to clear session during sign out", "error", saveErr)
		}
	}

	redirectWithLocale(w, r, "/")
}

// ShowDashboard renders the account dashboard.
func (h *Handler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())

	info, err := h.accounts.GetAccountInfoByID(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to get account info", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	var orgName string
	if info.CurrentlyActiveOrganizationID != "" && h.enricher != nil {
		name, err := h.enricher.GetOrgName(r.Context(), info.CurrentlyActiveOrganizationID)
		if err != nil {
			slog.Warn("failed to load org name for dashboard", "error", err)
		} else {
			orgName = name
		}
	}

	ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
	render.Page(w, r.WithContext(ctx), templates.Dashboard(info, orgName, render.CSRFTokenFromContext(ctx)))
}

// ShowSetPassword renders the set-password page.
func (h *Handler) ShowSetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
	render.Page(w, r.WithContext(ctx), templates.SetPassword(render.CSRFTokenFromContext(ctx), ""))
}

// HandleSetPassword processes the set-password form.
func (h *Handler) HandleSetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	user := auth.MustUserFromContext(r.Context())
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if password != passwordConfirm {
		ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
		render.Component(w, r.WithContext(ctx), http.StatusOK,
			templates.SetPassword(render.CSRFTokenFromContext(ctx), i18n.T(ctx, "auth.validation.passwordsDoNotMatch")))
		return
	}

	if err := h.accounts.SetPassword(r.Context(), user.ID, password); err != nil {
		if errors.Is(err, domain.ErrPasswordTooShort) {
			ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
			render.Component(w, r.WithContext(ctx), http.StatusOK,
				templates.SetPassword(render.CSRFTokenFromContext(ctx), i18n.T(ctx, "auth.validation.passwordTooShort")))
			return
		}
		slog.Error("set password failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.auth.passwordUpdated")
	redirectWithLocale(w, r, "/dashboard")
}

func (h *Handler) setSession(w http.ResponseWriter, r *http.Request, info facade.AccountInfoDTO) {
	sess, err := h.sessionStore.Get(r, appSession.SessionName)
	if err != nil {
		slog.Error("failed to get session", "error", err)
		return
	}

	sess.Values[appSession.KeyUserID] = info.ID
	sess.Values[appSession.KeyEmail] = info.Email
	sess.Values[appSession.KeyRoles] = info.Roles

	h.enrichSessionWithPartyAndOrg(r.Context(), sess, info)

	if err := sess.Save(r, w); err != nil {
		slog.Error("failed to save session", "error", err)
	}
}

func (h *Handler) enrichSessionWithPartyAndOrg(ctx context.Context, sess *sessions.Session, info facade.AccountInfoDTO) {
	if h.enricher == nil {
		return
	}

	if info.CurrentlyActivePartyID != "" {
		name, kind, err := h.enricher.GetPartyNameAndKind(ctx, info.CurrentlyActivePartyID)
		if err != nil {
			slog.Warn("failed to load party info for session", "error", err, "party_id", info.CurrentlyActivePartyID)
		} else {
			sess.Values[appSession.KeyActivePartyID] = info.CurrentlyActivePartyID
			sess.Values[appSession.KeyActivePartyKind] = kind
			sess.Values[appSession.KeyActivePartyName] = name
		}
	}

	if info.CurrentlyActiveOrganizationID != "" {
		name, err := h.enricher.GetOrgName(ctx, info.CurrentlyActiveOrganizationID)
		if err != nil {
			slog.Warn("failed to load org name for session", "error", err, "org_id", info.CurrentlyActiveOrganizationID)
		} else {
			sess.Values[appSession.KeyOrgName] = name
		}
	}
}

func redirectWithLocale(w http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), path), http.StatusSeeOther)
}

// ShowRequestMagicLink renders the magic link request page.
func (h *Handler) ShowRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
	render.Page(w, r.WithContext(ctx), templates.MagicLinkRequest(render.CSRFTokenFromContext(ctx), ""))
}

// HandleRequestMagicLink processes the magic link request form submission.
func (h *Handler) HandleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	// Build base URL for the magic link
	baseURL := getBaseURL(r)

	// Request the magic link (always succeeds from user perspective to prevent enumeration)
	err := h.accounts.RequestMagicLink(r.Context(), facade.MagicLinkRequestDTO{Email: email}, baseURL)
	if err != nil {
		// Log the error but don't show it to the user
		slog.Error("magic link request failed", "error", err, "email", email)
	}

	// Always show success message to prevent email enumeration
	render.Page(w, r, templates.MagicLinkSent())
}

// HandleMagicLinkLogin validates a magic link token and logs the user in.
func (h *Handler) HandleMagicLinkLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
		render.Page(w, r.WithContext(ctx), templates.MagicLinkInvalid(render.CSRFTokenFromContext(ctx), i18n.T(r.Context(), "auth.magicLink.invalid")))
		return
	}

	result, err := h.accounts.ValidateMagicLink(r.Context(), token)
	if err != nil {
		var validationErr *domain.MagicLinkValidationError
		if errors.As(err, &validationErr) {
			ctx := render.WithCSRFToken(r.Context(), appCSRF.Token(r))
			render.Page(w, r.WithContext(ctx), templates.MagicLinkInvalid(render.CSRFTokenFromContext(ctx), i18n.T(r.Context(), validationErr.Key)))
			return
		}
		slog.Error("magic link validation failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	// Create account info DTO from magic link result
	info := facade.AccountInfoDTO{
		ID:    result.AccountID,
		Email: result.Email,
		Roles: result.Roles,
	}

	// Set session
	h.setSession(w, r, info)
	flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.auth.welcomeBack")
	redirectWithLocale(w, r, "/dashboard")
}

// getBaseURL extracts the base URL from the request.
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
