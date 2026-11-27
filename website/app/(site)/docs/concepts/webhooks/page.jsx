import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "Webhook Concepts",
  description:
    "Webhook delivery model and contract details for Simple License Server, including headers, envelope, and event-specific payloads.",
  alternates: {
    canonical: "/docs/concepts/webhooks/"
  }
};

const webhookEnvelopeExample = `{
  "id": 42,
  "type": "license.generated",
  "attempt": 1,
  "occurred_at": "2026-04-21T12:00:00Z",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "slug": "default",
    "status": "inactive",
    "metadata": {
      "email": "user@example.com"
    },
    "attributes": {
      "plan": "starter",
      "features": []
    },
    "expires_at": null,
    "created_at": "2026-04-21T12:00:00Z"
  }
}`;

const webhookContractExamples = {
  generated: `{
  "type": "license.generated",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "slug": "default",
    "status": "inactive",
    "metadata": { "email": "user@example.com" },
    "attributes": { "plan": "starter", "features": [] },
    "expires_at": "2026-10-20T12:00:00Z",
    "created_at": "2026-04-21T12:00:00Z"
  }
}`,
  activated: `{
  "type": "license.activated",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "status": "active",
    "expires_at": "2026-10-20T12:00:00Z"
  }
}`,
  deactivated: `{
  "type": "license.deactivated",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "reason": "user_device_replaced",
    "released": true,
    "status": "inactive",
    "active_seats": 0,
    "max_activations": 3,
    "expires_at": "2026-10-20T12:00:00Z"
  }
}`,
  validated: `{
  "type": "license.validated",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "valid": true,
    "status": "active",
    "expires_at": "2026-10-20T12:00:00Z",
    "reason": ""
  }
}`,
  validationFailed: `{
  "type": "license.validation_failed",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "valid": false,
    "status": "inactive",
    "expires_at": "2026-10-20T12:00:00Z",
    "reason": "fingerprint_not_active"
  }
}`,
  revoked: `{
  "type": "license.revoked",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "status": "revoked",
    "revoked_at": "2026-04-21T12:10:00Z"
  }
}`,
  updated: `{
  "type": "license.updated",
  "data": {
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "status": "active",
    "max_activations": 5,
    "expires_at": "2028-10-20T23:59:59Z",
    "revoked_at": null
  }
}`
};

const verifyExample = `import crypto from "node:crypto";

// secret: the whsec_... value shown when the endpoint was created (or rotated).
// rawBody: the request body exactly as received, before JSON parsing.
export function verifyWebhook(rawBody, signatureHeader, secret, toleranceSeconds = 300) {
  const parts = Object.fromEntries(
    (signatureHeader || "").split(",").map((part) => part.trim().split("="))
  );
  const timestamp = Number(parts.t);
  if (!timestamp || !parts.v1) return false;

  // Reject old deliveries so a captured request can't be replayed later.
  if (Math.abs(Date.now() / 1000 - timestamp) > toleranceSeconds) return false;

  const expected = crypto
    .createHmac("sha256", secret)
    .update(\`\${parts.t}.\`)
    .update(rawBody)
    .digest("hex");

  const a = Buffer.from(expected, "hex");
  const b = Buffer.from(parts.v1, "hex");
  return a.length === b.length && crypto.timingSafeEqual(a, b);
}`;

export default function WebhookConceptsPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Concepts</p>
      <h1>Webhook contract and delivery model</h1>
      <p>
        Webhook endpoint configuration is managed through Management API, while actual delivery is asynchronous and
        triggered by License API events.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Delivery behavior</h3>
        <ul>
          <li>Deliveries are queued and retried with backoff on transient failures.</li>
          <li>Each delivery has a stable id across retry attempts.</li>
          <li>Receivers should dedupe by idempotency key to avoid duplicate side effects.</li>
          <li>Redirects are not followed: a 3xx response fails the delivery. Configure the final URL.</li>
          <li>
            Link-local and cloud metadata addresses (such as <code>169.254.169.254</code>) are refused, including
            hostnames that resolve to them. Private network addresses are allowed for internal receivers.
          </li>
          <li>
            Use <code>https://</code>. Payloads include license keys and customer metadata; the management UI warns
            about plain <code>http://</code> endpoints.
          </li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Verifying signatures</h3>
        <p>
          Anyone who knows your receiver&apos;s URL can POST to it, so check that a delivery really came from your
          server before acting on it. Each endpoint has a signing secret (<code>whsec_...</code>), shown once when you
          create the endpoint or rotate its secret. Every delivery carries:
        </p>
        <ul>
          <li>
            <code>X-Webhook-Signature: t=&lt;unix seconds&gt;,v1=&lt;hex&gt;</code>, where <code>v1</code> is the
            HMAC-SHA256 of <code>&lt;t&gt;.&lt;raw request body&gt;</code> keyed with the secret.
          </li>
          <li>Compute the HMAC over the raw body bytes, not re-serialized JSON, and compare in constant time.</li>
          <li>Reject timestamps more than a few minutes old to stop replays.</li>
          <li>
            Rotate with <code>POST /management/webhooks/{"{id}"}/rotate-secret</code> or the Rotate button in
            Settings → Webhooks. Endpoints from 0.4.0 and earlier received a random secret on upgrade; rotate once to see
            it.
          </li>
        </ul>
        <CodeExample language="javascript" code={verifyExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Headers contract</h3>
        <ul>
          <li><code>Content-Type: application/json</code></li>
          <li><code>Idempotency-Key: sls-webhook-&lt;delivery_id&gt;</code></li>
          <li><code>X-Webhook-Delivery-Id: &lt;delivery_id&gt;</code></li>
          <li><code>X-Webhook-Signature: t=&lt;unix seconds&gt;,v1=&lt;hex HMAC-SHA256&gt;</code> (see below)</li>
          <li><code>X-Webhook-Event: &lt;event_type&gt;</code></li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Envelope contract</h3>
        <CodeExample language="json" code={webhookEnvelopeExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Event payload contracts</h3>
        <div style={{ marginTop: "0.7rem", display: "grid", gap: "0.75rem" }}>
          <article className="api-endpoint-prose">
            <h4>license.generated</h4>
            <CodeExample language="json" code={webhookContractExamples.generated} />
          </article>
          <article className="api-endpoint-prose">
            <h4>license.activated</h4>
            <CodeExample language="json" code={webhookContractExamples.activated} />
          </article>
          <article className="api-endpoint-prose">
            <h4>license.deactivated</h4>
            <p>
              Sent when an app calls <code>POST /api/v1/deactivate</code>, and when an operator removes a device, in which
              case <code>reason</code> is <code>removed_by_admin</code>.
            </p>
            <CodeExample language="json" code={webhookContractExamples.deactivated} />
          </article>
          <article className="api-endpoint-prose">
            <h4>license.validated</h4>
            <CodeExample language="json" code={webhookContractExamples.validated} />
          </article>
          <article className="api-endpoint-prose">
            <h4>license.validation_failed</h4>
            <CodeExample language="json" code={webhookContractExamples.validationFailed} />
          </article>
          <article className="api-endpoint-prose">
            <h4>license.revoked</h4>
            <p>Sent by <code>POST /api/v1/revoke</code> and by an operator revoking through the management API.</p>
            <CodeExample language="json" code={webhookContractExamples.revoked} />
          </article>
          <article className="api-endpoint-prose">
            <h4>license.updated</h4>
            <p>
              Sent when an operator changes a license with <code>PATCH /management/licenses/{"{id}"}</code>: a new seat
              limit, a new expiry, or restoring a revoked license.
            </p>
            <CodeExample language="json" code={webhookContractExamples.updated} />
          </article>
        </div>
      </div>

      <p style={{ marginTop: "1rem" }}>
        <Link className="button button-secondary" href="/docs/concepts/">
          Back to concepts overview
        </Link>
      </p>
    </section>
  );
}
