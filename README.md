# Simple License Server API

Current API version: `0.2.0`

This repository contains the Go API implementation for the license and management flows described in `simple-license-server.md`.

## Implemented endpoints

License API:

- `GET /healthz`
- `POST /generate` (generated server API key required)
- `POST /revoke` (generated server API key required)
- `POST /activate`
- `POST /validate`
- `POST /deactivate`

Management API:

- `GET /management/slugs`
- `POST /management/slugs`
- `GET /management/slugs/{name}`
- `PATCH /management/slugs/{name}`
- `DELETE /management/slugs/{name}`
- `GET /management/api-keys`
- `POST /management/api-keys`
- `POST /management/api-keys/{id}/revoke`
- `GET /management/webhooks`
- `POST /management/webhooks`
- `PATCH /management/webhooks/{id}`
- `DELETE /management/webhooks/{id}`
- `GET /management/offline/signing-keys`
- `POST /management/offline/signing-keys`
- `POST /management/offline/signing-keys/{id}/activate`
- `POST /management/offline/signing-keys/{id}/retire`
- `GET /management/offline/public-keys`

`POST /generate` supports idempotency via the `Idempotency-Key` request header.

## Local development

1. Start Postgres.
2. Copy `.env.example` values into your environment.
3. Run:

```bash
go run ./cmd/server
```

The server auto-runs schema setup on startup and ensures a default slug named `default` exists.

## Docker compose

```bash
docker compose up --build
```

This starts:

- Postgres on `localhost:5432`
- API on `localhost:8080`

## Quick smoke test

1) Create a generated server key with the management key:

```bash
curl -sS http://localhost:8080/management/api-keys \
  -H "Authorization: Bearer management_key_dev_123456" \
  -H "Content-Type: application/json" \
  -d '{"name":"local-dev"}'
```

2) Use returned `api_key` against `/generate`:

```bash
curl -sS http://localhost:8080/generate \
  -H "Authorization: Bearer <generated_server_api_key>" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: evt_123" \
  -d '{"slug":"default","metadata":{"email":"user@example.com"}}'
```

## Authentication

- Management API routes require env bootstrap keys from `MANAGEMENT_API_KEYS` or `MANAGEMENT_API_KEY`.
- Provisioning routes (`/generate`, `/revoke`) require generated active server API keys from the management API.
- Runtime routes (`/activate`, `/validate`, `/deactivate`) use `license_key` + `fingerprint` and do not use API keys.

Both key-protected surfaces accept either:

- `Authorization: Bearer <key>`
- `X-API-Key: <key>`

## Operational defaults

Timeout controls:

- `REQUEST_TIMEOUT` (default `15s`)
- `SHUTDOWN_TIMEOUT` (default `10s`)
- `HTTP_READ_TIMEOUT` (default `15s`)
- `HTTP_WRITE_TIMEOUT` (default `30s`)
- `HTTP_IDLE_TIMEOUT` (default `60s`)

Rate limiting defaults:

- `RATE_LIMIT_ENABLED` (default `true`)
- `RATE_LIMIT_GLOBAL_RPS` (default `100`)
- `RATE_LIMIT_GLOBAL_BURST` (default `200`)
- `RATE_LIMIT_PER_IP_RPS` (default `20`)
- `RATE_LIMIT_PER_IP_BURST` (default `40`)
- `RATE_LIMIT_IP_TTL` (default `10m`)
- `RATE_LIMIT_MAX_IP_ENTRIES` (default `10000`)
- `TRUST_PROXY_HEADERS` (default `false`)

Offline token defaults:

- Offline JWT issuance is controlled per slug with `offline_enabled` and `offline_token_lifetime_hours`.
- All slugs, including the seeded `default` slug, default to offline disabled.
- `POST /activate` and valid `POST /validate` responses include `token` only when the license slug has offline enabled and an active signing key exists.
- `OFFLINE_SIGNING_ENCRYPTION_KEY` encrypts signing private keys at rest and must be at least 32 characters before creating or using signing keys.
- `OFFLINE_TOKEN_ISSUER` defaults to `simple-license-server`.
- `OFFLINE_TOKEN_AUDIENCE` is optional.
- Slug `offline_token_lifetime_hours` defaults to `24`.

Additional API security behavior:

- Requires `Content-Type: application/json` for JSON POST/PATCH endpoints.
- Applies defensive response headers (`nosniff`, CSP, no-store cache policy).
- Enforces request field and metadata size limits.
- Stores generated API keys hashed at rest.
