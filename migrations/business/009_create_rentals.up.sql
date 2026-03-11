CREATE TABLE rentals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL,
    tenant_party_id UUID NOT NULL,
    org_id UUID NOT NULL,
    created_by_account_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_rentals_subject_tenant ON rentals(subject_id, tenant_party_id);
CREATE INDEX idx_rentals_subject ON rentals(subject_id);
CREATE INDEX idx_rentals_tenant ON rentals(tenant_party_id);
CREATE INDEX idx_rentals_org ON rentals(org_id);
