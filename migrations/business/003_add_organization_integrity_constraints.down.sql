DROP INDEX IF EXISTS uq_org_invitations_org_email;
DROP INDEX IF EXISTS idx_org_group_members_org;

ALTER TABLE organization_group_members
    DROP CONSTRAINT IF EXISTS fk_org_group_members_member_org;

ALTER TABLE organization_group_members
    DROP CONSTRAINT IF EXISTS fk_org_group_members_group_org;

ALTER TABLE organization_group_members
    DROP COLUMN IF EXISTS organization_id;

ALTER TABLE organization_members
    DROP CONSTRAINT IF EXISTS uq_organization_members_account_org;

ALTER TABLE organization_groups
    DROP CONSTRAINT IF EXISTS uq_organization_groups_id_org;
