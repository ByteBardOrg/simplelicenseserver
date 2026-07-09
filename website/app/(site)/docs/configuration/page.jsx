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
MANAGEMENT_API_KEYS=management_key_prod_please_make_this_long_and_random

# Optional
PORT=8080
UI_ENABLED=false
REQUEST_TIMEOUT=15s
OFFLINE_SIGNING_ENCRYPTION_KEY=replace_with_32_plus_chars_before_creating_signing_keys`;

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
    description: "Enables global + per-IP rate limiting middleware."
  },
  {
    name: "RATE_LIMIT_GLOBAL_RPS",
    required: "No",
    defaultValue: "100",
    description: "Global requests-per-second allowance when rate limiting is enabled."
  },
  {
    name: "RATE_LIMIT_GLOBAL_BURST",
    required: "No",
    defaultValue: "200",
    description: "Global burst capacity when rate limiting is enabled."
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
    description: "Maximum number of tracked IP limiter entries."
  },
  {
    name: "TRUST_PROXY_HEADERS",
    required: "No",
    defaultValue: "false",
    description: "Trust X-Forwarded-For/X-Real-IP when running behind a trusted proxy."
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
    description: "Optional audience claim written to offline JWTs. Leave empty when clients do not validate an audience."
  },
];

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
        <h3>Validation notes</h3>
        <ul>
          <li><code>*</code> One of <code>MANAGEMENT_API_KEYS</code> or <code>MANAGEMENT_API_KEY</code> must be set.</li>
          <li>All API keys must be at least 16 characters.</li>
          <li>Set <code>UI_ENABLED=true</code> to serve the bundled console at <code>/ui/</code>; <code>/ui</code> redirects there.</li>
          <li>Use <code>POST /management/api-keys</code> to create generated server API keys for <code>/generate</code> and <code>/revoke</code>.</li>
          <li>All timeout values must be valid Go durations (for example <code>15s</code>, <code>10m</code>).</li>
          <li>When <code>RATE_LIMIT_ENABLED=true</code>, all rate-limit numeric values must be greater than zero.</li>
          <li>Booleans accept values like <code>true/false</code>, <code>1/0</code>, <code>yes/no</code>.</li>
          <li><code>*</code> <code>OFFLINE_SIGNING_ENCRYPTION_KEY</code> is optional for online-only licensing, but required before creating offline signing keys or issuing offline tokens.</li>
          <li>Offline token issuance is controlled per slug with <code>offline_enabled</code> and <code>offline_token_lifetime_hours</code>; all slugs default to offline disabled with a 24-hour token lifetime.</li>
        </ul>
      </div>
    </section>
  );
}
