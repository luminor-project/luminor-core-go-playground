DROP TABLE IF EXISTS account_party_pending_links;
ALTER TABLE account_cores DROP COLUMN IF EXISTS currently_active_party_id;
DROP TABLE IF EXISTS account_party_memberships;
