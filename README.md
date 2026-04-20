# Simple License Server API (v1 bootstrap)

This repository now includes a Go API implementation for the v1 licensing flows described in `simple-license-server.md` and `PAYMENT_INTEGRATIONS.md`.

## Implemented endpoints

- `GET /healthz`
- `POST /generate` (server API key required)
- `POST /revoke` (server API key required)
- `POST /activate`
- `POST /validate`
- `POST /deactivate`

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

Generate a key against the default slug:

```bash
curl -sS http://localhost:8080/generate \
  -H "Authorization: Bearer server_key_dev_123" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: evt_123" \
  -d '{"slug":"default","metadata":{"email":"user@example.com"}}'
```

## Authentication

Provisioning routes (`/generate`, `/revoke`) accept either:

- `Authorization: Bearer <server-key>`
- `X-API-Key: <server-key>`

Use `SERVER_API_KEYS` (comma-separated) or `SERVER_API_KEY` for a single key.

## Operational defaults

The API now includes explicit timeout controls for production hardening:

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

Additional API security behavior:

- Requires `Content-Type: application/json` for JSON POST endpoints.
- Applies defensive response headers (`nosniff`, CSP, no-store cache policy).
- Enforces request field and metadata size limits.
- Uses constant-time API key hash comparison in auth checks.
