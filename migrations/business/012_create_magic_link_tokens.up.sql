CREATE TABLE magic_link_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES account_cores(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_magic_link_tokens_token_hash ON magic_link_tokens(token_hash);
CREATE INDEX idx_magic_link_tokens_account_id ON magic_link_tokens(account_id);
CREATE INDEX idx_magic_link_tokens_expires_at ON magic_link_tokens(expires_at) WHERE used_at IS NULL;
