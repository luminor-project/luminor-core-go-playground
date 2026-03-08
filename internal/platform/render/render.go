package render

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/flash"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

type contextKey string

const (
	csrfTokenKey contextKey = "csrf_token"
	requestKey   contextKey = "http_request"
)

// PageData holds common data available to all templates.
type PageData struct {
	User            *auth.User
	IsAuthenticated bool
	CSRFToken       string
	FlashMessages   []flash.Message
	Locale          i18n.Locale
	Translator      *i18n.Translator
}

// PageDataFromContext builds PageData from a request context.
func PageDataFromContext(ctx context.Context) PageData {
	pd := PageData{}

	if user, ok := auth.UserFromContext(ctx); ok {
		pd.User = &user
		pd.IsAuthenticated = true
	}

	pd.FlashMessages = flash.FromContext(ctx)

	if token, ok := ctx.Value(csrfTokenKey).(string); ok {
		pd.CSRFToken = token
	}

	pd.Locale = i18n.LocaleFromContext(ctx)
	pd.Translator = i18n.TranslatorFromContext(ctx)

	return pd
}

// WithCSRFToken adds a CSRF token to the context.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenKey, token)
}

// CSRFTokenFromContext returns the CSRF token from the context.
func CSRFTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(csrfTokenKey).(string)
	return token
}

// Component renders a templ component with the given status code.
func Component(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("component render failed", "error", err)
	}
}

// Page renders a full-page templ component with 200 status.
func Page(w http.ResponseWriter, r *http.Request, component templ.Component) {
	Component(w, r, http.StatusOK, component)
}
