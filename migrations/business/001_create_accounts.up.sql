CREATE TABLE account_cores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    roles JSONB NOT NULL DEFAULT '["user"]',
    must_set_password BOOLEAN NOT NULL DEFAULT false,
    currently_active_organization_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_account_cores_active_org ON account_cores(currently_active_organization_id);
