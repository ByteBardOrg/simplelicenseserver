export const metadata = {
  title: "0.2.0 -> 0.3.0 Migration",
  description: "How to upgrade from Simple License Server 0.2.0 to 0.3.0.",
  alternates: {
    canonical: "/docs/migration/0-2-0-to-0-3-0/"
  }
};

export default function Migration020To030Page() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Migration</p>
      <h1>0.2.0 -&gt; 0.3.0</h1>
      <p>
        Version 0.3.0 is an additive release. Existing API integrations can continue running without a manual database
        migration. Offline license support is opt-in per slug and requires signing-key configuration before tokens are issued.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What changed</h3>
        <ul>
          <li>Built-in management UI can be served from the Docker image with <code>UI_ENABLED=true</code>.</li>
          <li>New license registry endpoint: <code>GET /management/licenses</code>.</li>
          <li>New webhook run log endpoint: <code>GET /management/webhooks/deliveries</code>.</li>
          <li>Offline JWT license support is now available per slug.</li>
          <li>New offline signing key endpoints under <code>/management/offline/*</code>.</li>
          <li><code>POST /validate</code> can return a refreshed offline token when the license&apos;s slug has offline enabled.</li>
          <li>The Webhooks UI now shows recent delivery attempts, statuses, response codes, and errors.</li>
          <li>The management UI includes license listing, generation, revocation, slug management, API keys, webhooks, and offline licenses.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Required actions</h3>
        <ul>
          <li>Pull or rebuild the 0.3.0 Docker image.</li>
          <li>No manual database migration is required for existing 0.2.0 installations; startup schema migration adds the offline columns and signing key table.</li>
          <li>No environment variable changes are required unless you want to enable the bundled UI or use offline signing.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Optional: enable the UI</h3>
        <p>
          Set <code>UI_ENABLED=true</code> to serve the bundled management console from the same Docker image as the API.
          The console uses your management API key for operator actions.
        </p>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Optional: enable offline licenses</h3>
        <p>
          Offline tokens are controlled per slug with <code>offline_enabled</code> and
          <code> offline_token_lifetime_hours</code>. All slugs, including the seeded <code>default</code> slug,
          default to offline-disabled unless you turn the setting on.
        </p>
        <ul>
          <li>Set <code>OFFLINE_SIGNING_ENCRYPTION_KEY</code> to a stable secret of at least 32 characters before creating signing keys.</li>
          <li>Optionally set <code>OFFLINE_TOKEN_ISSUER</code> and <code>OFFLINE_TOKEN_AUDIENCE</code>.</li>
          <li>Configure token lifetime per slug with <code>offline_token_lifetime_hours</code>; it defaults to <code>24</code>.</li>
          <li>Create and activate an Ed25519 signing key from the Offline Licenses UI or management API.</li>
          <li>Export active and verify-only public keys and ship them with clients that validate licenses offline.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Integration notes</h3>
        <ul>
          <li>Continue using generated server API keys for <code>/generate</code> and <code>/revoke</code>.</li>
          <li>Use <code>GET /management/licenses</code> for admin license registry screens and reporting.</li>
          <li>Use <code>GET /management/webhooks/deliveries</code> for webhook delivery debugging and support workflows.</li>
          <li>Treat the new <code>token</code> response field on <code>/activate</code> and <code>/validate</code> as optional; it is omitted when the slug is not offline-enabled or no active signing key exists.</li>
        </ul>
      </div>
    </section>
  );
}
