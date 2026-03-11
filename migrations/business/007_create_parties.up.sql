CREATE TABLE parties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_kind TEXT NOT NULL DEFAULT 'human',
    party_kind TEXT NOT NULL,
    name TEXT NOT NULL,
    owning_organization_id UUID NOT NULL,
    created_by_account_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_parties_owning_org ON parties(owning_organization_id);
CREATE INDEX idx_parties_party_kind ON parties(party_kind);
