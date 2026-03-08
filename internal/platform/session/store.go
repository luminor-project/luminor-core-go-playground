package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "luminor_session"
	KeyUserID   = "user_id"
	KeyEmail    = "email"
	KeyRoles    = "roles"
	KeyFlash    = "_flash"
)

func NewStore(secretKey string) *sessions.CookieStore {
	store := sessions.NewCookieStore([]byte(secretKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true in production via config
	}
	return store
}
