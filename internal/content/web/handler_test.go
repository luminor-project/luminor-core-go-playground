package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/content/config"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

type fakeGreetingsProvider struct {
	greeting    config.Greeting
	callCount   int
	returnEmpty bool
}

func (f *fakeGreetingsProvider) GetGreeting() config.Greeting {
	f.callCount++
	if f.returnEmpty {
		return config.Greeting{}
	}
	return f.greeting
}

func TestShowHomepage_WithGreeting(t *testing.T) {
	t.Parallel()

	fakeProvider := &fakeGreetingsProvider{
		greeting: config.Greeting{Text: "Hello from test!"},
	}

	h := NewHandler(fakeProvider)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"homepage.hero.title": "Test Title"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowHomepage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if fakeProvider.callCount != 1 {
		t.Fatalf("expected GetGreeting called once, got %d", fakeProvider.callCount)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Hello from test!") {
		t.Errorf("expected body to contain greeting 'Hello from test!', got %q", body)
	}
}

func TestShowHomepage_WithEmptyGreeting(t *testing.T) {
	t.Parallel()

	fakeProvider := &fakeGreetingsProvider{returnEmpty: true}

	h := NewHandler(fakeProvider)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"homepage.hero.title": "Test Title"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowHomepage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if fakeProvider.callCount != 1 {
		t.Fatalf("expected GetGreeting called once, got %d", fakeProvider.callCount)
	}

	body := rr.Body.String()
	// When greeting is empty, template should not render the greeting paragraph
	if strings.Contains(body, "Hello from test!") {
		t.Errorf("expected no greeting when empty, but found greeting in body")
	}
}

func TestShowHomepage_WithAuthenticatedUser(t *testing.T) {
	t.Parallel()

	fakeProvider := &fakeGreetingsProvider{
		greeting: config.Greeting{Text: "Welcome back!"},
	}

	h := NewHandler(fakeProvider)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"homepage.hero.title": "Test Title"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "user-1", Email: "test@example.com"}))
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowHomepage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if fakeProvider.callCount != 1 {
		t.Fatalf("expected GetGreeting called once, got %d", fakeProvider.callCount)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Welcome back!") {
		t.Errorf("expected body to contain greeting 'Welcome back!', got %q", body)
	}
}

func TestShowAbout(t *testing.T) {
	t.Parallel()

	fakeProvider := &fakeGreetingsProvider{}

	h := NewHandler(fakeProvider)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"about.title": "About"},
	})

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowAbout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// About page should not call greetings provider
	if fakeProvider.callCount != 0 {
		t.Errorf("expected GetGreeting not called for about page, got %d calls", fakeProvider.callCount)
	}
}

func TestShowLivingStyleguide(t *testing.T) {
	t.Parallel()

	fakeProvider := &fakeGreetingsProvider{}

	h := NewHandler(fakeProvider)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"styleguide.title": "Styleguide"},
	})

	req := httptest.NewRequest(http.MethodGet, "/living-styleguide", nil)
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowLivingStyleguide(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Styleguide should not call greetings provider
	if fakeProvider.callCount != 0 {
		t.Errorf("expected GetGreeting not called for styleguide page, got %d calls", fakeProvider.callCount)
	}
}

func TestShowStyleguideWorkbench(t *testing.T) {
	t.Parallel()

	fakeProvider := &fakeGreetingsProvider{}

	h := NewHandler(fakeProvider)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"workbench.title": "Workbench"},
	})

	req := httptest.NewRequest(http.MethodGet, "/living-styleguide/workbench", nil)
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowStyleguideWorkbench(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Workbench should not call greetings provider
	if fakeProvider.callCount != 0 {
		t.Errorf("expected GetGreeting not called for workbench page, got %d calls", fakeProvider.callCount)
	}
}

func TestNewHandler_WithNilProvider(t *testing.T) {
	t.Parallel()

	// This test documents that passing nil provider will panic when handler is used
	// In production, wiring should never pass nil
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Handler panicked as expected with nil provider: %v", r)
		}
	}()

	h := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// This should panic when trying to call GetGreeting on nil
	h.ShowHomepage(rr, req)
}

func TestShowHomepage_DifferentGreetingsPerRequest(t *testing.T) {
	t.Parallel()

	greetings := []string{"Greeting 1", "Greeting 2", "Greeting 3"}
	callIndex := 0

	fakeProviderWithRotating := &rotatingGreetingsProvider{
		greetings: greetings,
		index:     &callIndex,
	}

	h := NewHandler(fakeProviderWithRotating)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"homepage.hero.title": "Test Title"},
	})

	for i, expectedGreeting := range greetings {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
		req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
		rr := httptest.NewRecorder()

		h.ShowHomepage(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected status %d, got %d", i, http.StatusOK, rr.Code)
		}

		body := rr.Body.String()
		if !strings.Contains(body, expectedGreeting) {
			t.Errorf("request %d: expected body to contain greeting '%s', got %q", i, expectedGreeting, body)
		}
	}
}

type rotatingGreetingsProvider struct {
	greetings []string
	index     *int
}

func (r *rotatingGreetingsProvider) GetGreeting() config.Greeting {
	greeting := config.Greeting{Text: r.greetings[*r.index]}
	*r.index = (*r.index + 1) % len(r.greetings)
	return greeting
}

// TestGreetingsProviderIntegration tests the real integration between handler and config
func TestGreetingsProviderIntegration(t *testing.T) {
	t.Parallel()

	// Use the real greetings configuration
	cfg := config.NewGreetingsConfiguration()

	h := NewHandler(cfg)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"homepage.hero.title": "Test Title"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
	req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
	rr := httptest.NewRecorder()

	h.ShowHomepage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	// Should contain one of the default greetings
	found := false
	for _, greeting := range config.DefaultGreetings() {
		if strings.Contains(body, greeting.Text) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected body to contain one of the default greetings, got %q", body)
	}
}

// TestGreetingsProviderContextIsolation ensures each request gets independent greeting
func TestGreetingsProviderContextIsolation(t *testing.T) {
	t.Parallel()

	customGreetings := []config.Greeting{
		{Text: "Request 1"},
		{Text: "Request 2"},
	}
	cfg := config.NewGreetingsConfiguration(customGreetings...)

	h := NewHandler(cfg)

	translator := i18n.NewTranslator(i18n.LocaleEN, map[i18n.Locale]map[string]string{
		i18n.LocaleEN: {"homepage.hero.title": "Test Title"},
	})

	var results []string
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(i18n.WithLocale(req.Context(), i18n.LocaleEN))
		req = req.WithContext(i18n.WithTranslator(req.Context(), translator))
		rr := httptest.NewRecorder()

		h.ShowHomepage(rr, req)

		body := rr.Body.String()
		// Extract greeting from response body
		for _, g := range customGreetings {
			if strings.Contains(body, g.Text) {
				results = append(results, g.Text)
				break
			}
		}
	}

	// Verify both greetings were used across requests
	found := make(map[string]bool)
	for _, r := range results {
		found[r] = true
	}
	if len(found) < 2 {
		t.Errorf("expected different greetings across requests, only found %d unique: %v", len(found), found)
	}
}
