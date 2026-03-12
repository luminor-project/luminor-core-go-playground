CREATE TABLE password_reset_tokens (
    token_hash VARCHAR(64) PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES account_cores(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_tokens_account ON password_reset_tokens(account_id);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);
