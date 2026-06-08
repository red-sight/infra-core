-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE organizations (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    logto_org_id TEXT       NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE organizations;
