package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

func newTestSessionStore() *sessions.CookieStore {
	return sessions.NewCookieStore([]byte("test-session-secret-12345678901234567890"))
}

type fakeAccountUseCases struct {
	authenticateFunc          func(ctx context.Context, email, password string) (facade.AccountInfoDTO, error)
	authenticateCalls         int
	mustSetPasswordFunc       func(ctx context.Context, email string) (bool, error)
	mustSetPasswordCalls      int
	registerFunc              func(ctx context.Context, dto facade.RegistrationDTO) (string, error)
	registerCalls             int
	getAccountInfoByIDFunc    func(ctx context.Context, id string) (facade.AccountInfoDTO, error)
	getAccountInfoByIDCalls   int
	setPasswordFunc           func(ctx context.Context, accountID, newPassword string) error
	setPasswordCalls          int
	requestPasswordResetFunc  func(ctx context.Context, dto facade.PasswordResetRequestDTO) error
	completePasswordResetFunc func(ctx context.Context, dto facade.PasswordResetCompletionDTO) error
}

func (f *fakeAccountUseCases) Authenticate(ctx context.Context, email, password string) (facade.AccountInfoDTO, error) {
	f.authenticateCalls++
	if f.authenticateFunc != nil {
		return f.authenticateFunc(ctx, email, password)
	}
	return facade.AccountInfoDTO{}, nil
}

func (f *fakeAccountUseCases) MustSetPassword(ctx context.Context, email string) (bool, error) {
	f.mustSetPasswordCalls++
	if f.mustSetPasswordFunc != nil {
		return f.mustSetPasswordFunc(ctx, email)
	}
	return false, nil
}

func (f *fakeAccountUseCases) Register(ctx context.Context, dto facade.RegistrationDTO) (string, error) {
	f.registerCalls++
	if f.registerFunc != nil {
		return f.registerFunc(ctx, dto)
	}
	return "", nil
}

func (f *fakeAccountUseCases) GetAccountInfoByID(ctx context.Context, id string) (facade.AccountInfoDTO, error) {
	f.getAccountInfoByIDCalls++
	if f.getAccountInfoByIDFunc != nil {
		return f.getAccountInfoByIDFunc(ctx, id)
	}
	return facade.AccountInfoDTO{}, nil
}

func (f *fakeAccountUseCases) SetPassword(ctx context.Context, accountID, newPassword string) error {
	f.setPasswordCalls++
	if f.setPasswordFunc != nil {
		return f.setPasswordFunc(ctx, accountID, newPassword)
	}
	return nil
}

func (f *fakeAccountUseCases) RequestPasswordReset(ctx context.Context, dto facade.PasswordResetRequestDTO) error {
	if f.requestPasswordResetFunc != nil {
		return f.requestPasswordResetFunc(ctx, dto)
	}
	return nil
}

func (f *fakeAccountUseCases) CompletePasswordReset(ctx context.Context, dto facade.PasswordResetCompletionDTO) error {
	if f.completePasswordResetFunc != nil {
		return f.completePasswordResetFunc(ctx, dto)
	}
	return nil
}

func TestHandleSignIn_InvalidCredentials(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{
		authenticateFunc: func(_ context.Context, _, _ string) (facade.AccountInfoDTO, error) {
			return facade.AccountInfoDTO{}, domain.ErrInvalidCredentials
		},
	}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "wrongpassword")
	req := httptest.NewRequest(http.MethodPost, "/sign-in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSignIn(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "auth.validation.invalidCredentials") {
		t.Fatalf("expected body to contain invalidCredentials key, got %q", rr.Body.String())
	}
	if fake.authenticateCalls != 1 {
		t.Fatalf("expected Authenticate called once, got %d", fake.authenticateCalls)
	}
}

func TestHandleSignIn_InternalError(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{
		authenticateFunc: func(_ context.Context, _, _ string) (facade.AccountInfoDTO, error) {
			return facade.AccountInfoDTO{}, errors.New("database down")
		},
	}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "somepassword")
	req := httptest.NewRequest(http.MethodPost, "/sign-in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSignIn(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	if fake.authenticateCalls != 1 {
		t.Fatalf("expected Authenticate called once, got %d", fake.authenticateCalls)
	}
}

func TestHandleSignUp_PasswordMismatch(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("email", "new@example.com")
	form.Set("password", "password123")
	form.Set("password_confirm", "differentpassword")
	req := httptest.NewRequest(http.MethodPost, "/sign-up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSignUp(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "auth.validation.passwordsDoNotMatch") {
		t.Fatalf("expected body to contain passwordsDoNotMatch key, got %q", rr.Body.String())
	}
	if fake.registerCalls != 0 {
		t.Fatalf("expected Register not called, got %d calls", fake.registerCalls)
	}
}

func TestHandleSignUp_ValidationError(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{
		registerFunc: func(_ context.Context, _ facade.RegistrationDTO) (string, error) {
			return "", &domain.ValidationError{Key: "auth.validation.emailTaken"}
		},
	}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("email", "taken@example.com")
	form.Set("password", "password123")
	form.Set("password_confirm", "password123")
	req := httptest.NewRequest(http.MethodPost, "/sign-up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSignUp(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "auth.validation.emailTaken") {
		t.Fatalf("expected body to contain emailTaken key, got %q", rr.Body.String())
	}
	if fake.registerCalls != 1 {
		t.Fatalf("expected Register called once, got %d", fake.registerCalls)
	}
}

func TestHandleSignUp_InternalError(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{
		registerFunc: func(_ context.Context, _ facade.RegistrationDTO) (string, error) {
			return "", errors.New("unexpected failure")
		},
	}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "password123")
	form.Set("password_confirm", "password123")
	req := httptest.NewRequest(http.MethodPost, "/sign-up", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSignUp(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	if fake.registerCalls != 1 {
		t.Fatalf("expected Register called once, got %d", fake.registerCalls)
	}
}

func TestHandleSignOut_ClearsSession(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{}

	h := NewHandler(fake, newTestSessionStore(), nil)

	req := httptest.NewRequest(http.MethodPost, "/sign-out", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "user-1", Email: "test@example.com"}))
	rr := httptest.NewRecorder()

	h.HandleSignOut(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/en" {
		t.Fatalf("expected redirect to /en, got %q", got)
	}
}

func TestHandleSetPassword_PasswordMismatch(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("password", "newpassword123")
	form.Set("password_confirm", "differentpassword")
	req := httptest.NewRequest(http.MethodPost, "/set-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "user-1", Email: "test@example.com"}))
	rr := httptest.NewRecorder()

	h.HandleSetPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "auth.validation.passwordsDoNotMatch") {
		t.Fatalf("expected body to contain passwordsDoNotMatch key, got %q", rr.Body.String())
	}
	if fake.setPasswordCalls != 0 {
		t.Fatalf("expected SetPassword not called, got %d calls", fake.setPasswordCalls)
	}
}

func TestHandleSetPassword_PasswordTooShort(t *testing.T) {
	t.Parallel()

	fake := &fakeAccountUseCases{
		setPasswordFunc: func(_ context.Context, _, _ string) error {
			return domain.ErrPasswordTooShort
		},
	}

	h := NewHandler(fake, newTestSessionStore(), nil)

	form := url.Values{}
	form.Set("password", "short")
	form.Set("password_confirm", "short")
	req := httptest.NewRequest(http.MethodPost, "/set-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "user-1", Email: "test@example.com"}))
	rr := httptest.NewRecorder()

	h.HandleSetPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "auth.validation.passwordTooShort") {
		t.Fatalf("expected body to contain passwordTooShort key, got %q", rr.Body.String())
	}
	if fake.setPasswordCalls != 1 {
		t.Fatalf("expected SetPassword called once, got %d", fake.setPasswordCalls)
	}
}
