# Simple License Server API

Current API version: `0.5.0`

This repository contains the Go API implementation for the license and management flows described in `simple-license-server.md`.

## Implemented endpoints

License API:

- `GET /healthz`
- `POST /api/v1/generate` (generated server API key required)
- `POST /api/v1/revoke` (generated server API key required)
- `POST /api/v1/activate`
- `POST /api/v1/validate`
- `POST /api/v1/deactivate`

Management API:

- `GET /management/licenses`
- `GET /management/licenses/{key}`
- `PATCH /management/licenses/{key}`
- `DELETE /management/licenses/{key}/activations/{id}`
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
- `GET /management/webhooks/deliveries`
- `GET /management/offline/signing-keys`
- `POST /management/offline/signing-keys`
- `POST /management/offline/signing-keys/{id}/activate`
- `POST /management/offline/signing-keys/{id}/retire`
- `GET /management/offline/public-keys`

`POST /api/v1/generate` supports idempotency via the `Idempotency-Key` request header.

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

2) Use returned `api_key` against `/api/v1/generate`:

```bash
curl -sS http://localhost:8080/api/v1/generate \
  -H "Authorization: Bearer <generated_server_api_key>" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: evt_123" \
  -d '{"slug":"default","metadata":{"email":"user@example.com"}}'
```

## Authentication

- Management API routes require env bootstrap keys from `MANAGEMENT_API_KEYS` or `MANAGEMENT_API_KEY`.
- Provisioning routes (`/api/v1/generate`, `/api/v1/revoke`) require generated active server API keys from the management API.
- Runtime routes (`/api/v1/activate`, `/api/v1/validate`, `/api/v1/deactivate`) use `license_key` + `fingerprint` and do not use API keys. They share one response shape (`valid`, `reason`, `license`, optional `token`) and return HTTP 200 for every decided outcome, including unknown keys (`reason: "not_found"`).

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
- `RATE_LIMIT_GLOBAL_RPS` (default `0`, off; a cap shared by all clients lets a few abusers lock everyone out)
- `RATE_LIMIT_GLOBAL_BURST` (required when the global limit is on)
- `RATE_LIMIT_PER_IP_RPS` (default `20`)
- `RATE_LIMIT_PER_IP_BURST` (default `40`)
- `RATE_LIMIT_IP_TTL` (default `10m`)
- `RATE_LIMIT_MAX_IP_ENTRIES` (default `10000`)
- Per-IP budgets are kept separately for the public runtime routes (`/api/v1/activate|validate|deactivate`) and everything else. The bundled UI's static files are not rate limited.
- `CLIENT_IP_HEADER` (default empty: use the TCP peer address). Behind a proxy, name the header that carries the real client IP, for example `CF-Connecting-IP` behind Cloudflare. `X-Forwarded-For` uses the rightmost entry. Only set this when the server is reachable solely through that proxy.
- `AUTH_FAILURE_MAX_ATTEMPTS` (default `10`, `0` disables) and `AUTH_FAILURE_WINDOW` (default `15m`): wrong management or server API keys from one IP within the window lock that IP out of key-protected routes (HTTP 429) until the window ends.
- `TRUST_PROXY_HEADERS` was removed in 0.5.0; setting it to `true` stops startup with a pointer to `CLIENT_IP_HEADER`.

Offline token defaults:

- Offline JWT issuance is controlled per slug with `offline_enabled` and `offline_token_lifetime_hours`.
- Slugs also support `attributes` (JSON object) for license information such as feature flags or plan metadata.
- Slugs have an optional `product` naming the app their licenses are for. Runtime responses return it as `license.product`, and offline JWTs carry it as `aud` (falling back to `OFFLINE_TOKEN_AUDIENCE`). Apps should check both, plus the token `fingerprint`.
- Slug `attributes` are snapshotted onto each generated license and returned by `/api/v1/generate`, the runtime endpoints (`license.attributes`), and the management license endpoints.
- Offline JWTs include the snapshotted license `attributes` as a nested `attributes` claim.
- All slugs, including the seeded `default` slug, default to offline disabled.
- `POST /api/v1/activate` and valid `POST /api/v1/validate` responses include `token` only when the license slug has offline enabled and an active signing key exists.
- `OFFLINE_SIGNING_ENCRYPTION_KEY` encrypts signing private keys at rest and must be at least 32 characters before creating or using signing keys.
- `OFFLINE_TOKEN_ISSUER` defaults to `simple-license-server`.
- `OFFLINE_TOKEN_AUDIENCE` is optional.
- Slug `offline_token_lifetime_hours` defaults to `24`.

Additional API security behavior:

- Requires `Content-Type: application/json` for JSON POST/PATCH endpoints.
- Applies defensive response headers (`nosniff`, CSP, no-store cache policy).
- Enforces request field and metadata size limits.
- Stores generated API keys hashed at rest.
