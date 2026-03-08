package csrf

import (
	"log/slog"
	"net/http"
)

// Middleware returns a CSRF protection middleware.
func Middleware(_ string, _ bool, _ string) func(http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Warn("cross-origin protection blocked request")
		http.Error(w, "CSRF token invalid.", http.StatusForbidden)
	}))
	return cop.Handler
}

// Token returns the CSRF token for the current request.
func Token(r *http.Request) string {
	_ = r
	return ""
}

// TemplateField returns a hidden input field with the CSRF token.
func TemplateField(r *http.Request) string {
	_ = r
	return ""
}
