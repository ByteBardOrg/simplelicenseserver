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
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "slug": "default",
    "status": "inactive",
    "metadata": {
      "email": "user@example.com"
    },
    "expires_at": null,
    "created_at": "2026-04-21T12:00:00Z"
  }
}`;

const webhookContractExamples = {
  generated: `{
  "type": "license.generated",
  "data": {
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "slug": "default",
    "status": "inactive",
    "metadata": { "email": "user@example.com" },
    "expires_at": "2026-10-20T12:00:00Z",
    "created_at": "2026-04-21T12:00:00Z"
  }
}`,
  activated: `{
  "type": "license.activated",
  "data": {
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "status": "active",
    "expires_at": "2026-10-20T12:00:00Z"
  }
}`,
  deactivated: `{
  "type": "license.deactivated",
  "data": {
    "license_key": "XXXX-XXXX-XXXX-XXXX",
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
    "license_key": "XXXX-XXXX-XXXX-XXXX",
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
    "license_key": "XXXX-XXXX-XXXX-XXXX",
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
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "status": "revoked",
    "revoked_at": "2026-04-21T12:10:00Z"
  }
}`
};

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
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Headers contract</h3>
        <ul>
          <li><code>Content-Type: application/json</code></li>
          <li><code>Idempotency-Key: sls-webhook-&lt;delivery_id&gt;</code></li>
          <li><code>X-Webhook-Delivery-Id: &lt;delivery_id&gt;</code></li>
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
            <CodeExample language="json" code={webhookContractExamples.revoked} />
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
