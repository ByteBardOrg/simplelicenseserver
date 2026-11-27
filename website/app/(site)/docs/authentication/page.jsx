import CodeExample from "../../../../components/CodeExample";

export const metadata = {
  title: "Authentication",
  description:
    "Authentication model for Simple License Server, including management bootstrap keys and generated server API keys.",
  alternates: {
    canonical: "/docs/authentication/"
  }
};

const managementAuthExample = `curl -sS http://localhost:8080/management/api-keys \\
  -H "Authorization: Bearer <management_key>"`;

const serverAuthExample = `curl -sS http://localhost:8080/api/v1/generate \\
  -H "Authorization: Bearer <generated_server_api_key>" \\
  -H "Content-Type: application/json" \\
  -d '{"slug":"default","metadata":{"email":"user@example.com"}}'`;

export default function AuthenticationPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Authentication</p>
      <h1>Simple auth model, two key types</h1>
      <p>
        The API uses Bearer tokens in the <code>Authorization</code> header. Use management bootstrap keys for
        <code> /management/*</code> routes, and generated server API keys for protected license lifecycle routes.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>1) Management bootstrap key</h3>
        <p>
          Configure this with <code>MANAGEMENT_API_KEYS</code> (or <code>MANAGEMENT_API_KEY</code>) and keep it in
          trusted backend or operator tooling.
        </p>
        <CodeExample language="bash" code={managementAuthExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>2) Generated server API key</h3>
        <p>
          Create these from the Management API, then use them for <code>/api/v1/generate</code> and <code>/api/v1/revoke</code>.
        </p>
        <CodeExample language="bash" code={serverAuthExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Public runtime routes</h3>
        <p>
          <code>/api/v1/activate</code>, <code>/api/v1/validate</code>, and <code>/api/v1/deactivate</code> do not require API keys, but you
          should still call them from your app backend when possible.
        </p>
      </div>
    </section>
  );
}
