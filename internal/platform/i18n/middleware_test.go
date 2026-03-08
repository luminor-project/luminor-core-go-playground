package i18n_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

func TestMiddleware_RedirectsUnprefixedRoutes(t *testing.T) {
	translator, err := i18n.LoadEmbeddedTranslator()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	handler := i18n.Middleware(translator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected %d got %d", http.StatusPermanentRedirect, rr.Code)
	}
	location := rr.Header().Get("Location")
	if location != "/de/about" {
		t.Fatalf("unexpected location %q", location)
	}
}

func TestMiddleware_StripsLocalePrefixForRouter(t *testing.T) {
	translator, err := i18n.LoadEmbeddedTranslator()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	handler := i18n.Middleware(translator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dashboard" {
			t.Fatalf("expected stripped path '/dashboard', got %q", r.URL.Path)
		}
		if got := i18n.LocaleFromContext(r.Context()); got != i18n.LocaleFR {
			t.Fatalf("expected locale fr got %s", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/fr/dashboard", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Language"); got != "fr" {
		t.Fatalf("expected Content-Language=fr got %q", got)
	}
}
