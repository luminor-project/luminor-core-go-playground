CREATE TABLE case_dashboard (
    work_item_id    TEXT        PRIMARY KEY,
    status          TEXT        NOT NULL DEFAULT 'new',
    party_name      TEXT        NOT NULL DEFAULT '',
    party_actor_kind TEXT       NOT NULL DEFAULT '',
    subject_name    TEXT        NOT NULL DEFAULT '',
    subject_detail  TEXT        NOT NULL DEFAULT '',
    timeline_json   JSONB       NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
