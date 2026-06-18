-- +goose Up

-- Provider-neutral rename: an organization's identity lives in Logto (or any
-- future IdP), but core is the source of truth. external_id is the link, filled
-- only after the outbox worker provisions the org in Logto — so it is nullable.
-- "synced" is derived from external_id IS NOT NULL; there is no status column.
ALTER TABLE organizations RENAME COLUMN logto_org_id TO external_id;
ALTER TABLE organizations ALTER COLUMN external_id DROP NOT NULL;

-- Transactional outbox: a create writes the organization row and one event row
-- in the same DB transaction (core stays consistent atomically). A background
-- worker delivers pending events to Logto with retries — no periodic reconcile.
CREATE TABLE outbox_events (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate       TEXT        NOT NULL,
    aggregate_id    UUID        NOT NULL,
    event_type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending',
    attempts        INT         NOT NULL DEFAULT 0,
    last_error      TEXT        NOT NULL DEFAULT '',
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The worker only ever scans due, undelivered events; index exactly that path.
CREATE INDEX idx_outbox_due ON outbox_events (next_attempt_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE outbox_events;
ALTER TABLE organizations RENAME COLUMN external_id TO logto_org_id;
-- Existing NULLs must be resolved before this column can be NOT NULL again.
ALTER TABLE organizations ALTER COLUMN logto_org_id SET NOT NULL;
