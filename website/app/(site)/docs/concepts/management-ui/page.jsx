import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";
import Screenshot from "../../../../../components/Screenshot";

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

      <Screenshot
        src="/docs/screenshots/ui-licenses.webp"
        alt="The Licenses page: status filters with counts, a search box, and a table of license keys with status, slug, seats in use and creation date."
        width={1200}
        height={780}
        caption="Licenses: filter by status, search by key, slug or metadata, and click a row for details."
      />

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
          <li>Generated server API keys are still used for provisioning routes like <code>/api/v1/generate</code> and <code>/api/v1/revoke</code>.</li>
          <li>Do not share management keys with untrusted users or ship them in customer-facing apps.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What the UI manages</h3>
        <ul>
          <li>
            <strong>Licenses</strong>: search, filter by status, issue, edit seats and expiry, revoke and restore, list a
            license&apos;s devices, add or remove devices, and issue offline tokens for a device.
          </li>
          <li><strong>Slugs</strong>: create, edit, and archive slug policy templates.</li>
          <li>
            <strong>Settings</strong>: create and revoke generated server API keys, configure webhook endpoints and
            inspect recent deliveries, and create, activate, retire, and export offline signing keys.
          </li>
        </ul>
        <div className="docs-screenshot-pair">
          <Screenshot
            narrow
            src="/docs/screenshots/ui-license-details.webp"
            alt="License details panel showing seat and expiry limits with an Edit button, and a Devices list where each device has Token and Remove buttons."
            width={448}
            height={796}
            caption="License details: edit seats and expiry, see each device, free a seat, or revoke."
          />
          <Screenshot
            narrow
            src="/docs/screenshots/ui-new-license.webp"
            alt="New license panel with a slug selector, the slug's product, seats, expiry and offline settings shown as tags, and an optional metadata field."
            width={448}
            height={616}
            caption="New license: pick a slug, optionally add metadata, and copy the issued key."
          />
        </div>
        <p>
          Apps normally activate themselves with <code>POST /api/v1/activate</code>. The <strong>Add device</strong>{" "}
          action in a license&apos;s details does the same call for you, which is useful for on-prem installs that never
          reach the server: enter the customer&apos;s machine id, and if the slug has offline tokens enabled, copy or
          download the signed JWT and send it to them. Set the slug&apos;s offline token lifetime to match the license
          term (for example 8760 hours for a year); tokens never outlive the license&apos;s own expiry. Use{" "}
          <strong>Token</strong> on an existing device to issue a fresh JWT after a renewal.
        </p>
        <Screenshot
          narrow
          src="/docs/screenshots/ui-offline-token.webp"
          alt="A device was added to an on-prem license and the panel shows the signed offline token, valid for one year, with Download and Copy token buttons."
          width={448}
          height={876}
          caption="Add device on an on-prem license: the signed offline token is ready to download or copy."
        />
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
