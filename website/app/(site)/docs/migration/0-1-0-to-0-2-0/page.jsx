import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "0.1.0 -> 0.2.0 Migration",
  description: "How to migrate from Simple License Server 0.1.0 to 0.2.0.",
  alternates: {
    canonical: "/docs/migration/0-1-0-to-0-2-0/"
  }
};

const envDiff = `# Before (0.1.0)
SERVER_API_KEYS=server_key_dev_123456

# After (0.2.0)
MANAGEMENT_API_KEYS=management_key_dev_123456`;

const createServerKeyExample = `curl -sS http://localhost:8080/management/api-keys \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{"name":"stripe-prod"}'`;

export default function Migration010To020Page() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Migration</p>
      <h1>0.1.0 -&gt; 0.2.0</h1>
      <p>
        Version 0.2.0 introduces a strict split between License API and Management API and changes how key
        authentication works.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What changed</h3>
        <ul>
          <li>New Management API routes under <code>/management/*</code> for API key and webhook configuration.</li>
          <li>
            <code>/generate</code> and <code>/revoke</code> no longer accept env bootstrap keys.
          </li>
          <li>
            Env auth config renamed from <code>SERVER_API_KEYS</code> / <code>SERVER_API_KEY</code> to
            <code> MANAGEMENT_API_KEYS</code> / <code>MANAGEMENT_API_KEY</code>.
          </li>
          <li>Health endpoint now returns an explicit API version field.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>1) Update environment variables</h3>
        <CodeExample language="bash" code={envDiff} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>2) Create generated server API keys</h3>
        <p>
          Create at least one server key through the Management API, then use that generated key for
          <code> /generate</code> and <code>/revoke</code>.
        </p>
        <CodeExample language="bash" code={createServerKeyExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>3) Update integrations and dashboards</h3>
        <ul>
          <li>Payment/webhook bridges should use generated server keys.</li>
          <li>Internal operator UI should call <code>/management/*</code> with management auth.</li>
          <li>Review docs links: License API is now separate from Management API pages.</li>
        </ul>
      </div>
    </section>
  );
}
