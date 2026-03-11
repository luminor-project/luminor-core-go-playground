CREATE TABLE subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    owning_organization_id UUID NOT NULL,
    created_by_account_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subjects_owning_org ON subjects(owning_organization_id);
