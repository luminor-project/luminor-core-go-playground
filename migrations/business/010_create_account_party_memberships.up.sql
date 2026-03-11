CREATE TABLE account_party_memberships (
    account_id UUID NOT NULL,
    party_id UUID NOT NULL,
    org_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, party_id)
);

CREATE INDEX idx_apm_party ON account_party_memberships(party_id);
CREATE INDEX idx_apm_org ON account_party_memberships(org_id);

ALTER TABLE account_cores ADD COLUMN currently_active_party_id UUID;

CREATE TABLE account_party_pending_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL UNIQUE,
    party_id UUID NOT NULL,
    org_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
