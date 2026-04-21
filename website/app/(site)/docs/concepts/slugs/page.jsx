import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "Slug Concepts",
  description:
    "How slug policy templates work in Simple License Server, including naming rules, expiration policies, and issuance snapshot behavior.",
  alternates: {
    canonical: "/docs/concepts/slugs/"
  }
};

const slugPolicyExample = `{
  "name": "pro-annual",
  "max_activations": 3,
  "expiration_type": "duration",
  "expiration_days": 365,
  "fixed_expires_at": null
}`;

const generatedFromSlugExample = `{
  "license_key": "C1B6DE-39A6E3-DE1529-8559A0-4AF593",
  "slug": "pro-annual",
  "status": "inactive",
  "expires_at": "2027-04-21T12:00:00Z",
  "created_at": "2026-04-21T12:00:00Z"
}`;

export default function SlugConceptsPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Concepts</p>
      <h1>Slug policy model</h1>
      <p>
        A slug is a policy template for future licenses. It defines seat limits and expiration behavior, and those
        values are copied onto each license at generation time.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What a slug controls</h3>
        <ul>
          <li>
            <code>max_activations</code>: maximum concurrent active fingerprints (seats).
          </li>
          <li>
            Expiration policy via <code>expiration_type</code>: <code>forever</code>, <code>duration</code>, or
            <code> fixed_date</code>.
          </li>
          <li>
            Optional policy fields: <code>expiration_days</code> for <code>duration</code>, or
            <code> fixed_expires_at</code> for <code>fixed_date</code>.
          </li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Name rules</h3>
        <p>Slug names are strict and URL-safe so they can be safely used in API paths and integration code.</p>
        <ul>
          <li>Lower-case only.</li>
          <li>Allowed characters: <code>a-z</code>, <code>0-9</code>, and <code>-</code>.</li>
          <li>Must start with an alphanumeric character.</li>
          <li>Example valid names: <code>default</code>, <code>pro-annual</code>, <code>team-v2</code>.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Snapshot behavior at issuance</h3>
        <p>
          When your backend calls <code>POST /generate</code>, the server resolves slug policy and stores it on the
          license. Later slug edits affect new licenses, not existing ones.
        </p>
        <p>Example slug policy:</p>
        <CodeExample language="json" code={slugPolicyExample} />
        <p style={{ marginTop: "0.7rem" }}>Example generated license state:</p>
        <CodeExample language="json" code={generatedFromSlugExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Management flow</h3>
        <ul>
          <li>
            Create and update slug templates with Management API routes under <code>/management/slugs</code>.
          </li>
          <li>
            Generate licenses by sending that slug name to <code>POST /generate</code>.
          </li>
          <li>
            Keep stable slug names in integrations; rename intentionally and only when you can update callers.
          </li>
        </ul>
        <p style={{ marginTop: "0.7rem" }}>
          For endpoint-level request and response payloads, see <Link href="/docs/management-api/">Management API</Link>.
        </p>
      </div>

      <p style={{ marginTop: "1rem", display: "flex", gap: "0.6rem", flexWrap: "wrap" }}>
        <Link className="button button-secondary" href="/docs/concepts/">
          Back to concepts overview
        </Link>
        <Link className="button button-primary" href="/docs/management-api/">
          Open Management API
        </Link>
      </p>
    </section>
  );
}
