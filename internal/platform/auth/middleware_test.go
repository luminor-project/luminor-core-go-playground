package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"

	appSession "github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

func TestRequirePartyKind_Allowed(t *testing.T) {
	t.Parallel()

	called := false
	handler := RequirePartyKind("property_manager")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/property-management", nil)
	req = req.WithContext(WithUser(req.Context(), User{
		ID:              "user-1",
		Email:           "test@example.com",
		ActivePartyID:   "party-1",
		ActivePartyKind: "property_manager",
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected next handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequirePartyKind_Blocked(t *testing.T) {
	t.Parallel()

	called := false
	handler := RequirePartyKind("property_manager")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/property-management", nil)
	req = req.WithContext(WithUser(req.Context(), User{
		ID:              "user-1",
		Email:           "test@example.com",
		ActivePartyID:   "party-1",
		ActivePartyKind: "tenant",
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected next handler NOT to be called")
	}
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect, got %d", rr.Code)
	}
}

func TestRequirePartyKind_EmptyKindTreatedAsPM(t *testing.T) {
	t.Parallel()

	called := false
	handler := RequirePartyKind("property_manager")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/property-management", nil)
	req = req.WithContext(WithUser(req.Context(), User{
		ID:              "user-1",
		Email:           "test@example.com",
		ActivePartyKind: "", // empty → treated as property_manager
	}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected next handler to be called (empty kind treated as PM)")
	}
}

func TestRequirePartyKind_NoUser_Redirects(t *testing.T) {
	t.Parallel()

	called := false
	handler := RequirePartyKind("property_manager")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/property-management", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected next handler NOT to be called")
	}
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}
}

func TestLoadUser_ReadsPartyFromSession(t *testing.T) {
	t.Parallel()

	store := newTestSessionStore()

	var capturedUser User
	handler := LoadUser(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if ok {
			capturedUser = u
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with a session that has party info.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Set session values.
	sess, _ := store.Get(req, appSession.SessionName)
	sess.Values[appSession.KeyUserID] = "user-1"
	sess.Values[appSession.KeyEmail] = "test@example.com"
	sess.Values[appSession.KeyRoles] = []string{"user"}
	sess.Values[appSession.KeyActivePartyID] = "party-1"
	sess.Values[appSession.KeyActivePartyKind] = "tenant"
	_ = sess.Save(req, rr)

	// Re-create request with the cookie from the response.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rr.Result().Cookies() {
		req2.AddCookie(c)
	}
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)

	if capturedUser.ActivePartyID != "party-1" {
		t.Errorf("expected ActivePartyID 'party-1', got %q", capturedUser.ActivePartyID)
	}
	if capturedUser.ActivePartyKind != "tenant" {
		t.Errorf("expected ActivePartyKind 'tenant', got %q", capturedUser.ActivePartyKind)
	}
}

func newTestSessionStore() *sessions.CookieStore {
	return sessions.NewCookieStore([]byte("test-session-secret-12345678901234567890"))
}
