package web

import (
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

// RegisterRoutes registers all account-related routes on the mux.
func RegisterRoutes(mux *http.ServeMux, accounts accountUseCases, sessionStore *sessions.CookieStore) {
	h := NewHandler(accounts, sessionStore)

	// Guest-only routes (redirect if authenticated)
	mux.Handle("GET /sign-in", auth.RequireGuest(http.HandlerFunc(h.ShowSignIn)))
	mux.Handle("POST /sign-in", auth.RequireGuest(http.HandlerFunc(h.HandleSignIn)))
	mux.Handle("GET /sign-up", auth.RequireGuest(http.HandlerFunc(h.ShowSignUp)))
	mux.Handle("POST /sign-up", auth.RequireGuest(http.HandlerFunc(h.HandleSignUp)))

	// Auth-required routes
	mux.Handle("POST /sign-out", auth.RequireAuth(http.HandlerFunc(h.HandleSignOut)))
	mux.Handle("GET /dashboard", auth.RequireAuth(http.HandlerFunc(h.ShowDashboard)))
	mux.Handle("GET /set-password", auth.RequireAuth(http.HandlerFunc(h.ShowSetPassword)))
	mux.Handle("POST /set-password", auth.RequireAuth(http.HandlerFunc(h.HandleSetPassword)))
}
