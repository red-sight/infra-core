-- +goose Up

-- slug is the DNS-safe label that addresses the organization's tenant frontend
-- (e.g. <slug>.app.localhost). Added nullable, backfilled for existing rows, then
-- made NOT NULL UNIQUE.
ALTER TABLE organizations ADD COLUMN slug TEXT;
UPDATE organizations SET slug = 'org-' || left(id::text, 8) WHERE slug IS NULL;
ALTER TABLE organizations ALTER COLUMN slug SET NOT NULL;
ALTER TABLE organizations ADD CONSTRAINT organizations_slug_key UNIQUE (slug);

-- +goose Down
ALTER TABLE organizations DROP CONSTRAINT organizations_slug_key;
ALTER TABLE organizations DROP COLUMN slug;
