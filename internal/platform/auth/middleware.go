package auth

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	appSession "github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

// LoadUser is middleware that loads the authenticated user from the session
// into the request context, if present. It does NOT require authentication.
func LoadUser(store *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := store.Get(r, appSession.SessionName)
			if err != nil {
				slog.Warn("failed to get session", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			userID, ok := sess.Values[appSession.KeyUserID].(string)
			if !ok || userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			email, _ := sess.Values[appSession.KeyEmail].(string)
			roles, _ := sess.Values[appSession.KeyRoles].([]string)

			user := User{
				ID:    userID,
				Email: email,
				Roles: roles,
			}

			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is middleware that redirects unauthenticated users to the sign-in page.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r.Context()) {
			http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/sign-in"), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireGuest is middleware that redirects authenticated users to the dashboard.
func RequireGuest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsAuthenticated(r.Context()) {
			http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/dashboard"), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
