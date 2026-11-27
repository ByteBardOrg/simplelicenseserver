import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "0.4.0 -> 0.5.0 Migration",
  description: "How to upgrade from Simple License Server 0.4.0 to 0.5.0: /api/v1 routes and one runtime response shape.",
  alternates: {
    canonical: "/docs/migration/0-4-0-to-0-5-0/"
  }
};

const pathChanges = `POST /generate    ->  POST /api/v1/generate
POST /revoke      ->  POST /api/v1/revoke
POST /activate    ->  POST /api/v1/activate
POST /validate    ->  POST /api/v1/validate
POST /deactivate  ->  POST /api/v1/deactivate`;

const oldValidate = `// 0.4.0: unknown key
HTTP 404  { "error": "license not found" }

// 0.4.0: known key
HTTP 200  { "valid": true, "status": "active", "expires_at": "...", "token": "...", "reason": "" }`;

const newValidate = `// 0.5.0: every decided outcome is HTTP 200 with the same shape
{
  "valid": true,
  "reason": null,
  "license": {
    "status": "active",
    "expires_at": "2027-10-20T12:00:00Z",
    "seats": { "used": 1, "max": 3 },
    "attributes": { "plan": "pro" }
  },
  "token": "jwt-if-slug-offline-enabled"
}

// 0.5.0: unknown key
{ "valid": false, "reason": "not_found", "license": null }`;

const clientExample = `const res = await fetch(SERVER + "/api/v1/validate", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ license_key, fingerprint })
});
const result = await res.json();

if (result.valid && result.license.product === "desktop-app") {
  unlock(result.license.attributes);
} else if (result.reason === "fingerprint_not_active") {
  // seat was freed or never activated on this device: try activating again
} else {
  showLicenseProblem(result.reason); // not_found, revoked, expired, activation_limit_reached
}`;

export default function Migration040To050Page() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Migration</p>
      <h1>0.4.0 -&gt; 0.5.0</h1>
      <p>
        Version 0.5.0 is a <strong>breaking</strong> release. License routes move under <code>/api/v1</code>, and the
        three runtime endpoints now share one response shape. Management routes are unchanged apart from three
        additions, and slugs gain an optional product.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>1. Route paths moved under /api/v1</h3>
        <p>The old root paths are removed and now return 404.</p>
        <CodeExample language="text" code={pathChanges} />
        <p>
          <code>/healthz</code> and everything under <code>/management</code> keep their paths. Request bodies and
          authentication are unchanged.
        </p>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>2. One response shape for activate, validate and deactivate</h3>
        <p>Before:</p>
        <CodeExample language="javascript" code={oldValidate} />
        <p>After:</p>
        <CodeExample language="javascript" code={newValidate} />
        <ul>
          <li>
            Unknown keys return HTTP 200 with <code>reason: "not_found"</code> instead of a 404. Your app only needs
            to check <code>valid</code>.
          </li>
          <li>
            <code>reason</code> is <code>null</code> when valid. Otherwise it is one of <code>not_found</code>,{" "}
            <code>revoked</code>, <code>expired</code>, <code>activation_limit_reached</code> or{" "}
            <code>fingerprint_not_active</code>.
          </li>
          <li>
            <code>status</code> and <code>expires_at</code> moved into <code>license</code>. <code>status</code> can
            now be <code>expired</code>.
          </li>
          <li>
            New: <code>license.seats</code> and <code>license.attributes</code>, so online-only slugs can unlock
            features without offline tokens.
          </li>
          <li>
            Removed: the echoed <code>license_key</code> and <code>fingerprint</code> on activate, and the top-level{" "}
            <code>released</code>, <code>active_seats</code> and <code>max_activations</code> on deactivate. Use{" "}
            <code>license.seats</code>.
          </li>
          <li>
            On deactivate, <code>valid</code> now reports whether the license is still usable rather than always being
            true.
          </li>
          <li>
            These endpoints never return license <code>metadata</code>, slug names, or other devices, because anyone
            holding the key can call them.
          </li>
        </ul>
        <p>A client check in 0.5.0:</p>
        <CodeExample language="javascript" code={clientExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>3. New management endpoints</h3>
        <ul>
          <li>
            <Link href="/docs/api/management-licenses-get/">
              <code>GET /management/licenses/{"{id}"}</code>
            </Link>
            : one license with its active devices.
          </li>
          <li>
            <Link href="/docs/api/management-licenses-update/">
              <code>PATCH /management/licenses/{"{id}"}</code>
            </Link>
            : change <code>max_activations</code> or <code>expires_at</code>, or set <code>revoked</code> to revoke and
            restore. Revoking is no longer permanent.
          </li>
          <li>
            <Link href="/docs/api/management-licenses-remove-activation/">
              <code>DELETE /management/licenses/{"{id}"}/activations/{"{activationID}"}</code>
            </Link>
            : free a device's seat on the customer's behalf.
          </li>
          <li>
            New webhook event <code>license.updated</code>. Admin device removal sends <code>license.deactivated</code>{" "}
            with reason <code>removed_by_admin</code>.
          </li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>4. Products on slugs</h3>
        <p>
          All offline tokens are signed by the same key, and before 0.5.0 nothing in a response said which app a license
          was for, so a key sold for one product could unlock another. Slugs now have an optional{" "}
          <code>product</code>.
        </p>
        <ul>
          <li>Set the same product (for example <code>desktop-app</code>) on every slug for one app.</li>
          <li>
            Activate, validate and deactivate return it as <code>license.product</code>. Compare it with a value
            hard-coded in your app, as you would with <code>store_id</code>/<code>product_id</code> on Lemon Squeezy.
          </li>
          <li>
            Offline tokens use it as <code>aud</code>, falling back to <code>OFFLINE_TOKEN_AUDIENCE</code>. Check{" "}
            <code>aud</code> and <code>fingerprint</code> when verifying tokens; see{" "}
            <Link href="/docs/concepts/offline-licenses/">offline licenses</Link>.
          </li>
          <li>Slugs without a product behave exactly as before, with <code>license.product</code> set to null.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>5. Client IP and abuse protection</h3>
        <ul>
          <li>
            <code>TRUST_PROXY_HEADERS</code> is replaced by <code>CLIENT_IP_HEADER</code>. The old setting trusted the
            leftmost <code>X-Forwarded-For</code> entry, which clients control. If it is set to <code>true</code>, the
            server now refuses to start and says what to set instead. Behind Cloudflare use{" "}
            <code>CF-Connecting-IP</code>; see{" "}
            <Link href="/docs/configuration/">running behind a proxy</Link>.
          </li>
          <li>
            New lockout: 10 wrong management or server API keys from one IP within 15 minutes return HTTP 429 until the
            window ends (<code>AUTH_FAILURE_MAX_ATTEMPTS</code>, <code>AUTH_FAILURE_WINDOW</code>).
          </li>
          <li>
            The global rate limit (<code>RATE_LIMIT_GLOBAL_RPS</code>) is now off by default. Public runtime routes and
            admin routes have separate per-IP budgets, and the UI&apos;s static files are no longer rate limited.
          </li>
          <li>When the client table is full, the least recently seen client is dropped instead of refusing new clients.</li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>6. Security hardening</h3>
        <ul>
          <li>
            <strong>Example secrets are refused.</strong> The server will not start with management keys or signing
            secrets published in these docs (such as <code>management_key_dev_123456</code>). Generate your own with{" "}
            <code>openssl rand -hex 32</code>. <code>ALLOW_EXAMPLE_SECRETS=true</code> exists only for local
            development.
          </li>
          <li>
            <strong>Admin license routes use the license id</strong>, not the key:{" "}
            <code>/management/licenses/{"{id}"}</code>. License keys no longer appear in URLs or access logs.{" "}
            <code>POST /api/v1/generate</code> now returns the <code>id</code>; list responses already included it.
          </li>
          <li>
            <strong>Webhooks are signed.</strong> Each delivery has an <code>X-Webhook-Signature</code> header; verify
            it in your receivers (<Link href="/docs/concepts/webhooks/">how</Link>). Existing endpoints get a random
            secret on upgrade: rotate it once in Settings → Webhooks to obtain the value.
          </li>
          <li>
            <strong>Webhook delivery rules:</strong> redirects are no longer followed, and link-local / cloud metadata
            addresses are refused.
          </li>
          <li>
            <strong>The container runs as a non-root user</strong> (uid 65532) instead of root. The API container
            keeps no files, so no volume changes are needed.
          </li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Required actions</h3>
        <ul>
          <li>Update your backend's <code>/generate</code> and <code>/revoke</code> calls to the <code>/api/v1</code> paths.</li>
          <li>Update app code that calls activate, validate or deactivate to the new paths and response shape.</li>
          <li>If you filter webhook events, decide whether to subscribe endpoints to <code>license.updated</code>.</li>
          <li>Recommended: set a product on your slugs, and check <code>license.product</code> and the token <code>aud</code> in your apps.</li>
          <li>Replace <code>TRUST_PROXY_HEADERS</code> with <code>CLIENT_IP_HEADER</code> if you run behind a proxy.</li>
          <li>If you use an example key or signing secret from the docs, replace it before upgrading or the server will not start.</li>
          <li>Address licenses by <code>id</code> in any tooling that calls the new admin license endpoints.</li>
          <li>Rotate each webhook endpoint&apos;s secret and add signature verification to your receivers.</li>
          <li>Pull or rebuild the 0.5.0 Docker image. The database migrates itself on startup.</li>
        </ul>
        <p>
          Apps built against 0.4.0 keep working only as long as the server stays on 0.4.0, so upgrade the server and
          your clients together.
        </p>
      </div>
    </section>
  );
}
