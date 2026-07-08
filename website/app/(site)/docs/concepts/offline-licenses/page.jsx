import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "Offline License Concepts",
  description:
    "How per-slug offline JWT licensing, signing keys, token refresh, and public key export work in Simple License Server.",
  alternates: {
    canonical: "/docs/concepts/offline-licenses/"
  }
};

const tokenExample = `{
  "iss": "simple-license-server",
  "sub": "5345c73e-e338-4f5d-8585-dc9eff7c84e5",
  "slug": "desktop-pro",
  "fingerprint": "abc123-machine-fingerprint",
  "attributes": {
    "plan": "pro",
    "features": ["analytics", "exports"]
  },
  "iat": 1777749983,
  "nbf": 1777749983,
  "exp": 1777836383,
  "jti": "d1db30b604fe469d32fe47533fe93ed3"
}`;

export default function OfflineLicenseConceptsPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Concepts</p>
      <h1>Offline license model</h1>
      <p>
        Offline licenses are short-lived JWTs returned by <code>/activate</code> and refreshed by valid <code>/validate</code>
        calls. They let a client keep validating locally for a bounded period when it cannot reach the server.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Per-slug control</h3>
        <p>
          Offline support is enabled on each slug with <code>offline_enabled</code>. Token lifetime is also set on the
          slug with <code>offline_token_lifetime_hours</code>. There is no global offline switch or global token lifetime.
          All slugs, including <code>default</code>, start offline-disabled unless an operator enables the slug.
        </p>
        <ul>
          <li>When <code>offline_enabled=false</code>, activation and validation remain online-only and omit <code>token</code>.</li>
          <li>When <code>offline_enabled=true</code>, valid activation and validation responses include <code>token</code> if an active signing key exists.</li>
          <li>Changing a slug controls future token issuance immediately; it does not revoke already-issued offline tokens.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Signing keys</h3>
        <p>
          The server generates Ed25519 signing key pairs. Private keys are encrypted at rest with
          <code> OFFLINE_SIGNING_ENCRYPTION_KEY</code>, and clients verify tokens using exported public keys.
        </p>
        <ul>
          <li><code>active</code>: signs newly issued and refreshed offline tokens.</li>
          <li><code>verify_only</code>: exported for verifying older tokens during key rollover.</li>
          <li><code>retired</code>: no longer signs tokens and is omitted from public key exports.</li>
          <li>Each key has a <code>kid</code> so clients can choose the right public key for a token.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Token refresh behavior</h3>
        <ul>
          <li><code>POST /activate</code> returns <code>token</code> only after a valid activation for an offline-enabled slug.</li>
          <li><code>POST /validate</code> returns a refreshed <code>token</code> only after a valid online validation for an offline-enabled slug.</li>
          <li>If no active signing key exists, online validation still succeeds but no token is returned.</li>
          <li><code>offline_token_lifetime_hours</code> defaults to <code>24</code>; license expiration caps the token expiration when sooner.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>JWT claims</h3>
        <p>Clients should validate the signature, expiration, issuer, and local fingerprint before trusting a token.</p>
        <ul>
          <li>Use the nested <code>attributes</code> object claim for feature gating or plan logic.</li>
          <li>The <code>attributes</code> claim is a snapshot from the license at token-issuance time.</li>
        </ul>
        <CodeExample language="json" code={tokenExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Revocation and compromise tradeoffs</h3>
        <p>
          Offline tokens are intentionally disconnected from the server until they expire or the client reconnects.
          Revocation takes effect on the next online check, or when the current offline token expires.
        </p>
        <ul>
          <li>Use short token lifetimes to reduce revocation lag.</li>
          <li>Retire compromised signing keys and ship updated public key bundles to clients.</li>
          <li>Already-offline clients cannot learn about a retired key until they reconnect or receive an app/config update.</li>
        </ul>
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
