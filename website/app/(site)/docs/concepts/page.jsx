import Link from "next/link";

export const metadata = {
  title: "Concepts",
  description:
    "Core concepts in Simple License Server: slugs, licenses, seats, expiration, and License API vs Management API flows.",
  alternates: {
    canonical: "/docs/concepts/"
  }
};


const glossaryRows = [
  {
    term: "Slug",
    meaning:
      "Policy template for newly generated licenses. Controls max activations and expiration policy for licenses created under it."
  },
  {
    term: "License",
    meaning:
      "Issued key tied to a slug. Stores status, metadata, created time, and resolved expires_at."
  },
  {
    term: "Seat",
    meaning:
      "One active fingerprint binding for a license. Seats are represented by active rows in activations."
  },
  {
    term: "Fingerprint",
    meaning:
      "Device/app-instance identifier sent by your client to activate, validate, and deactivate."
  },
  {
    term: "License API",
    meaning:
      "License lifecycle endpoints. /generate and /revoke require a generated server API key; runtime routes are called by your app."
  },
  {
    term: "Management API",
    meaning:
      "Operator/internal endpoints under /management/* that require a management bootstrap key."
  },
  {
    term: "Runtime API",
    meaning:
      "Client-facing endpoints (/activate, /validate, /deactivate) used by your shipped app."
  },
  {
    term: "Idempotency-Key",
    meaning:
      "Header used on /generate to make webhook retries safe and avoid duplicate license issuance."
  },
  {
    term: "Revocation",
    meaning:
      "License-level action that sets status to revoked and blocks future activate/validate calls."
  }
];

function GlossaryTable({ rows }) {
  return (
    <div className="docs-table-wrap">
      <table className="docs-table">
        <thead>
          <tr>
            <th scope="col">Concept</th>
            <th scope="col">Meaning</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.term}>
              <td>
                <strong>{row.term}</strong>
              </td>
              <td>{row.meaning}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function ConceptsPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Concepts</p>
      <h1>How the license model works</h1>
      <p>
        This is the mental model behind Simple License Server. If you understand these concepts, the endpoint behavior
        is straightforward.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>System flow in plain language</h3>
        <ol>
          <li>You define a slug (policy template) for a product or plan.</li>
          <li>Your trusted backend calls <code>POST /generate</code> with that slug.</li>
          <li>The server creates a license key with expiration and seat limits resolved at that moment.</li>
          <li>The client app activates the license with a fingerprint, consuming a seat.</li>
          <li>The client periodically validates the same license+fingerprint pair.</li>
          <li>Seats can be released with <code>POST /deactivate</code>; full access can be blocked with <code>POST /revoke</code>.</li>
        </ol>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What is a slug?</h3>
        <p>
          A slug is a policy anchor for new licenses. It does not represent a customer; it represents how licenses
          should behave when they are created.
        </p>
        <ul>
          <li>Defines <code>max_activations</code> (seat limit).</li>
          <li>Defines expiration policy (<code>forever</code>, <code>duration</code>, or <code>fixed_date</code>).</li>
          <li>Is resolved at generation time, then stored on the license record.</li>
          <li>Changing a slug later affects future licenses, not already-issued ones.</li>
        </ul>
        <p>
          By default, the server creates a <code>default</code> slug with one seat and no expiration.
        </p>
        <p style={{ marginTop: "0.7rem" }}>
          Want a deeper view of naming rules and policy snapshots? See <Link href="/docs/concepts/slugs/">Slug concepts</Link>.
        </p>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>How seats work</h3>
        <p>
          Seats are tracked as active activation rows per fingerprint. A seat is in use when that fingerprint has an
          activation with no <code>deactivated_at</code> value.
        </p>
        <ul>
          <li><code>/activate</code> with a new fingerprint creates a seat if the license still has capacity.</li>
          <li>Calling <code>/activate</code> again for an already-active fingerprint does not consume another seat.</li>
          <li><code>/deactivate</code> releases that fingerprint seat immediately.</li>
          <li>Revocation is different: <code>/revoke</code> blocks the whole license, not one seat.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>License API vs Management API</h3>
        <ul>
          <li>
            <strong>License API</strong>: <code>/generate</code>, <code>/revoke</code>, <code>/activate</code>,
            <code>/validate</code>, <code>/deactivate</code>.
          </li>
          <li>
            <strong>Management API</strong>: <code>/management/slugs</code>, <code>/management/api-keys</code>, and
            <code> /management/webhooks</code> routes for operator tooling and frontend-backed workflows.
          </li>
          <li>Never call key-protected routes directly from browser code.</li>
          <li>For payment webhooks, use provider event id as <code>Idempotency-Key</code> on <code>/generate</code>.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Outbound webhooks and idempotency</h3>
        <p>
          Outbound webhooks are asynchronous delivery jobs. Endpoint configuration lives in Management API, and
          delivery is triggered by License API events.
        </p>
        <ul>
          <li>
            Delivery retries can happen, so receivers should dedupe using <code>Idempotency-Key: sls-webhook-&lt;delivery_id&gt;</code>.
          </li>
          <li>
            Additional headers include <code>X-Webhook-Delivery-Id</code> and <code>X-Webhook-Event</code>.
          </li>
          <li>
            Payload shape includes top-level <code>id</code>, <code>type</code>, <code>attempt</code>, <code>occurred_at</code>, and a
            <code> data</code> object with event-specific fields.
          </li>
          <li>
            Current event types: <code>license.generated</code>, <code>license.activated</code>, <code>license.deactivated</code>,
            <code> license.validated</code>, <code>license.validation_failed</code>, <code>license.revoked</code>.
          </li>
        </ul>
        <p style={{ marginTop: "0.7rem" }}>
          For full contract details (headers, envelope, and event payload schemas), see{" "}
          <Link href="/docs/concepts/webhooks/">Webhook concepts</Link>.
        </p>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Glossary</h3>
        <GlossaryTable rows={glossaryRows} />
      </div>
    </section>
  );
}
