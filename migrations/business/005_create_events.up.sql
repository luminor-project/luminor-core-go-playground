CREATE TABLE events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id       TEXT        NOT NULL,
    stream_version  INT         NOT NULL,
    event_type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    causation_id    TEXT        NOT NULL DEFAULT '',
    correlation_id  TEXT        NOT NULL DEFAULT '',
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_events_stream_id ON events (stream_id, stream_version);
