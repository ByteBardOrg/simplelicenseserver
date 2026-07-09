import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "Management UI Concepts",
  description:
    "How to enable and access the bundled Simple License Server management UI at /ui/, including authentication and operator scope.",
  alternates: {
    canonical: "/docs/concepts/management-ui/"
  }
};

const enableUIExample = `UI_ENABLED=true`;

export default function ManagementUIConceptsPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Concepts</p>
      <h1>Management UI</h1>
      <p>
        Simple License Server includes a bundled operator console for common management tasks. When enabled, the API
        server serves it from <code>/ui/</code> on the same host and port as the API.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Enable and open the UI</h3>
        <p>
          Set <code>UI_ENABLED=true</code>, restart the server, then open <code>http://localhost:8080/ui/</code> for a
          local Docker or development server. The shorter <code>/ui</code> path redirects to <code>/ui/</code>.
        </p>
        <CodeExample language="bash" code={enableUIExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Authentication model</h3>
        <p>
          The UI performs operator actions against Management API routes. It uses your management bootstrap key, the
          same key accepted by <code>Authorization: Bearer &lt;key&gt;</code> or <code>X-API-Key: &lt;key&gt;</code> for
          <code> /management/*</code> endpoints.
        </p>
        <ul>
          <li>Management keys are configured with <code>MANAGEMENT_API_KEYS</code> or <code>MANAGEMENT_API_KEY</code>.</li>
          <li>Generated server API keys are still used for provisioning routes like <code>/generate</code> and <code>/revoke</code>.</li>
          <li>Do not share management keys with untrusted users or ship them in customer-facing apps.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What the UI manages</h3>
        <ul>
          <li>List, inspect, generate, activate, and revoke licenses.</li>
          <li>Create and edit slug policy templates.</li>
          <li>Create and revoke generated server API keys.</li>
          <li>Configure webhook endpoints and inspect recent delivery runs.</li>
          <li>Create, activate, retire, and export offline license signing keys.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Deployment notes</h3>
        <ul>
          <li>The Docker image includes the built UI assets.</li>
          <li>The UI is served by the API process, so it uses the same origin and connects back to the same API host.</li>
          <li>If you expose the API publicly, protect access to <code>/ui/</code> the same way you protect other operator tooling.</li>
        </ul>
      </div>

      <p style={{ marginTop: "1rem", display: "flex", gap: "0.6rem", flexWrap: "wrap" }}>
        <Link className="button button-secondary" href="/docs/concepts/">
          Back to concepts overview
        </Link>
        <Link className="button button-primary" href="/docs/configuration/">
          Open Configuration
        </Link>
      </p>
    </section>
  );
}
