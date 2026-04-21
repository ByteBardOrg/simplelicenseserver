# simplelicenseserver
https://simplelicenseserver.com

A simple, postgres-backed license server.

## Endpoints
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


## Example Compose
```yaml
services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: simple_license_server
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d simple_license_server"]
      interval: 5s
      timeout: 5s
      retries: 10
    volumes:
      - postgres_data:/var/lib/postgresql/data
    # Optional: expose Postgres to your host for debugging.
    # ports:
    #   - "5432:5432"

  api:
    image: bytebardorg/simplelicenseserver
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8080:8080"
    environment:
      # REQUIRED: database connection used by the API service.
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/simple_license_server?sslmode=disable

      # REQUIRED: management bootstrap key (16+ chars). You can also use MANAGEMENT_API_KEY.
      MANAGEMENT_API_KEYS: management_key_dev_123456

      # OPTIONAL: API listen port (default: 8080).
      PORT: "8080"

      # OPTIONAL: request and server shutdown controls.
      REQUEST_TIMEOUT: 15s
      SHUTDOWN_TIMEOUT: 10s
      HTTP_READ_TIMEOUT: 15s
      HTTP_WRITE_TIMEOUT: 30s
      HTTP_IDLE_TIMEOUT: 60s

      # OPTIONAL: global and per-IP rate limiting.
      RATE_LIMIT_ENABLED: "true"
      RATE_LIMIT_GLOBAL_RPS: "100"
      RATE_LIMIT_GLOBAL_BURST: "200"
      RATE_LIMIT_PER_IP_RPS: "20"
      RATE_LIMIT_PER_IP_BURST: "40"
      RATE_LIMIT_IP_TTL: 10m
      RATE_LIMIT_MAX_IP_ENTRIES: "10000"

      # OPTIONAL: set true only when a trusted proxy sets forwarding headers.
      TRUST_PROXY_HEADERS: "false"

volumes:
  postgres_data:
```

## What it is not
Simple License Server is not a product management solution.

It is also not a checkout or billing orchestration system.
It is a focused license server that can be safely exposed to the internet and used for:

- issuing licenses
- activating licenses
- deactivating seats
- validating licenses
- revoking licenses
- managing simple policy through slugs
