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

ALTER TABLE idempotency_records
    ALTER COLUMN response_body DROP NOT NULL;

INSERT INTO slugs (name, max_activations, expiration_type, is_default)
VALUES ('default', 1, 'forever', TRUE)
ON CONFLICT (name) DO NOTHING;
