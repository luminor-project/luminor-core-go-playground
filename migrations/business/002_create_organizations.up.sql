CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owning_users_id UUID NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_organizations_owner ON organizations(owning_users_id);

CREATE TABLE organization_members (
    account_core_id UUID NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (account_core_id, organization_id)
);

CREATE INDEX idx_organization_members_account ON organization_members(account_core_id);

CREATE TABLE organization_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    access_rights TEXT[] NOT NULL DEFAULT '{}',
    is_default_for_new_members BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_organization_groups_org ON organization_groups(organization_id);

CREATE TABLE organization_group_members (
    account_core_id UUID NOT NULL,
    group_id UUID NOT NULL REFERENCES organization_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (account_core_id, group_id)
);

CREATE INDEX idx_org_group_members_account ON organization_group_members(account_core_id);

CREATE TABLE organization_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_org_invitations_org ON organization_invitations(organization_id);
CREATE INDEX idx_org_invitations_email ON organization_invitations(email);
