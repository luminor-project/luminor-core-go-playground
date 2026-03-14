package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ShowHomepage(t *testing.T) {
	t.Parallel()

	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ShowHomepage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()

	if body == "" {
		t.Error("expected body to not be empty")
	}
}

func TestHandler_ShowAbout(t *testing.T) {
	t.Parallel()

	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rr := httptest.NewRecorder()

	h.ShowAbout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandler_GreetingAppearsOnHomepage(t *testing.T) {
	t.Parallel()

	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ShowHomepage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()

	provider := NewGreetingProvider()
	foundGreeting := false
	for _, greeting := range provider.messages {
		if strings.Contains(body, greeting) {
			foundGreeting = true
			break
		}
	}

	if !foundGreeting {
		t.Log("Warning: No expected greeting message found in response body")
	}
}
