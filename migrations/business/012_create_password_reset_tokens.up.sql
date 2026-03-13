CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES account_cores(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,  -- SHA-256 = 64 hex characters
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for account lookups
CREATE INDEX idx_password_reset_tokens_account ON password_reset_tokens(account_id);

-- Index for hash lookups (used for token validation)
CREATE UNIQUE INDEX idx_password_reset_tokens_hash ON password_reset_tokens(token_hash);

-- Index for cleanup of expired tokens
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at) WHERE used_at IS NULL;
