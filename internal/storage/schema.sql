CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS slugs (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    max_activations INTEGER NOT NULL CHECK (max_activations > 0),
    expiration_type TEXT NOT NULL CHECK (expiration_type IN ('forever', 'duration', 'fixed_date')),
    expiration_days INTEGER,
    fixed_expires_at TIMESTAMPTZ,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (expiration_type = 'forever' AND expiration_days IS NULL AND fixed_expires_at IS NULL)
        OR (expiration_type = 'duration' AND expiration_days IS NOT NULL AND expiration_days > 0 AND fixed_expires_at IS NULL)
        OR (expiration_type = 'fixed_date' AND expiration_days IS NULL AND fixed_expires_at IS NOT NULL)
    )
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_slugs_name_format'
          AND conrelid = 'slugs'::regclass
    ) THEN
        ALTER TABLE slugs
            ADD CONSTRAINT chk_slugs_name_format
            CHECK (name ~ '^[a-z0-9][a-z0-9-]*$');
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_slugs_default_true ON slugs (is_default) WHERE is_default;

CREATE TABLE IF NOT EXISTS licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL UNIQUE,
    slug_id BIGINT NOT NULL REFERENCES slugs(id),
    status TEXT NOT NULL CHECK (status IN ('inactive', 'active', 'revoked')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

ALTER TABLE licenses
    ADD COLUMN IF NOT EXISTS max_activations INTEGER;

UPDATE licenses l
SET max_activations = s.max_activations
FROM slugs s
WHERE l.slug_id = s.id
  AND l.max_activations IS NULL;

ALTER TABLE licenses
    ALTER COLUMN max_activations SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_licenses_max_activations_positive'
          AND conrelid = 'licenses'::regclass
    ) THEN
        ALTER TABLE licenses
            ADD CONSTRAINT chk_licenses_max_activations_positive
            CHECK (max_activations > 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_licenses_slug_id ON licenses (slug_id);

CREATE TABLE IF NOT EXISTS activations (
    id BIGSERIAL PRIMARY KEY,
    license_id UUID NOT NULL REFERENCES licenses(id),
    fingerprint TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_validated_at TIMESTAMPTZ,
    deactivated_at TIMESTAMPTZ,
    deactivation_reason TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_active_activation_by_fingerprint
ON activations (license_id, fingerprint)
WHERE deactivated_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_activations_license_id ON activations (license_id);

CREATE TABLE IF NOT EXISTS idempotency_records (
    endpoint TEXT NOT NULL,
    idem_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (endpoint, idem_key)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    key_type TEXT NOT NULL CHECK (key_type IN ('server')),
    key_hint TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'api_keys'
          AND column_name = 'key_prefix'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'api_keys'
          AND column_name = 'key_hint'
    ) THEN
        ALTER TABLE api_keys RENAME COLUMN key_prefix TO key_hint;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_api_keys_type_active
ON api_keys (key_type, created_at DESC)
WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    events JSONB NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (jsonb_typeof(events) = 'array')
);

CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_enabled
ON webhook_endpoints (enabled);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id BIGSERIAL PRIMARY KEY,
    endpoint_id BIGINT NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'sending', 'delivered', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    last_response_status INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_pending
ON webhook_deliveries (status, next_attempt_at, id);

ALTER TABLE idempotency_records
    ALTER COLUMN response_body DROP NOT NULL;

INSERT INTO slugs (name, max_activations, expiration_type, is_default)
VALUES ('default', 1, 'forever', TRUE)
ON CONFLICT (name) DO NOTHING;
