export const metadata = {
  title: "Concepts",
  description:
    "Core concepts in Simple License Server: slugs, licenses, seats, expiration, and provisioning vs runtime flows.",
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
    term: "Provisioning API",
    meaning:
      "Trusted backend endpoints (/generate, /revoke) that require a server API key."
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
        <h3>Provisioning vs runtime endpoints</h3>
        <ul>
          <li>
            <strong>Provisioning</strong>: <code>/generate</code>, <code>/revoke</code> (trusted backend only, server API key
            required).
          </li>
          <li>
            <strong>Runtime</strong>: <code>/activate</code>, <code>/validate</code>, <code>/deactivate</code> (called by your app).
          </li>
          <li>Never call provisioning endpoints from browser code.</li>
          <li>For payment webhooks, use provider event id as <code>Idempotency-Key</code> on <code>/generate</code>.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Glossary</h3>
        <GlossaryTable rows={glossaryRows} />
      </div>
    </section>
  );
}
