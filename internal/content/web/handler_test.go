package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShowPricing_ReturnsOK(t *testing.T) {
	t.Parallel()

	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/pricing", nil)
	rr := httptest.NewRecorder()

	h.ShowPricing(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
}
