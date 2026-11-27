export const metadata = {
  title: "0.3.0 -> 0.4.0 Migration",
  description: "How to upgrade from Simple License Server 0.3.0 to 0.4.0.",
  alternates: {
    canonical: "/docs/migration/0-3-0-to-0-4-0/"
  }
};

export default function Migration030To040Page() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Migration</p>
      <h1>0.3.0 -&gt; 0.4.0</h1>
      <p>
        Version 0.4.0 is an additive release centered on slug attributes and license snapshots. Existing integrations
        continue to work, but responses and webhook/JWT payloads include new optional fields.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What changed</h3>
        <ul>
          <li>Slugs now support <code>attributes</code> (JSON object) for license information such as gated features.</li>
          <li>Generated licenses now store snapshotted slug <code>attributes</code>.</li>
          <li><code>POST /generate</code> responses now include <code>attributes</code>.</li>
          <li><code>GET /management/licenses</code> now returns license <code>attributes</code>.</li>
          <li><code>GET/POST/PATCH /management/slugs</code> now include slug <code>attributes</code>.</li>
          <li><code>license.generated</code> webhook payloads now include <code>attributes</code>.</li>
          <li>Offline JWT tokens now include an <code>attributes</code> object claim from the license snapshot.</li>
          <li>The management UI license details modal now supports runtime activation with fingerprint and optional activation metadata.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Required actions</h3>
        <ul>
          <li>Pull or rebuild the 0.4.0 Docker image.</li>
          <li>No manual database migration is required for existing 0.3.0 installations; startup schema migration adds slug/license attributes columns.</li>
          <li>No new environment variables are required.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Integration notes</h3>
        <ul>
          <li>If you consume <code>/generate</code> responses, handle the new optional <code>attributes</code> object.</li>
          <li>If you consume <code>/management/licenses</code> or <code>/management/slugs</code>, update response parsing for <code>attributes</code>.</li>
          <li>If you consume <code>license.generated</code> webhooks, update payload parsing for <code>attributes</code>.</li>
          <li>If clients use offline JWTs for feature gating, read feature data from the nested <code>attributes</code> claim.</li>
          <li><code>POST /activate</code> metadata is activation-seat scoped, while <code>POST /generate</code> metadata remains license scoped.</li>
          <li>The new UI activation action surfaces the full <code>/activate</code> response, including offline JWTs when the slug is offline-enabled.</li>
        </ul>
      </div>
    </section>
  );
}
