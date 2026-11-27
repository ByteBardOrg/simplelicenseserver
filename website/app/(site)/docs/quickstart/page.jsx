import CodeExample from "../../../../components/CodeExample";

export const metadata = {
  title: "Quickstart",
  description:
    "Quickstart guide for running Simple License Server from Docker image with a full Docker Compose configuration.",
  alternates: {
    canonical: "/docs/quickstart/"
  }
};

const pullImageCommand = "docker pull bytebardorg/simplelicenseserver";

// Secrets are generated locally into .env. Never reuse values from docs: the server refuses the
// example keys published on this site.
const generateSecretsCommand = `cat > .env <<EOF
POSTGRES_PASSWORD=$(openssl rand -hex 24)
MANAGEMENT_API_KEYS=$(openssl rand -hex 32)
OFFLINE_SIGNING_ENCRYPTION_KEY=$(openssl rand -hex 32)
EOF
chmod 600 .env`;

const composeExample = `services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:?create .env first (step 2)}
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
    # Bound to localhost. To serve it publicly, put a TLS reverse proxy (Caddy, Traefik, nginx,
    # Cloudflare Tunnel) in front instead of opening this port.
    ports:
      - "127.0.0.1:8080:8080"
    environment:
      # REQUIRED: database connection used by the API service.
      DATABASE_URL: postgres://postgres:\${POSTGRES_PASSWORD}@postgres:5432/simple_license_server?sslmode=disable

      # REQUIRED: management (admin) key from .env. You can also use MANAGEMENT_API_KEY.
      MANAGEMENT_API_KEYS: \${MANAGEMENT_API_KEYS:?create .env first (step 2)}

      # REQUIRED for offline tokens: encrypts signing keys at rest. Keep it stable and secret.
      OFFLINE_SIGNING_ENCRYPTION_KEY: \${OFFLINE_SIGNING_ENCRYPTION_KEY:?create .env first (step 2)}

      # OPTIONAL: serve the management UI at /ui/.
      UI_ENABLED: "true"

      # OPTIONAL: API listen port (default: 8080).
      PORT: "8080"

      # OPTIONAL: request and server shutdown controls.
      REQUEST_TIMEOUT: 15s
      SHUTDOWN_TIMEOUT: 10s
      HTTP_READ_TIMEOUT: 15s
      HTTP_WRITE_TIMEOUT: 30s
      HTTP_IDLE_TIMEOUT: 60s

      # OPTIONAL: per-IP rate limiting.
      RATE_LIMIT_ENABLED: "true"
      RATE_LIMIT_PER_IP_RPS: "20"
      RATE_LIMIT_PER_IP_BURST: "40"
      RATE_LIMIT_IP_TTL: 10m
      RATE_LIMIT_MAX_IP_ENTRIES: "10000"

      # OPTIONAL: behind a proxy, the header carrying the real client IP
      # (for example CF-Connecting-IP behind Cloudflare). Leave empty otherwise.
      CLIENT_IP_HEADER: ""

volumes:
  postgres_data:`;

const startComposeCommand = "docker compose up -d";

const verifyHealthCommand = "curl -sS http://localhost:8080/healthz";

const createServerKeyCommand = `MANAGEMENT_KEY=$(grep '^MANAGEMENT_API_KEYS=' .env | cut -d= -f2)

curl -sS http://localhost:8080/management/api-keys \\
  -H "Authorization: Bearer $MANAGEMENT_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"name":"quickstart"}'`;

const generateKeyCommand = `curl -sS http://localhost:8080/api/v1/generate \\
  -H "Authorization: Bearer <generated_server_api_key>" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: evt_quickstart_001" \\
  -d '{"slug":"default","metadata":{"email":"user@example.com","source":"quickstart"}}'`;

export default function QuickstartPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Quickstart</p>
      <h1>Run from Docker image in a few minutes</h1>

      <div className="card" style={{ marginTop: "1rem" }} id="pull-image">
        <h3>1) Pull the image</h3>
        <CodeExample language="bash" code={pullImageCommand} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="generate-secrets">
        <h3>2) Generate your secrets</h3>
        <p>
          This writes a <code>.env</code> file next to your compose file with random secrets. Docker Compose reads it
          automatically. Keep it private and back it up: losing <code>OFFLINE_SIGNING_ENCRYPTION_KEY</code> makes your
          offline signing keys unusable.
        </p>
        <CodeExample language="bash" code={generateSecretsCommand} />
        <p>
          Don&apos;t copy keys from any documentation, including this site: the server refuses to start with example
          values published here.
        </p>
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="compose-example">
        <h3>3) Create docker-compose.yml (full settings example)</h3>
        <p>
          Save this as <code>docker-compose.yml</code>. The comments call out which API environment variables are
          required and which are optional.
        </p>
        <CodeExample language="yaml" code={composeExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="start-stack">
        <h3>4) Start the stack</h3>
        <CodeExample language="bash" code={startComposeCommand} />
        <p>
          The management UI is then at <code>http://localhost:8080/ui/</code>; sign in with the{" "}
          <code>MANAGEMENT_API_KEYS</code> value from <code>.env</code>.
        </p>
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="verify-health">
        <h3>5) Verify health</h3>
        <CodeExample language="bash" code={verifyHealthCommand} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="create-server-key">
        <h3>6) Create your first generated server API key</h3>
        <CodeExample language="bash" code={createServerKeyCommand} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="generate-first-key">
        <h3>7) Generate your first license key</h3>
        <CodeExample language="bash" code={generateKeyCommand} />
      </div>

    </section>
  );
}
