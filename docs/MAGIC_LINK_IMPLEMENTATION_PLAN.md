# Magic Mail Login Link Implementation Plan

## Executive Summary

This plan outlines the implementation of a magic mail login link feature that allows users to authenticate via secure, time-limited tokens sent to their email. The feature integrates with the existing session-based authentication system and follows the vertical slice architecture patterns established in the codebase.

**Key Findings from Codebase Analysis:**

- Session-based auth with Gorilla Sessions (30-day duration)
- No existing email infrastructure (needs to be built)
- Vertical slice architecture with CRUD pattern for account vertical
- No rate limiting currently implemented
- templ + htmx + Alpine.js frontend stack

---

## 1. Database Schema

### 1.1 Migration File

**File:** `migrations/business/012_create_magic_tokens.up.sql`

```sql
CREATE TABLE magic_login_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES account_cores(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL, -- bcrypt hash of the token (never store raw tokens)
    email TEXT NOT NULL, -- Normalized email for audit/debugging
    redirect_path TEXT DEFAULT '/dashboard', -- Where to redirect after successful login
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL, -- Must be explicitly set
    used_at TIMESTAMPTZ, -- NULL until used, then timestamp
    ip_address_created TEXT, -- Audit trail
    user_agent_created TEXT, -- Audit trail
    ip_address_used TEXT, -- Audit trail when used
    user_agent_used TEXT -- Audit trail when used
);

-- Indexes for performance and cleanup
CREATE INDEX idx_magic_tokens_account_id ON magic_login_tokens(account_id);
CREATE INDEX idx_magic_tokens_expires_at ON magic_login_tokens(expires_at);
CREATE INDEX idx_magic_tokens_used_at ON magic_login_tokens(used_at) WHERE used_at IS NULL;
CREATE UNIQUE INDEX idx_magic_tokens_valid_token ON magic_login_tokens(token_hash) WHERE used_at IS NULL;

-- Cleanup job can use this to remove old used/expired tokens
CREATE INDEX idx_magic_tokens_cleanup ON magic_login_tokens(used_at, expires_at);
```

**File:** `migrations/business/012_create_magic_tokens.down.sql`

```sql
DROP TABLE IF EXISTS magic_login_tokens;
```

### 1.2 Schema Design Rationale

- **token_hash stores bcrypt hash, not raw token**: Prevents token theft even if database is compromised
- **Separate used_at timestamp**: Allows audit trail and distinguishes consumed vs expired tokens
- **Audit fields**: Track creation and consumption metadata for security monitoring
- **Cascading delete**: Tokens automatically removed when account is deleted
- **Unique constraint on unused tokens**: Prevents duplicate active tokens for same hash (shouldn't happen with random tokens, but safety measure)

---

## 2. API Endpoint Design

### 2.1 Request Link Endpoint

**Route:** `POST /auth/magic-link/request`

**Handler:** `internal/account/web/handler.go` - new method `HandleMagicLinkRequest()`

**Flow:**

1. Parse email from form submission
2. Normalize and validate email format
3. **CRITICAL**: Always return success response regardless of whether email exists (prevents email enumeration)
4. If account exists:
    - Generate cryptographically secure random token (32 bytes, base64url encoded)
    - Hash token with bcrypt (cost 12 for security, slower than password cost 10)
    - Store hash with expiration (15 minutes)
    - Send email with raw token in URL
5. Redirect to confirmation page

**Request Format:**

```
POST /en/auth/magic-link/request
Content-Type: application/x-www-form-urlencoded

csrf_token=<token>&email=user@example.com&redirect=/specific-page
```

**Response:**

- Success: Redirect to `/auth/magic-link/sent` (same page whether email exists or not)
- Error (invalid email format): Return to form with validation error

### 2.2 Validate Token Endpoint

**Route:** `GET /auth/magic-link/verify?token=<token>&redirect=<path>`

**Handler:** `internal/account/web/handler.go` - new method `HandleMagicLinkVerify()`

**Flow:**

1. Extract token from query parameter
2. Look up token hash in database (iterate through unused tokens, bcrypt compare each - slow but secure)
3. Validate:
    - Token exists and hasn't been used
    - Token hasn't expired
    - Associated account exists and is active
4. Create session (same as password login)
5. Mark token as used with consumption metadata
6. Redirect to specified path or default dashboard

**Security Considerations:**

- Token validation must be idempotent (if already used, treat as invalid)
- Clear any existing session before creating new one (prevents session fixation)
- Log IP and user agent of consumption for audit

### 2.3 Confirmation Page

**Route:** `GET /auth/magic-link/sent`

**Handler:** `internal/account/web/handler.go` - new method `ShowMagicLinkSent()`

**Purpose:** Generic confirmation page shown after requesting link. Does not reveal whether email exists in system.

### 2.4 Routes Registration

**File:** `internal/account/web/routes.go`

Add to existing routes:

```go
// Guest-only routes (RequireGuest middleware)
mux.Handle("GET /auth/magic-link", auth.RequireGuest(http.HandlerFunc(h.ShowMagicLinkRequest)))
mux.Handle("POST /auth/magic-link/request", auth.RequireGuest(http.HandlerFunc(h.HandleMagicLinkRequest)))
mux.Handle("GET /auth/magic-link/sent", auth.RequireGuest(http.HandlerFunc(h.ShowMagicLinkSent)))
mux.Handle("GET /auth/magic-link/verify", auth.RequireGuest(http.HandlerFunc(h.HandleMagicLinkVerify)))
```

---

## 3. Email Template and Delivery Mechanism

### 3.1 Email Service Infrastructure

Since no email infrastructure exists, we need to create a new platform service:

**File:** `internal/platform/email/service.go`

```go
package email

type Service interface {
    SendMagicLink(ctx context.Context, to, token, redirectPath string) error
    // Future expansion: SendPasswordReset, SendInvitation, etc.
}

type Config struct {
    SMTPHost     string
    SMTPPort     int
    SMTPUsername string
    SMTPPassword string
    FromAddress  string
    FromName     string
    BaseURL      string // For constructing magic link URLs
}
```

**File:** `internal/platform/email/smtp.go` (Implementation)

```go
package email

import (
    "bytes"
    "context"
    "fmt"
    "html/template"
    "net/smtp"

    "github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

type smtpService struct {
    config Config
}

func NewSMTPService(config Config) Service {
    return &smtpService{config: config}
}

func (s *smtpService) SendMagicLink(ctx context.Context, to, token, redirectPath string) error {
    // Load and execute template
    magicLinkURL := fmt.Sprintf("%s/auth/magic-link/verify?token=%s&redirect=%s",
        s.config.BaseURL, token, redirectPath)

    data := map[string]string{
        "MagicLinkURL": magicLinkURL,
        "Email":       to,
        "ExpiresIn":   "15 minutes",
    }

    // Send email via SMTP
    // ... SMTP implementation
}
```

**Configuration:** Add to `internal/platform/config/config.go`:

```go
type Config struct {
    // ... existing fields
    SMTPHost         string `env:"SMTP_HOST"`
    SMTPPort         int    `env:"SMTP_PORT" envDefault:"587"`
    SMTPUsername     string `env:"SMTP_USERNAME"`
    SMTPPassword     string `env:"SMTP_PASSWORD"`
    EmailFromAddress string `env:"EMAIL_FROM_ADDRESS" envDefault:"noreply@luminor.app"`
    EmailFromName    string `env:"EMAIL_FROM_NAME" envDefault:"Luminor"`
}
```

### 3.2 Email Templates

**File:** `internal/platform/email/templates/magic_link.templ`

```templ
package templates

import "github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"

templ MagicLinkEmail(magicLinkURL, email, expiresIn string) {
    <!DOCTYPE html>
    <html>
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>{ i18n.T(ctx, "email.magicLink.subject") }</title>
        <style>
            /* Responsive, accessible email styles */
            body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
            .container { max-width: 600px; margin: 0 auto; padding: 20px; }
            .button { display: inline-block; padding: 12px 24px; background-color: #4f46e5; color: white; text-decoration: none; border-radius: 6px; }
            .link { color: #4f46e5; word-break: break-all; }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>{ i18n.T(ctx, "email.magicLink.heading") }</h1>
            <p>{ i18n.T(ctx, "email.magicLink.greeting") }</p>
            <p>{ i18n.T(ctx, "email.magicLink.instructions") }</p>
            <p style="text-align: center; margin: 30px 0;">
                <a href={ magicLinkURL } class="button">{ i18n.T(ctx, "email.magicLink.button") }</a>
            </p>
            <p>{ i18n.T(ctx, "email.magicLink.expires", "expiresIn", expiresIn) }</p>
            <p>{ i18n.T(ctx, "email.magicLink.fallback") }</p>
            <p class="link">{ magicLinkURL }</p>
            <hr>
            <p style="font-size: 12px; color: #666;">
                { i18n.T(ctx, "email.magicLink.securityNotice") }
            </p>
        </div>
    </body>
    </html>
}
```

**Text version:** Also create plain text template for email clients that don't support HTML.

**File:** `internal/platform/email/templates/magic_link.txt`

```
{{i18n "email.magicLink.heading"}}

{{i18n "email.magicLink.greeting"}}

{{i18n "email.magicLink.instructions"}}

{{i18n "email.magicLink.button"}}: {{.MagicLinkURL}}

{{i18n "email.magicLink.expires" .ExpiresIn}}

{{i18n "email.magicLink.securityNotice"}}
```

### 3.3 Localization

**File:** `internal/platform/i18n/locales/en.json` (additions)

```json
{
    "email.magicLink.subject": "Your magic login link",
    "email.magicLink.heading": "Sign in to Luminor",
    "email.magicLink.greeting": "Hello!",
    "email.magicLink.instructions": "Click the button below to sign in. This link will expire in {expiresIn}.",
    "email.magicLink.button": "Sign In",
    "email.magicLink.expires": "This link expires in {expiresIn}.",
    "email.magicLink.fallback": "If the button doesn't work, copy and paste this link into your browser:",
    "email.magicLink.securityNotice": "If you didn't request this link, you can safely ignore this email. Someone may have entered your email address by mistake.",
    "auth.magicLink.pageTitle": "Sign in with email",
    "auth.magicLink.description": "Enter your email address and we'll send you a secure sign-in link.",
    "auth.magicLink.submit": "Send magic link",
    "auth.magicLink.emailPlaceholder": "you@example.com",
    "auth.magicLink.sentTitle": "Check your email",
    "auth.magicLink.sentDescription": "If an account exists with that email, we've sent a magic link to sign in. The link expires in 15 minutes.",
    "auth.magicLink.invalidToken": "This sign-in link is invalid or has expired. Please request a new one.",
    "auth.magicLink.success": "Successfully signed in!",
    "error.invalidEmail": "Please enter a valid email address."
}
```

Add equivalent entries to `de.json` and `fr.json`.

---

## 4. Security Considerations

### 4.1 Token Generation

**Implementation:** `internal/account/domain/service.go` additions

```go
import (
    "crypto/rand"
    "encoding/base64"
)

const (
    MagicTokenBytes = 32 // 256 bits of entropy
    MagicTokenTTL   = 15 * time.Minute
    BcryptCost      = 12 // Higher than password cost since tokens are single-use
)

func generateSecureToken() (rawToken string, hashedToken string, err error) {
    bytes := make([]byte, MagicTokenBytes)
    if _, err := rand.Read(bytes); err != nil {
        return "", "", err
    }

    rawToken = base64.URLEncoding.EncodeToString(bytes)
    hash, err := bcrypt.GenerateFromPassword([]byte(rawToken), BcryptCost)
    if err != nil {
        return "", "", err
    }

    return rawToken, string(hash), nil
}
```

**Security Properties:**

- 256 bits of entropy (base64url = ~43 characters)
- Cryptographically secure random generation
- bcrypt hashing prevents rainbow table attacks if DB compromised
- Cost factor 12 balances security with performance (tokens validated infrequently)

### 4.2 Token Expiration

- **TTL: 15 minutes** - Short enough to prevent prolonged exposure
- Server-enforced expiration via `expires_at` column
- Tokens valid for single use only (consumed immediately on first valid use)

### 4.3 Rate Limiting (NEW INFRASTRUCTURE)

Since no rate limiting exists, implement a simple in-memory solution for magic links:

**File:** `internal/platform/ratelimit/memory.go`

```go
package ratelimit

import (
    "context"
    "sync"
    "time"
)

type Limiter interface {
    Allow(ctx context.Context, key string, limit Rate) (bool, time.Duration)
}

type Rate struct {
    Requests int
    Window   time.Duration
}

type memoryLimiter struct {
    mu      sync.RWMutex
    windows map[string][]time.Time
}

func NewMemoryLimiter() Limiter {
    return &memoryLimiter{windows: make(map[string][]time.Time)}
}

func (m *memoryLimiter) Allow(ctx context.Context, key string, limit Rate) (bool, time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-limit.Window)

    // Clean old entries and count current
    var valid []time.Time
    for _, t := range m.windows[key] {
        if t.After(cutoff) {
            valid = append(valid, t)
        }
    }

    if len(valid) >= limit.Requests {
        // Rate limit exceeded, return time until oldest request expires
        retryAfter := valid[0].Sub(cutoff)
        m.windows[key] = valid
        return false, retryAfter
    }

    valid = append(valid, now)
    m.windows[key] = valid
    return true, 0
}
```

**Rate Limits for Magic Links:**

- Per email: 3 requests per 15 minutes
- Per IP: 10 requests per 15 minutes
- Stricter limits prevent abuse and email spam

### 4.4 Additional Security Measures

1. **Email Enumeration Prevention**: Always return same success message whether email exists or not
2. **Audit Logging**: Log all token creations and consumptions with IP/user agent
3. **Session Fixation Protection**: Clear any existing session before creating new one
4. **HTTPS Enforcement**: Magic links should only work over HTTPS (already enforced by middleware)
5. **Token Visibility**: Never log raw tokens (only hashed versions)
6. **Concurrent Use Prevention**: Token consumption is atomic (UPDATE ... WHERE used_at IS NULL)

---

## 5. Frontend Flow and Components

### 5.1 Magic Link Request Page

**File:** `internal/account/web/templates/magic_link_request.templ`

```templ
package templates

import (
    "github.com/luminor-project/luminor-core-go-playground/internal/common/web/templates/layouts"
    "github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

templ MagicLinkRequest(csrfToken, errorMsg, lastEmail string) {
    @layouts.AppShell(i18n.T(ctx, "auth.magicLink.pageTitle")) {
        @layouts.CenteredPanel() {
            @layouts.AuthHeader(
                i18n.T(ctx, "auth.magicLink.pageTitle"),
                i18n.T(ctx, "auth.magicLink.description"),
            )

            if errorMsg != "" {
                @layouts.StandardAlert("error", errorMsg)
            }

            @layouts.SurfaceCard() {
                <form action={ i18n.LocalizedPath(ctx, "/auth/magic-link/request") }
                      method="post"
                      class="space-y-6">

                    <input type="hidden" name="csrf_token" value={ csrfToken }/>

                    <!-- Optional: preserve redirect parameter -->
                    if redirect := ctx.Value("redirect_path"); redirect != nil {
                        <input type="hidden" name="redirect" value={ redirect.(string) }/>
                    }

                    @layouts.FormField(
                        "email",
                        "email",
                        i18n.T(ctx, "auth.common.email"),
                        "email",
                        i18n.T(ctx, "auth.magicLink.emailPlaceholder"),
                        "email",
                        lastEmail,
                        true,
                    )

                    <button type="submit"
                            class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
                        { i18n.T(ctx, "auth.magicLink.submit") }
                    </button>
                </form>

                <div class="mt-6 text-center">
                    <p class="text-sm text-gray-600">
                        { i18n.T(ctx, "auth.common.or") }
                        <a href={ i18n.LocalizedPath(ctx, "/sign-in") }
                           class="font-medium text-indigo-600 hover:text-indigo-500">
                            { i18n.T(ctx, "auth.common.signInWithPassword") }
                        </a>
                    </p>
                </div>
            }
        }
    }
}
```

### 5.2 Magic Link Sent Confirmation Page

**File:** `internal/account/web/templates/magic_link_sent.templ`

```templ
package templates

import (
    "github.com/luminor-project/luminor-core-go-playground/internal/common/web/templates/layouts"
    "github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

templ MagicLinkSent() {
    @layouts.AppShell(i18n.T(ctx, "auth.magicLink.sentTitle")) {
        @layouts.CenteredPanel() {
            @layouts.AuthHeader(
                i18n.T(ctx, "auth.magicLink.sentTitle"),
                i18n.T(ctx, "auth.magicLink.sentDescription"),
            )

            @layouts.SurfaceCard() {
                <div class="text-center space-y-4">
                    <svg class="mx-auto h-12 w-12 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>

                    <p class="text-sm text-gray-600">
                        { i18n.T(ctx, "auth.magicLink.sentDescription") }
                    </p>

                    <div class="mt-6">
                        <a href={ i18n.LocalizedPath(ctx, "/auth/magic-link") }
                           class="text-sm font-medium text-indigo-600 hover:text-indigo-500">
                            { i18n.T(ctx, "auth.magicLink.requestAnother") }
                        </a>
                    </div>
                </div>
            }

            <div class="mt-6 text-center">
                <p class="text-xs text-gray-500">
                    { i18n.T(ctx, "auth.magicLink.emailHelp") }
                </p>
            </div>
        }
    }
}
```

### 5.3 Update Existing Sign-In Page

**File:** `internal/account/web/templates/sign_in.templ` (add magic link option)

Add below the password sign-in form:

```templ
<div class="mt-6">
    <div class="relative">
        <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-gray-300"></div>
        </div>
        <div class="relative flex justify-center text-sm">
            <span class="px-2 bg-white text-gray-500">{ i18n.T(ctx, "auth.common.orContinueWith") }</span>
        </div>
    </div>

    <div class="mt-6">
        <a href={ i18n.LocalizedPath(ctx, "/auth/magic-link") }
           class="w-full flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-500 hover:bg-gray-50">
            { i18n.T(ctx, "auth.magicLink.buttonLabel") }
        </a>
    </div>
</div>
```

### 5.4 Client-Side Considerations

Since we use htmx/Alpine.js:

1. **Progressive Enhancement**: Form works without JavaScript (full page reload)
2. **Loading States**: Add htmx loading indicators during form submission
3. **Auto-Redirect Detection**: Could use Alpine.js to detect when user returns from email client:

```templ
<!-- On verify page, optional enhancement -->
<div x-data="{ checking: true }" x-init="setTimeout(() => checking = false, 2000)">
    <div x-show="checking" class="text-center">
        <p>Validating your link...</p>
    </div>
</div>
```

---

## 6. Integration with Existing Authentication Infrastructure

### 6.1 Leverage Existing Components

1. **Session Management** (`internal/platform/session/store.go`)
    - Use existing `session.SetAuthenticated(w, r, h.sessionStore, accountID, email, roles, activePartyInfo)`
    - No changes needed to session infrastructure

2. **Auth Middleware** (`internal/platform/auth/middleware.go`)
    - Reuse `RequireGuest` middleware for magic link pages
    - After successful verification, user session works with existing `RequireAuth` middleware

3. **Flash Messages** (`internal/platform/flash/store.go`)
    - Use existing flash message system for success/error feedback
    - Example: `flash.SetKey(w, r, h.sessionStore, flash.TypeSuccess, "flash.auth.welcomeBack")`

4. **CSRF Protection** (`internal/platform/csrf/csrf.go`)
    - All forms already include CSRF tokens
    - Magic link request form continues this pattern

5. **User Context** (`internal/platform/auth/context.go`)
    - After magic link login, user loaded into context same as password login
    - All downstream handlers work unchanged

### 6.2 Account Vertical Integration

**Facade Extension:** `internal/account/facade/impl.go`

Add new use case methods to existing `accountUseCases` interface:

```go
type accountUseCases interface {
    // ... existing methods

    // Magic link methods
    RequestMagicLink(ctx context.Context, email, redirectPath, ipAddress, userAgent string) (accountExists bool, err error)
    VerifyMagicLink(ctx context.Context, rawToken, ipAddress, userAgent string) (AccountInfoDTO, error)
    GetAccountForMagicLink(ctx context.Context, email string) (AccountInfoDTO, error)
}
```

**Service Implementation:** `internal/account/domain/service.go`

```go
func (s *Service) RequestMagicLink(ctx context.Context, email, redirectPath, ip, userAgent string) (bool, error) {
    // 1. Find account by email
    account, err := s.repo.FindByEmail(ctx, email)
    if err != nil {
        if errors.Is(err, ErrAccountNotFound) {
            return false, nil // Don't reveal account doesn't exist
        }
        return false, err
    }

    // 2. Generate token
    rawToken, hash, err := generateSecureToken()
    if err != nil {
        return false, err
    }

    // 3. Store token
    expiresAt := s.clock.Now().Add(MagicTokenTTL)
    err = s.repo.CreateMagicToken(ctx, account.ID, hash, email, redirectPath, expiresAt, ip, userAgent)
    if err != nil {
        return false, err
    }

    // 4. Send email (via injected email service)
    err = s.emailService.SendMagicLink(ctx, email, rawToken, redirectPath)
    if err != nil {
        // Log but don't fail - user can request another link
        s.logger.Error("failed to send magic link email", "error", err, "email", email)
    }

    return true, nil
}

func (s *Service) VerifyMagicLink(ctx context.Context, rawToken, ip, userAgent string) (AccountInfoDTO, error) {
    // 1. Find unused tokens for this time period
    tokens, err := s.repo.FindValidMagicTokens(ctx, s.clock.Now())
    if err != nil {
        return AccountInfoDTO{}, err
    }

    // 2. Brute force check each token hash (slow but secure)
    var matchedToken *MagicToken
    for _, t := range tokens {
        if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(rawToken)); err == nil {
            matchedToken = &t
            break
        }
    }

    if matchedToken == nil {
        return AccountInfoDTO{}, ErrInvalidMagicToken
    }

    // 3. Mark as used (atomic operation)
    err = s.repo.ConsumeMagicToken(ctx, matchedToken.ID, ip, userAgent, s.clock.Now())
    if err != nil {
        return AccountInfoDTO{}, err
    }

    // 4. Get account info
    return s.repo.GetAccountInfo(ctx, matchedToken.AccountID)
}
```

### 6.3 Repository Additions

**File:** `internal/account/infra/repo.go`

Add methods to existing PostgresRepository:

```go
func (r *PostgresRepository) CreateMagicToken(ctx context.Context, accountID, hash, email, redirect string, expires time.Time, ip, ua string) error {
    _, err := r.pool.Exec(ctx, `
        INSERT INTO magic_login_tokens (account_id, token_hash, email, redirect_path, expires_at, ip_address_created, user_agent_created)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, accountID, hash, email, redirect, expires, ip, ua)
    return err
}

func (r *PostgresRepository) FindValidMagicTokens(ctx context.Context, now time.Time) ([]MagicToken, error) {
    rows, err := r.pool.Query(ctx, `
        SELECT id, account_id, token_hash, redirect_path
        FROM magic_login_tokens
        WHERE used_at IS NULL AND expires_at > $1
    `, now)
    // ... scan rows
}

func (r *PostgresRepository) ConsumeMagicToken(ctx context.Context, tokenID, ip, ua string, usedAt time.Time) error {
    // Use UPDATE with WHERE to ensure atomic consumption
    tag, err := r.pool.Exec(ctx, `
        UPDATE magic_login_tokens
        SET used_at = $1, ip_address_used = $2, user_agent_used = $3
        WHERE id = $4 AND used_at IS NULL
    `, usedAt, ip, ua, tokenID)

    if tag.RowsAffected() == 0 {
        return ErrTokenAlreadyUsed // Someone else consumed it first
    }
    return err
}
```

---

## 7. Testing Strategy

### 7.1 Unit Tests

**File:** `internal/account/domain/service_test.go`

```go
func TestRequestMagicLink_Success(t *testing.T) {
    // Use fixed clock for deterministic testing
    clock := clock.NewFixed(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

    // Test token generation
    // Test email is sent
    // Test token stored with correct expiration
}

func TestVerifyMagicLink_Success(t *testing.T) {
    // Test valid token creates session
    // Test token marked as used
    // Test audit fields populated
}

func TestVerifyMagicLink_ExpiredToken(t *testing.T) {
    // Test expired token rejected
}

func TestVerifyMagicLink_AlreadyUsed(t *testing.T) {
    // Test consumed token rejected
}
```

### 7.2 Handler Tests

**File:** `internal/account/web/handler_test.go`

```go
func TestHandleMagicLinkRequest(t *testing.T) {
    // Test form parsing
    // Test email validation
    // Test rate limiting
    // Test always redirects to confirmation page
}

func TestHandleMagicLinkVerify(t *testing.T) {
    // Test valid token creates session and redirects
    // Test invalid token shows error
    // Test redirect parameter honored
}
```

### 7.3 Integration Tests

**File:** Test full flow:

1. Request magic link
2. Extract token from mock email service
3. Visit verification URL
4. Verify session created
5. Verify token consumed in DB

---

## 8. Implementation Phases

### Phase 1: Foundation (Day 1-2)

1. Database migration for `magic_login_tokens` table
2. Create email service infrastructure (interface + SMTP implementation)
3. Email template (templ) with localization
4. Add configuration for email settings

### Phase 2: Core Logic (Day 3-4)

1. Extend account domain service with token generation/validation
2. Extend repository with token CRUD operations
3. Rate limiter implementation
4. Unit tests for domain logic

### Phase 3: HTTP Layer (Day 5)

1. Handler methods for request/verify/sent pages
2. Route registration
3. Frontend templates (templ files)
4. Handler tests

### Phase 4: Integration & Polish (Day 6)

1. Wire everything together in `cmd/server/main.go`
2. Add magic link button to existing sign-in page
3. Integration tests
4. Manual testing
5. Code review & quality checks

### Phase 5: Security Review (Day 7)

1. Security audit of token handling
2. Rate limiting verification
3. Email enumeration testing
4. Session handling verification

---

## 9. Monitoring and Maintenance

### 9.1 Metrics to Track

1. **Magic link requests per hour/day** - Detect abuse
2. **Success rate** - % of tokens successfully consumed
3. **Time-to-use** - How quickly users click links after receiving
4. **Expired tokens** - % that expire unused
5. **Email delivery failures** - Monitor email service health

### 9.2 Database Cleanup Job

**File:** New worker or scheduled task

```go
// Cleanup tokens older than 30 days
func CleanupExpiredMagicTokens(ctx context.Context, db *pgxpool.Pool) error {
    _, err := db.Exec(ctx, `
        DELETE FROM magic_login_tokens
        WHERE used_at IS NOT NULL
           OR expires_at < NOW() - INTERVAL '30 days'
    `)
    return err
}
```

Schedule to run daily via existing worker infrastructure or cron.

### 9.3 Security Alerts

Monitor for:

- Sudden spike in magic link requests (DDoS/email spam)
- Multiple failed verification attempts (brute force)
- Tokens consumed from different IP than created (account takeover attempt)

---

## 10. File Structure Summary

```
internal/
├── platform/
│   ├── email/
│   │   ├── service.go              # Interface
│   │   ├── smtp.go                 # SMTP implementation
│   │   └── templates/
│   │       ├── magic_link.templ    # HTML email template
│   │       └── magic_link.txt      # Text email template
│   └── ratelimit/
│       └── memory.go               # In-memory rate limiter
├── account/
│   ├── domain/
│   │   ├── service.go              # Extended with magic link logic
│   │   ├── entities.go             # MagicToken entity (if needed)
│   │   └── errors.go               # New error types
│   ├── facade/
│   │   ├── interface.go            # Extended interface
│   │   └── impl.go                 # Extended implementation
│   ├── infra/
│   │   └── repo.go                 # Extended repository
│   ├── web/
│   │   ├── handler.go              # Extended handlers
│   │   ├── routes.go               # Extended routes
│   │   └── templates/
│   │       ├── magic_link_request.templ
│   │       └── magic_link_sent.templ
│   └── testharness/
│       └── fixtures.go             # Test fixtures
├── platform/i18n/locales/
│   ├── en.json                     # Extended with magic link translations
│   ├── de.json
│   └── fr.json
migrations/business/
├── 012_create_magic_tokens.up.sql
└── 012_create_magic_tokens.down.sql
```

---

## 11. Key Design Decisions

### 11.1 Why bcrypt for token hashing?

Unlike password hashing where we need to verify frequently, magic tokens are validated once. Bcrypt cost 12 provides strong protection against hash cracking if database is compromised, while single-use nature means performance impact is minimal.

### 11.2 Why no email enumeration prevention?

Magic link systems inherently prevent enumeration by showing the same confirmation page regardless of email existence. This is crucial for security.

### 11.3 Why 15-minute expiration?

Balance between security (short exposure window) and UX (enough time for email delivery and user action). Industry standard ranges from 10-60 minutes; 15 is conservative but practical.

### 11.4 Why not use event sourcing?

Magic tokens are ephemeral operational data, not domain state. CRUD pattern is appropriate here as tokens don't represent business domain concepts that need audit history or complex state transitions.

---

## 12. Risk Assessment

| Risk                    | Likelihood | Impact    | Mitigation                                              |
| ----------------------- | ---------- | --------- | ------------------------------------------------------- |
| Email delivery failure  | Medium     | High      | Log errors, allow retry, fallback to password login     |
| Token leakage via email | Low        | High      | Short expiration, single-use, HTTPS only                |
| Rate limit bypass       | Low        | Medium    | IP + email dual rate limiting, monitoring alerts        |
| Session fixation        | Low        | High      | Clear existing session before creating new one          |
| Token enumeration       | Low        | Medium    | Constant-time hash comparison, no error differentiation |
| Database compromise     | Very Low   | Very High | Token hashing prevents immediate token use              |

---

This implementation plan provides a complete, secure, and architecturally consistent solution for magic mail login links that integrates seamlessly with the existing Luminor application.
