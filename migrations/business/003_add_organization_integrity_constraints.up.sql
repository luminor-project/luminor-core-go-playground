ALTER TABLE organization_groups
    ADD CONSTRAINT uq_organization_groups_id_org UNIQUE (id, organization_id);

ALTER TABLE organization_members
    ADD CONSTRAINT uq_organization_members_account_org UNIQUE (account_core_id, organization_id);

ALTER TABLE organization_group_members
    ADD COLUMN organization_id UUID;

UPDATE organization_group_members ogm
SET organization_id = og.organization_id
FROM organization_groups og
WHERE ogm.group_id = og.id;

ALTER TABLE organization_group_members
    ALTER COLUMN organization_id SET NOT NULL;

ALTER TABLE organization_group_members
    ADD CONSTRAINT fk_org_group_members_group_org
        FOREIGN KEY (group_id, organization_id)
        REFERENCES organization_groups(id, organization_id)
        ON DELETE CASCADE;

ALTER TABLE organization_group_members
    ADD CONSTRAINT fk_org_group_members_member_org
        FOREIGN KEY (account_core_id, organization_id)
        REFERENCES organization_members(account_core_id, organization_id)
        ON DELETE CASCADE;

CREATE INDEX idx_org_group_members_org ON organization_group_members(organization_id);

CREATE UNIQUE INDEX uq_org_invitations_org_email
    ON organization_invitations (organization_id, email);
