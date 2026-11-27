import CodeExample from "../../../../components/CodeExample";

export const metadata = {
  title: "Configuration",
  description:
    "Environment variables for Simple License Server, including defaults, required settings, and behavior notes.",
  alternates: {
    canonical: "/docs/configuration/"
  }
};

const minimalEnvExample = `# Required
DATABASE_URL=postgres://postgres:postgres@postgres:5432/simple_license_server?sslmode=disable
MANAGEMENT_API_KEYS=<output of: openssl rand -hex 32>

# Optional
PORT=8080
UI_ENABLED=false
REQUEST_TIMEOUT=15s
OFFLINE_SIGNING_ENCRYPTION_KEY=<output of: openssl rand -hex 32>`;

const optionRows = [
  {
    name: "DATABASE_URL",
    required: "Yes",
    defaultValue: "None",
    description: "Postgres connection string used by the API at startup."
  },
  {
    name: "MANAGEMENT_API_KEYS",
    required: "Yes*",
    defaultValue: "None",
    description: "Comma-separated management bootstrap keys for /management/* endpoints (each key must be 16+ chars)."
  },
  {
    name: "MANAGEMENT_API_KEY",
    required: "Yes*",
    defaultValue: "None",
    description: "Single-key fallback used only when MANAGEMENT_API_KEYS is not set."
  },
  {
    name: "PORT",
    required: "No",
    defaultValue: "8080",
    description: "HTTP listen port for the API server."
  },
  {
    name: "UI_ENABLED",
    required: "No",
    defaultValue: "false",
    description: "Serve the bundled management console from /ui/ on the API server when true. The Docker image includes the UI assets."
  },
  {
    name: "REQUEST_TIMEOUT",
    required: "No",
    defaultValue: "15s",
    description: "Per-request timeout applied by middleware."
  },
  {
    name: "SHUTDOWN_TIMEOUT",
    required: "No",
    defaultValue: "10s",
    description: "Graceful shutdown timeout when the process receives a stop signal."
  },
  {
    name: "HTTP_READ_TIMEOUT",
    required: "No",
    defaultValue: "15s",
    description: "Maximum time for reading request headers/body."
  },
  {
    name: "HTTP_WRITE_TIMEOUT",
    required: "No",
    defaultValue: "30s",
    description: "Maximum time before timing out writes of the response."
  },
  {
    name: "HTTP_IDLE_TIMEOUT",
    required: "No",
    defaultValue: "60s",
    description: "Maximum keep-alive idle time for client connections."
  },
  {
    name: "RATE_LIMIT_ENABLED",
    required: "No",
    defaultValue: "true",
    description: "Enables per-IP rate limiting. Public runtime routes and admin/provisioning routes have separate budgets; the UI's static files are not limited."
  },
  {
    name: "RATE_LIMIT_GLOBAL_RPS",
    required: "No",
    defaultValue: "0",
    description: "Optional cap on all traffic combined. 0 disables it, which is recommended: a shared cap lets a few abusive clients lock everyone out."
  },
  {
    name: "RATE_LIMIT_GLOBAL_BURST",
    required: "No",
    defaultValue: "0",
    description: "Burst for the global cap. Required when RATE_LIMIT_GLOBAL_RPS is set."
  },
  {
    name: "RATE_LIMIT_PER_IP_RPS",
    required: "No",
    defaultValue: "20",
    description: "Per-client-IP requests-per-second allowance."
  },
  {
    name: "RATE_LIMIT_PER_IP_BURST",
    required: "No",
    defaultValue: "40",
    description: "Per-client-IP burst capacity."
  },
  {
    name: "RATE_LIMIT_IP_TTL",
    required: "No",
    defaultValue: "10m",
    description: "TTL for per-IP limiter entries before cleanup."
  },
  {
    name: "RATE_LIMIT_MAX_IP_ENTRIES",
    required: "No",
    defaultValue: "10000",
    description: "Maximum tracked clients. When full, the least recently seen client is dropped; new clients are never refused."
  },
  {
    name: "ALLOW_EXAMPLE_SECRETS",
    required: "No",
    defaultValue: "false",
    description: "The server refuses to start with management keys or signing secrets published in these docs and examples. Set true only for a local development stack that is not reachable from other machines."
  },
  {
    name: "CLIENT_IP_HEADER",
    required: "No",
    defaultValue: "(empty)",
    description: "Header holding the real client IP when behind a proxy, such as CF-Connecting-IP or X-Real-IP. X-Forwarded-For uses its rightmost entry. Empty uses the TCP peer address. Replaces TRUST_PROXY_HEADERS."
  },
  {
    name: "AUTH_FAILURE_MAX_ATTEMPTS",
    required: "No",
    defaultValue: "10",
    description: "Wrong management or server API keys allowed per IP within AUTH_FAILURE_WINDOW before that IP gets HTTP 429 on key-protected routes. 0 disables the lockout."
  },
  {
    name: "AUTH_FAILURE_WINDOW",
    required: "No",
    defaultValue: "15m",
    description: "Length of the failed-authentication window and lockout."
  },
  {
    name: "OFFLINE_SIGNING_ENCRYPTION_KEY",
    required: "No*",
    defaultValue: "None",
    description: "Stable 32+ character secret used to encrypt offline signing private keys at rest. Required before creating or using signing keys."
  },
  {
    name: "OFFLINE_TOKEN_ISSUER",
    required: "No",
    defaultValue: "simple-license-server",
    description: "Issuer claim written to offline JWTs. Clients should verify this value during offline validation."
  },
  {
    name: "OFFLINE_TOKEN_AUDIENCE",
    required: "No",
    defaultValue: "None",
    description: "Fallback aud claim for offline JWTs from slugs that have no product set. A slug's product always takes precedence."
  },
];

const proxyExample = `# Cloudflare Tunnel -> Traefik -> Simple License Server
services:
  api:
    environment:
      CLIENT_IP_HEADER: CF-Connecting-IP
    # No "ports:" mapping: only Traefik on the internal network can reach the server.
    labels:
      - traefik.enable=true
      - traefik.http.routers.licenses.rule=Host(\`licenses.example.com\`)
      - traefik.http.services.licenses.loadbalancer.server.port=8080`;

function OptionsTable({ rows }) {
  return (
    <div className="docs-table-wrap">
      <table className="docs-table">
        <thead>
          <tr>
            <th scope="col">Variable</th>
            <th scope="col">Required</th>
            <th scope="col">Default</th>
            <th scope="col">What it does</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.name}>
              <td>
                <code>{row.name}</code>
              </td>
              <td>{row.required}</td>
              <td>
                <code>{row.defaultValue}</code>
              </td>
              <td>{row.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function ConfigurationPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Configuration</p>
      <h1>License server environment options</h1>
      <p>
        This page lists every supported environment variable, what it controls, and its default value.
        <code> DATABASE_URL</code> and one of <code>MANAGEMENT_API_KEYS</code> or <code>MANAGEMENT_API_KEY</code> are
        required to start the server.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Minimal startup example</h3>
        <CodeExample language="bash" code={minimalEnvExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Environment variables</h3>
        <OptionsTable rows={optionRows} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Running behind a proxy</h3>
        <p>
          Rate limits and the failed sign-in lockout are per client IP. Behind a proxy every request arrives from the
          proxy&apos;s address, so tell the server which header carries the real client IP:
        </p>
        <ul>
          <li>
            <strong>Cloudflare</strong> (including Cloudflare Tunnel, with or without Traefik or nginx in between):{" "}
            <code>CLIENT_IP_HEADER=CF-Connecting-IP</code>. Cloudflare sets this header itself, so clients cannot fake
            it through Cloudflare.
          </li>
          <li>
            <strong>A single reverse proxy</strong> such as Traefik, Caddy or nginx:{" "}
            <code>CLIENT_IP_HEADER=X-Forwarded-For</code>, which uses the rightmost entry (the one your proxy added), or{" "}
            <code>X-Real-IP</code> if your proxy sets it.
          </li>
        </ul>
        <p>
          Only set this when the server can be reached <strong>solely</strong> through that proxy. If its port is also
          reachable directly, anyone can send the header and choose their own address. Each request&apos;s resolved{" "}
          <code>client_ip</code> appears in the server&apos;s request log, so you can confirm the setting works.
        </p>
        <CodeExample language="yaml" code={proxyExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Validation notes</h3>
        <ul>
          <li><code>*</code> One of <code>MANAGEMENT_API_KEYS</code> or <code>MANAGEMENT_API_KEY</code> must be set.</li>
          <li>All API keys must be at least 16 characters. Generate them with <code>openssl rand -hex 32</code>.</li>
          <li>
            Example secrets from these docs (such as <code>management_key_dev_123456</code>) are refused at startup
            unless <code>ALLOW_EXAMPLE_SECRETS=true</code>.
          </li>
          <li>Set <code>UI_ENABLED=true</code> to serve the bundled console at <code>/ui/</code>; <code>/ui</code> redirects there.</li>
          <li>Use <code>POST /management/api-keys</code> to create generated server API keys for <code>/api/v1/generate</code> and <code>/api/v1/revoke</code>.</li>
          <li>All timeout values must be valid Go durations (for example <code>15s</code>, <code>10m</code>).</li>
          <li>When <code>RATE_LIMIT_ENABLED=true</code>, the per-IP values must be greater than zero; the global cap may be <code>0</code> (off).</li>
          <li>
            <code>TRUST_PROXY_HEADERS</code> was removed in 0.5.0. Setting it to <code>true</code> stops startup with a
            message pointing to <code>CLIENT_IP_HEADER</code>.
          </li>
          <li>Booleans accept values like <code>true/false</code>, <code>1/0</code>, <code>yes/no</code>.</li>
          <li><code>*</code> <code>OFFLINE_SIGNING_ENCRYPTION_KEY</code> is optional for online-only licensing, but required before creating offline signing keys or issuing offline tokens.</li>
          <li>Offline token issuance is controlled per slug with <code>offline_enabled</code> and <code>offline_token_lifetime_hours</code>; all slugs default to offline disabled with a 24-hour token lifetime.</li>
        </ul>
      </div>
    </section>
  );
}
