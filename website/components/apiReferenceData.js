const webhookSupportedEvents = [
  "license.generated",
  "license.activated",
  "license.deactivated",
  "license.validated",
  "license.validation_failed",
  "license.revoked",
  "license.updated"
];

export const apiEndpointGroups = {
  license: {
    key: "license",
    title: "License API",
    summary:
      "Versioned /api/v1 routes: provisioning (generate, revoke) for your backend and runtime checks (activate, validate, deactivate) for your app.",
    description:
      "All license routes live under /api/v1. /api/v1/generate and /api/v1/revoke are called from your backend and require a server API key. /api/v1/activate, /api/v1/validate and /api/v1/deactivate are called from your app with license_key plus fingerprint and need no other credentials; they share one response shape.",
    endpoints: ["generate", "revoke", "activate", "validate", "deactivate"]
  },
  management: {
    key: "management",
    title: "Management API",
    summary:
      "Operator-facing routes for license inspection, slug policy management, API key lifecycle, webhook endpoint configuration/logs, and offline signing keys.",
    description:
      "Use these endpoints from internal tooling or your frontend backend to inspect licenses, manage slug policy templates, provisioning API keys, webhook endpoint configuration/logs, and offline JWT signing keys. Management routes require the management bootstrap key configured via MANAGEMENT_API_KEYS or MANAGEMENT_API_KEY.",
    endpoints: [
      "management-licenses-list",
      "management-licenses-get",
      "management-licenses-update",
      "management-licenses-remove-activation",
      "management-slugs-list",
      "management-slugs-create",
      "management-slugs-get",
      "management-slugs-update",
      "management-slugs-delete",
      "management-api-keys-list",
      "management-api-keys-create",
      "management-api-keys-revoke",
      "management-webhooks-list",
      "management-webhooks-deliveries",
      "management-webhooks-create",
      "management-webhooks-update",
      "management-webhooks-delete",
      "management-offline-signing-keys-list",
      "management-offline-signing-keys-create",
      "management-offline-signing-keys-activate",
      "management-offline-signing-keys-retire",
      "management-offline-public-keys-list"
    ]
  }
};

export const apiEndpoints = {
  "management-offline-signing-keys-list": {
    key: "management-offline-signing-keys-list",
    group: "management",
    label: "GET /management/offline/signing-keys",
    method: "GET",
    path: "/management/offline/signing-keys",
    title: "List Offline Signing Keys",
    summary: "List offline JWT signing keys and their lifecycle state.",
    details: [
      "Requires management API key auth.",
      "Private key material is never returned.",
      "Use this endpoint for the Offline Licenses management UI and operator audits."
    ],
    curl: `curl -sS http://localhost:8080/management/offline/signing-keys \
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "signing_keys": [
    {
      "id": 1,
      "name": "desktop-app-2026",
      "kid": "61426d843cf40341f68a3053",
      "algorithm": "Ed25519",
      "status": "active",
      "created_at": "2026-04-21T12:00:00Z",
      "activated_at": "2026-04-21T12:05:00Z",
      "retired_at": null
    }
  ]
}`,
    responseFields: [
      { name: "signing_keys", type: "array", description: "Signing key metadata records." },
      { name: "signing_keys[].kid", type: "string", description: "JWT key id emitted in the token header." },
      { name: "signing_keys[].algorithm", type: '"Ed25519"', description: "Signing algorithm for this key." },
      { name: "signing_keys[].status", type: '"active" | "verify_only" | "retired"', description: "Lifecycle state. Only active signs new tokens." }
    ],
    errorCodes: [
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-offline-signing-keys-create": {
    key: "management-offline-signing-keys-create",
    group: "management",
    label: "POST /management/offline/signing-keys",
    method: "POST",
    path: "/management/offline/signing-keys",
    title: "Create Offline Signing Key",
    summary: "Generate a new Ed25519 key pair for offline JWT signing.",
    details: [
      "Requires management API key auth.",
      "Requires OFFLINE_SIGNING_ENCRYPTION_KEY to be configured with at least 32 characters.",
      "New keys start in verify_only mode until activated.",
      "The encrypted private key is stored server-side; public key material is returned for export."
    ],
    curl: `curl -sS http://localhost:8080/management/offline/signing-keys \
  -H "Authorization: Bearer management_key_dev_123456" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "desktop-app-2026"
  }'`,
    requestBody: `{
  "name": "desktop-app-2026"
}`,
    responseBody: `{
  "id": 1,
  "name": "desktop-app-2026",
  "kid": "61426d843cf40341f68a3053",
  "algorithm": "Ed25519",
  "status": "verify_only",
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n",
  "created_at": "2026-04-21T12:00:00Z",
  "activated_at": null,
  "retired_at": null
}`,
    responseFields: [
      { name: "id", type: "number", description: "Signing key id." },
      { name: "kid", type: "string", description: "JWT key id for this key pair." },
      { name: "public_key_pem", type: "string", description: "PEM public key to ship with offline-validating clients." },
      { name: "status", type: '"verify_only"', description: "New keys do not sign tokens until activated." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, oversized name, or missing/invalid OFFLINE_SIGNING_ENCRYPTION_KEY." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 409, reason: "Generated key id conflict; retry the request." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-offline-signing-keys-activate": {
    key: "management-offline-signing-keys-activate",
    group: "management",
    label: "POST /management/offline/signing-keys/{id}/activate",
    method: "POST",
    path: "/management/offline/signing-keys/{id}/activate",
    title: "Activate Offline Signing Key",
    summary: "Promote one signing key to active so it signs new offline JWTs.",
    details: [
      "Requires management API key auth.",
      "Only one signing key can be active at a time.",
      "The previously active key is moved to verify_only so older tokens can continue validating until retired."
    ],
    curl: `curl -sS -X POST http://localhost:8080/management/offline/signing-keys/1/activate \
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "id": 1,
  "name": "desktop-app-2026",
  "kid": "61426d843cf40341f68a3053",
  "algorithm": "Ed25519",
  "status": "active",
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n",
  "created_at": "2026-04-21T12:00:00Z",
  "activated_at": "2026-04-21T12:05:00Z",
  "retired_at": null
}`,
    responseFields: [
      { name: "status", type: '"active"', description: "The key is now used for newly issued/refreshed offline JWTs." },
      { name: "activated_at", type: "string", description: "RFC3339 timestamp when the key first became active." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid key id." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Signing key not found." },
      { status: 409, reason: "Retired signing key cannot be activated." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-offline-signing-keys-retire": {
    key: "management-offline-signing-keys-retire",
    group: "management",
    label: "POST /management/offline/signing-keys/{id}/retire",
    method: "POST",
    path: "/management/offline/signing-keys/{id}/retire",
    title: "Retire Offline Signing Key",
    summary: "Retire a signing key so it is no longer exported as trusted verification material.",
    details: [
      "Requires management API key auth.",
      "Retired keys do not sign new tokens and are omitted from public key exports.",
      "Clients that remove the public key will reject existing offline tokens signed by that kid."
    ],
    curl: `curl -sS -X POST http://localhost:8080/management/offline/signing-keys/1/retire \
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "id": 1,
  "name": "desktop-app-2026",
  "kid": "61426d843cf40341f68a3053",
  "algorithm": "Ed25519",
  "status": "retired",
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n",
  "created_at": "2026-04-21T12:00:00Z",
  "activated_at": "2026-04-21T12:05:00Z",
  "retired_at": "2026-04-22T12:00:00Z"
}`,
    responseFields: [
      { name: "status", type: '"retired"', description: "The key has been removed from active/verify-only trust." },
      { name: "retired_at", type: "string", description: "RFC3339 timestamp when retirement was recorded." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid key id." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Signing key not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-offline-public-keys-list": {
    key: "management-offline-public-keys-list",
    group: "management",
    label: "GET /management/offline/public-keys",
    method: "GET",
    path: "/management/offline/public-keys",
    title: "List Offline Public Keys",
    summary: "Export active and verify-only public keys for offline-validating clients.",
    details: [
      "Requires management API key auth.",
      "Returns only active and verify_only signing keys.",
      "Ship this public verification material with client applications that validate offline JWTs."
    ],
    curl: `curl -sS http://localhost:8080/management/offline/public-keys \
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "signing_keys": [
    {
      "id": 1,
      "name": "desktop-app-2026",
      "kid": "61426d843cf40341f68a3053",
      "algorithm": "Ed25519",
      "status": "active",
      "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n",
      "created_at": "2026-04-21T12:00:00Z",
      "activated_at": "2026-04-21T12:05:00Z",
      "retired_at": null
    }
  ]
}`,
    responseFields: [
      { name: "signing_keys", type: "array", description: "Active and verify-only public signing key records." },
      { name: "signing_keys[].public_key_pem", type: "string", description: "PEM public key for JWT verification." },
      { name: "signing_keys[].kid", type: "string", description: "JWT header kid this public key verifies." }
    ],
    errorCodes: [
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  generate: {
    key: "generate",
    group: "license",
    label: "POST /api/v1/generate",
    method: "POST",
    path: "/api/v1/generate",
    title: "Generate License",
    summary: "Create a new license for a slug. Intended for trusted backend systems after payment success.",
    details: [
      "Requires a generated active server API key.",
      "Supports Idempotency-Key for safe webhook retries.",
      "Resolves slug expiration policy at generation time and stores expires_at on the license.",
      "Snapshots slug attributes at generation time and returns them as license attributes."
    ],
    curl: `curl -X POST http://localhost:8080/api/v1/generate \\
  -H "Authorization: Bearer <generated_server_api_key>" \\
  -H "Idempotency-Key: evt_123" \\
  -H "Content-Type: application/json" \\
  -d '{
    "slug": "default",
    "metadata": {
      "email": "user@example.com",
      "external_customer_id": "cus_123",
      "external_order_id": "ord_456"
    }
  }'`,
    requestBody: `{
  "slug": "my-product",
  "metadata": {
    "email": "user@example.com",
    "external_customer_id": "cus_123",
    "external_order_id": "ord_456"
  }
}`,
    responseBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "slug": "my-product",
  "status": "inactive",
  "metadata": {
    "email": "user@example.com",
    "external_customer_id": "cus_123",
    "external_order_id": "ord_456"
  },
  "attributes": {
    "plan": "pro",
    "features": ["analytics", "exports"]
  },
  "expires_at": "2026-10-20T12:00:00Z",
  "created_at": "2026-04-20T12:00:00Z"
}`,
    responseFields: [
      { name: "license_key", type: "string", description: "Generated license key." },
      { name: "slug", type: "string", description: "Slug used at generation time." },
      { name: "status", type: '"inactive" | "active" | "revoked"', description: "Current license status." },
      { name: "metadata", type: "object", description: "Caller-provided JSON from POST /api/v1/generate for per-license business context (for example customer/order ids). Never returned by the public runtime endpoints." },
      { name: "attributes", type: "object", description: "Slug-defined entitlement JSON (for example sku/edition/features/channel) snapshotted onto the license at generation time." },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 expiration timestamp, or null for non-expiring policies."
      },
      { name: "created_at", type: "string", description: "RFC3339 creation timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing slug, oversized fields, or invalid metadata shape." },
      { status: 401, reason: "Missing or invalid API key." },
      { status: 404, reason: "Referenced slug not found." },
      { status: 409, reason: "Idempotency key conflict or in-progress idempotent request." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  revoke: {
    key: "revoke",
    group: "license",
    label: "POST /api/v1/revoke",
    method: "POST",
    path: "/api/v1/revoke",
    title: "Revoke License",
    summary: "Revoke a license key. Revoked licenses cannot be activated or validated until restored.",
    details: [
      "Requires a generated active server API key.",
      "Revocation is license-level, not seat-level.",
      "After revocation, runtime checks return valid=false with reason revoked.",
      "An operator can restore a revoked license with PATCH /management/licenses/{key} and revoked=false."
    ],
    curl: `curl -X POST http://localhost:8080/api/v1/revoke \\
  -H "Authorization: Bearer <generated_server_api_key>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX"
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX"
}`,
    responseBody: `{
  "valid": false,
  "status": "revoked",
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "revoked_at": "2026-04-20T12:00:00Z"
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "Always false for a successful revoke response." },
      { name: "status", type: '"revoked"', description: "License status after revocation." },
      { name: "license_key", type: "string", description: "License key that was revoked." },
      { name: "revoked_at", type: "string", description: "RFC3339 timestamp when revocation was recorded." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON or missing license_key." },
      { status: 401, reason: "Missing or invalid API key." },
      { status: 404, reason: "License not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  activate: {
    key: "activate",
    group: "license",
    label: "POST /api/v1/activate",
    method: "POST",
    path: "/api/v1/activate",
    title: "Activate License",
    summary: "Bind a license key to a device fingerprint. Usually the first call after the user enters their key.",
    details: [
      "Checks revoked/expired state and the seat limit, then creates or confirms an activation for the fingerprint.",
      "Activating the same fingerprint again is safe and does not use another seat.",
      "Send a stable fingerprint, ideally a hash of the machine id rather than the raw hardware id.",
      "Optional metadata is stored on the activation. A string metadata.name is shown in the management UI device list.",
      "Check license.product against your app's own product so keys sold for another product are rejected.",
      "Returns HTTP 200 for every decided outcome, including unknown keys (valid=false, reason=not_found, license=null). 4xx is only used for malformed requests and rate limits.",
      "Never returns license metadata, slug names, internal ids, or other devices: anyone holding the key can call this endpoint."
    ],
    curl: `curl -X POST http://localhost:8080/api/v1/activate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "sha256:9f86d081884c7d65",
    "metadata": {
      "name": "Work laptop",
      "platform": "macos"
    }
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "fingerprint": "sha256:9f86d081884c7d65",
  "metadata": {
    "name": "Work laptop",
    "platform": "macos"
  }
}`,
    responseBody: `{
  "valid": true,
  "reason": null,
  "license": {
    "product": "desktop-app",
    "status": "active",
    "expires_at": "2027-10-20T12:00:00Z",
    "seats": { "used": 1, "max": 3 },
    "attributes": {
      "plan": "pro",
      "features": ["analytics", "exports"]
    }
  },
  "token": "jwt-if-slug-offline-enabled"
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "The only field your app needs to branch on." },
      {
        name: "reason",
        type: "string | null",
        description: "Why valid is false: not_found, revoked, expired, activation_limit_reached, or fingerprint_not_active. null when valid."
      },
      {
        name: "license",
        type: "object | null",
        description: "The license as your app may see it. null only when reason is not_found."
      },
      {
        name: "license.product",
        type: "string | null",
        description: "The slug's product. Compare it with a value hard-coded in your app to reject keys sold for a different product."
      },
      {
        name: "license.status",
        type: '"inactive" | "active" | "revoked" | "expired"',
        description: "Display status. expired is derived from expires_at."
      },
      { name: "license.expires_at", type: "string | null", description: "RFC3339 expiry, or null when the license never expires." },
      { name: "license.seats.used", type: "number", description: "Devices currently activated on this license." },
      { name: "license.seats.max", type: "number", description: "Maximum number of devices." },
      { name: "license.attributes", type: "object", description: "Slug attributes snapshotted onto the license, for unlocking features." },
      {
        name: "token",
        type: "string (optional)",
        description: "Offline JWT, present only when valid is true, the slug has offline tokens enabled, and an active signing key exists. Includes an attributes claim."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing required fields, or oversized fields." },
      { status: 429, reason: "Rate limited." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  validate: {
    key: "validate",
    group: "license",
    label: "POST /api/v1/validate",
    method: "POST",
    path: "/api/v1/validate",
    title: "Validate License",
    summary: "Routine check that a license is still good on this device.",
    details: [
      "Verifies the license state and that this fingerprint holds an active seat.",
      "Returns reason fingerprint_not_active when the device was never activated or its seat was removed; activate again to recover.",
      "Refreshes the offline token when the slug has offline tokens enabled.",
      "Check license.product against your app's own product so keys sold for another product are rejected.",
      "Returns HTTP 200 for every decided outcome, including unknown keys (valid=false, reason=not_found, license=null). 4xx is only used for malformed requests and rate limits.",
      "Never returns license metadata, slug names, internal ids, or other devices: anyone holding the key can call this endpoint."
    ],
    curl: `curl -X POST http://localhost:8080/api/v1/validate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "sha256:9f86d081884c7d65"
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "fingerprint": "sha256:9f86d081884c7d65"
}`,
    responseBody: `{
  "valid": false,
  "reason": "expired",
  "license": {
    "product": "desktop-app",
    "status": "expired",
    "expires_at": "2026-10-20T12:00:00Z",
    "seats": { "used": 1, "max": 3 },
    "attributes": {
      "plan": "pro",
      "features": ["analytics", "exports"]
    }
  }
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "The only field your app needs to branch on." },
      {
        name: "reason",
        type: "string | null",
        description: "Why valid is false: not_found, revoked, expired, activation_limit_reached, or fingerprint_not_active. null when valid."
      },
      {
        name: "license",
        type: "object | null",
        description: "The license as your app may see it. null only when reason is not_found."
      },
      {
        name: "license.product",
        type: "string | null",
        description: "The slug's product. Compare it with a value hard-coded in your app to reject keys sold for a different product."
      },
      {
        name: "license.status",
        type: '"inactive" | "active" | "revoked" | "expired"',
        description: "Display status. expired is derived from expires_at."
      },
      { name: "license.expires_at", type: "string | null", description: "RFC3339 expiry, or null when the license never expires." },
      { name: "license.seats.used", type: "number", description: "Devices currently activated on this license." },
      { name: "license.seats.max", type: "number", description: "Maximum number of devices." },
      { name: "license.attributes", type: "object", description: "Slug attributes snapshotted onto the license, for unlocking features." },
      {
        name: "token",
        type: "string (optional)",
        description: "Offline JWT, present only when valid is true, the slug has offline tokens enabled, and an active signing key exists. Includes an attributes claim."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing required fields, or oversized fields." },
      { status: 429, reason: "Rate limited." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  deactivate: {
    key: "deactivate",
    group: "license",
    label: "POST /api/v1/deactivate",
    method: "POST",
    path: "/api/v1/deactivate",
    title: "Deactivate Seat",
    summary: "Release this device's seat so the license can be used on another device.",
    details: [
      "Useful for sign-out, reinstalls, and moving to new hardware.",
      "Idempotent: repeated calls never release more than one seat, and license.seats.used shows the result.",
      "valid reflects whether the license itself is still usable (not revoked or expired).",
      "Operators can also free a seat without the device via DELETE /management/licenses/{key}/activations/{id}.",
      "Returns HTTP 200 for every decided outcome, including unknown keys (valid=false, reason=not_found, license=null). 4xx is only used for malformed requests and rate limits.",
      "Never returns license metadata, slug names, internal ids, or other devices: anyone holding the key can call this endpoint."
    ],
    curl: `curl -X POST http://localhost:8080/api/v1/deactivate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "sha256:9f86d081884c7d65",
    "reason": "user_signed_out"
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "fingerprint": "sha256:9f86d081884c7d65",
  "reason": "user_signed_out"
}`,
    responseBody: `{
  "valid": true,
  "reason": null,
  "license": {
    "product": "desktop-app",
    "status": "inactive",
    "expires_at": "2027-10-20T12:00:00Z",
    "seats": { "used": 0, "max": 3 },
    "attributes": {
      "plan": "pro",
      "features": ["analytics", "exports"]
    }
  }
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "The only field your app needs to branch on." },
      {
        name: "reason",
        type: "string | null",
        description: "Why valid is false: not_found, revoked, expired, activation_limit_reached, or fingerprint_not_active. null when valid."
      },
      {
        name: "license",
        type: "object | null",
        description: "The license as your app may see it. null only when reason is not_found."
      },
      {
        name: "license.product",
        type: "string | null",
        description: "The slug's product. Compare it with a value hard-coded in your app to reject keys sold for a different product."
      },
      {
        name: "license.status",
        type: '"inactive" | "active" | "revoked" | "expired"',
        description: "Display status. expired is derived from expires_at."
      },
      { name: "license.expires_at", type: "string | null", description: "RFC3339 expiry, or null when the license never expires." },
      { name: "license.seats.used", type: "number", description: "Devices currently activated on this license." },
      { name: "license.seats.max", type: "number", description: "Maximum number of devices." },
      { name: "license.attributes", type: "object", description: "Slug attributes snapshotted onto the license, for unlocking features." },
      {
        name: "token",
        type: "string (optional)",
        description: "Offline JWT, present only when valid is true, the slug has offline tokens enabled, and an active signing key exists. Includes an attributes claim."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing required fields, or oversized fields." },
      { status: 429, reason: "Rate limited." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-licenses-list": {
    key: "management-licenses-list",
    group: "management",
    label: "GET /management/licenses",
    method: "GET",
    path: "/management/licenses",
    title: "List Licenses",
    summary: "List issued licenses with pagination, search, status filtering, and aggregate status counts.",
    details: [
      "Requires management API key auth.",
      "Supports optional query parameters: page, page_size, q, and status.",
      "page defaults to 1; page_size defaults to 10 and is capped at 100.",
      "status may be inactive, active, revoked, or expired.",
      "q searches license key, slug name, and metadata text; max length is 256 characters.",
      "active_seats is derived from active activation rows at request time.",
      "last_validated_at is the latest validation timestamp across this license's activations."
    ],
    curl: `curl -sS "http://localhost:8080/management/licenses?page=1&page_size=10&q=user@example.com&status=active" \
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body; use optional query parameters page, page_size, q, and status)`,
    responseBody: `{
  "licenses": [
    {
      "id": "2f4d7c2c-9e2f-4d7e-932c-b2e5c1d92f01",
      "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
      "slug": "pro-monthly",
      "status": "active",
      "metadata": {
        "email": "user@example.com",
        "external_customer_id": "cus_123"
      },
      "attributes": {
        "plan": "pro",
        "features": ["analytics", "exports"]
      },
      "max_activations": 3,
      "active_seats": 1,
      "offline_enabled": true,
      "product": "desktop-app",
      "offline_token_lifetime_hours": 24,
      "expires_at": "2026-10-20T12:00:00Z",
      "created_at": "2026-04-20T12:00:00Z",
      "activated_at": "2026-04-21T09:30:00Z",
      "last_validated_at": "2026-04-21T10:45:00Z",
      "revoked_at": null
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 1,
    "total_pages": 1
  },
  "counts": {
    "total": 1,
    "active": 1,
    "inactive": 0,
    "revoked": 0,
    "expired": 0
  }
}`,
    responseFields: [
      { name: "licenses", type: "array", description: "Page of issued license records." },
      { name: "licenses[].id", type: "string", description: "Stable internal license id." },
      { name: "licenses[].license_key", type: "string", description: "Issued license key." },
      { name: "licenses[].slug", type: "string", description: "Slug used when the license was generated." },
      { name: "licenses[].product", type: "string | null", description: "The slug's current product." },
      { name: "licenses[].status", type: '"inactive" | "active" | "revoked" | "expired"', description: "Display status. Expired is derived from expires_at for non-revoked licenses." },
      { name: "licenses[].metadata", type: "object", description: "Caller-provided JSON originally sent to POST /generate for this license instance." },
      { name: "licenses[].attributes", type: "object", description: "Slug-defined entitlement JSON snapshotted onto the license at generation time." },
      { name: "licenses[].max_activations", type: "number", description: "Seat limit snapshotted from the slug at generation time." },
      { name: "licenses[].active_seats", type: "number", description: "Current active activation count." },
      { name: "licenses[].offline_enabled", type: "boolean", description: "Whether the license's slug currently allows offline token issuance." },
      { name: "licenses[].offline_token_lifetime_hours", type: "number", description: "Current offline JWT lifetime configured on the license slug, in hours." },
      { name: "licenses[].expires_at", type: "string | null", description: "RFC3339 expiration timestamp, or null when the license does not expire." },
      { name: "licenses[].created_at", type: "string", description: "RFC3339 creation timestamp." },
      { name: "licenses[].activated_at", type: "string | null", description: "RFC3339 timestamp when the license first became active." },
      { name: "licenses[].last_validated_at", type: "string | null", description: "Most recent activation validation timestamp for this license." },
      { name: "licenses[].revoked_at", type: "string | null", description: "RFC3339 timestamp when the license was revoked." },
      { name: "pagination", type: "object", description: "Current page, page_size, total result count, and total_pages." },
      { name: "counts", type: "object", description: "Aggregate counts for total, active, inactive, revoked, and expired licenses." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid page, page_size, search length, or status filter." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-licenses-get": {
    key: "management-licenses-get",
    group: "management",
    label: "GET /management/licenses/{key}",
    method: "GET",
    path: "/management/licenses/{key}",
    title: "Get License",
    summary: "One license with the devices currently holding its seats.",
    details: [
      "Requires management API key auth.",
      "Only active activations are listed; removed or deactivated devices are not."
    ],
    curl: `curl -sS http://localhost:8080/management/licenses/XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "id": "2f4d7c2c-9e2f-4d7e-932c-b2e5c1d92f01",
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "slug": "pro-yearly",
  "status": "active",
  "metadata": { "email": "user@example.com" },
  "attributes": { "plan": "pro" },
  "max_activations": 3,
  "active_seats": 1,
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 24,
  "expires_at": "2027-10-20T12:00:00Z",
  "created_at": "2026-04-20T12:00:00Z",
  "activated_at": "2026-04-21T09:30:00Z",
  "last_validated_at": "2026-04-21T10:45:00Z",
  "revoked_at": null,
  "activations": [
    {
      "id": 42,
      "fingerprint": "sha256:9f86d081884c7d65",
      "metadata": { "name": "Work laptop" },
      "activated_at": "2026-04-21T09:30:00Z",
      "last_validated_at": "2026-04-21T10:45:00Z"
    }
  ]
}`,
    responseFields: [
      { name: "(license fields)", type: "object", description: "Same license fields as GET /management/licenses, plus activations." },
      { name: "activations", type: "array", description: "Devices currently holding a seat, oldest first." },
      { name: "activations[].id", type: "number", description: "Activation id, used to remove the device." },
      { name: "activations[].fingerprint", type: "string", description: "Fingerprint the app sent on activation." },
      { name: "activations[].metadata", type: "object", description: "Metadata the app sent on activation." },
      { name: "activations[].activated_at", type: "string", description: "RFC3339 activation timestamp." },
      { name: "activations[].last_validated_at", type: "string | null", description: "Most recent validation from this device." }
    ],
    errorCodes: [
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "License not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-licenses-update": {
    key: "management-licenses-update",
    group: "management",
    label: "PATCH /management/licenses/{key}",
    method: "PATCH",
    path: "/management/licenses/{key}",
    title: "Update License",
    summary: "Change a single license's seat limit or expiry, or revoke and restore it.",
    details: [
      "Requires management API key auth.",
      "Send any of max_activations, expires_at and revoked. Fields you leave out are unchanged.",
      "expires_at: null makes the license never expire. Use this to extend a license when a customer renews.",
      "Lowering max_activations below the current seat count keeps existing devices working but blocks new activations.",
      "revoked: false restores a revoked license; its existing devices keep their seats.",
      "Emits license.revoked when the update revokes the license, otherwise license.updated."
    ],
    curl: `curl -sS -X PATCH http://localhost:8080/management/licenses/XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "max_activations": 5,
    "expires_at": "2028-10-20T23:59:59Z"
  }'`,
    requestBody: `{
  "max_activations": 5,
  "expires_at": "2028-10-20T23:59:59Z",
  "revoked": false
}`,
    responseBody: `{
  "id": "2f4d7c2c-9e2f-4d7e-932c-b2e5c1d92f01",
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "slug": "pro-yearly",
  "status": "active",
  "metadata": { "email": "user@example.com" },
  "attributes": { "plan": "pro" },
  "max_activations": 3,
  "active_seats": 1,
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 24,
  "expires_at": "2027-10-20T12:00:00Z",
  "created_at": "2026-04-20T12:00:00Z",
  "activated_at": "2026-04-21T09:30:00Z",
  "last_validated_at": "2026-04-21T10:45:00Z",
  "revoked_at": null,
  "activations": [
    {
      "id": 42,
      "fingerprint": "sha256:9f86d081884c7d65",
      "metadata": { "name": "Work laptop" },
      "activated_at": "2026-04-21T09:30:00Z",
      "last_validated_at": "2026-04-21T10:45:00Z"
    }
  ]
}`,
    responseFields: [
      { name: "max_activations", type: "number (request, optional)", description: "New seat limit. Must be at least 1." },
      { name: "expires_at", type: "string | null (request, optional)", description: "RFC3339 expiry, or null for never." },
      { name: "revoked", type: "boolean (request, optional)", description: "true revokes, false restores." },
      { name: "(response)", type: "object", description: "The updated license, same shape as GET /management/licenses/{key}." }
    ],
    errorCodes: [
      { status: 400, reason: "No fields sent, max_activations below 1, or expires_at not RFC3339/null." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "License not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-licenses-remove-activation": {
    key: "management-licenses-remove-activation",
    group: "management",
    label: "DELETE /management/licenses/{key}/activations/{id}",
    method: "DELETE",
    path: "/management/licenses/{key}/activations/{id}",
    title: "Remove Device",
    summary: "Free a seat on a customer's behalf, for example when their old laptop is gone.",
    details: [
      "Requires management API key auth.",
      "The removed device gets reason fingerprint_not_active on its next validate and can activate again if a seat is free.",
      "Emits license.deactivated with reason removed_by_admin."
    ],
    curl: `curl -sS -X DELETE http://localhost:8080/management/licenses/XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX/activations/42 \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `(the updated license, same shape as GET /management/licenses/{key})`,
    responseFields: [
      { name: "(response)", type: "object", description: "The license after the seat was freed, including the remaining activations." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid activation id." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "License or active activation not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-slugs-list": {
    key: "management-slugs-list",
    group: "management",
    label: "GET /management/slugs",
    method: "GET",
    path: "/management/slugs",
    title: "List Slugs",
    summary: "List configured slug policy templates used during license generation.",
    details: [
      "Requires management API key auth.",
      "Returns slug records including expiration policy, offline policy, and default slug flag.",
      "Returns slug attributes used for license-information snapshots.",
      "Slug policies affect newly generated licenses and are snapshotted at issuance time."
    ],
    curl: `curl -sS http://localhost:8080/management/slugs \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "slugs": [
    {
      "id": 1,
      "name": "default",
      "max_activations": 1,
      "expiration_type": "forever",
      "expiration_days": null,
      "fixed_expires_at": null,
      "offline_enabled": false,
      "product": "desktop-app",
      "offline_token_lifetime_hours": 24,
      "attributes": {
        "plan": "starter",
        "features": []
      },
      "is_default": true,
      "created_at": "2026-04-21T09:00:00Z",
      "updated_at": "2026-04-21T09:00:00Z"
    }
  ]
}`,
    responseFields: [
      { name: "slugs", type: "array", description: "Slug policy records available for generation." },
      { name: "slugs[].offline_enabled", type: "boolean", description: "Whether licenses under this slug can receive offline JWTs." },
      { name: "slugs[].offline_token_lifetime_hours", type: "number", description: "Maximum lifetime for issued/refreshed offline JWTs, in hours." },
      { name: "slugs[].product", type: "string | null", description: "Which app this slug's licenses are for. Returned to apps as license.product and written to offline tokens as aud. Empty clears it." },
      { name: "slugs[].attributes", type: "object", description: "JSON object copied to generated licenses and exported as the JWT attributes claim." }
    ],
    errorCodes: [
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-slugs-create": {
    key: "management-slugs-create",
    group: "management",
    label: "POST /management/slugs",
    method: "POST",
    path: "/management/slugs",
    title: "Create Slug",
    summary: "Create a new slug policy template for future license generation.",
    details: [
      "Requires management API key auth.",
      "Slug names must be lowercase URL-safe (a-z, 0-9, hyphen).",
      "Expiration policy must be one of forever, duration, or fixed_date.",
      "offline_enabled defaults to false unless explicitly set.",
      "offline_token_lifetime_hours defaults to 24 unless explicitly set.",
      "attributes is an optional JSON object used for license information snapshots; defaults to {}."
    ],
    curl: `curl -sS http://localhost:8080/management/slugs \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "pro-monthly",
    "max_activations": 3,
    "expiration_type": "duration",
    "expiration_days": 30,
    "offline_enabled": true,
    "product": "desktop-app",
    "offline_token_lifetime_hours": 24,
    "attributes": {
      "plan": "pro",
      "features": ["analytics", "exports"]
    }
  }'`,
    requestBody: `{
  "name": "pro-monthly",
  "max_activations": 3,
  "expiration_type": "duration",
  "expiration_days": 30,
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 24,
  "attributes": {
    "plan": "pro",
    "features": ["analytics", "exports"]
  }
}`,
    responseBody: `{
  "id": 2,
  "name": "pro-monthly",
  "max_activations": 3,
  "expiration_type": "duration",
  "expiration_days": 30,
  "fixed_expires_at": null,
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 24,
  "attributes": {
    "plan": "pro",
    "features": ["analytics", "exports"]
  },
  "is_default": false,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:00:00Z"
}`,
    responseFields: [
      { name: "id", type: "number", description: "Stable internal slug id." },
      { name: "name", type: "string", description: "Lowercase URL-safe slug key used by /generate." },
      { name: "max_activations", type: "number", description: "Per-license seat limit for newly generated licenses." },
      { name: "expiration_type", type: '"forever" | "duration" | "fixed_date"', description: "Expiration policy mode." },
      { name: "expiration_days", type: "number | null", description: "Required when expiration_type is duration." },
      { name: "fixed_expires_at", type: "string | null", description: "RFC3339 timestamp required when expiration_type is fixed_date." },
      { name: "offline_enabled", type: "boolean", description: "Whether licenses under this slug can receive offline JWTs." },
      { name: "offline_token_lifetime_hours", type: "number", description: "Maximum lifetime for issued/refreshed offline JWTs, in hours." },
      { name: "product", type: "string | null", description: "Which app this slug's licenses are for. Returned to apps as license.product and written to offline tokens as aud. Empty clears it." },
      { name: "attributes", type: "object", description: "JSON object snapshotted to generated licenses and embedded in offline JWT attributes claim." },
      { name: "is_default", type: "boolean", description: "True only for the default slug." },
      { name: "created_at", type: "string", description: "RFC3339 creation timestamp." },
      { name: "updated_at", type: "string", description: "RFC3339 update timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, invalid slug name, invalid expiration policy fields, or invalid attributes shape." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 409, reason: "Slug name already exists." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-slugs-get": {
    key: "management-slugs-get",
    group: "management",
    label: "GET /management/slugs/{name}",
    method: "GET",
    path: "/management/slugs/{name}",
    title: "Get Slug",
    summary: "Fetch one slug policy template by slug name.",
    details: [
      "Requires management API key auth.",
      "Path slug name must be lowercase URL-safe.",
      "Useful for edit forms and operator diagnostics."
    ],
    curl: `curl -sS http://localhost:8080/management/slugs/pro-monthly \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "id": 2,
  "name": "pro-monthly",
  "max_activations": 3,
  "expiration_type": "duration",
  "expiration_days": 30,
  "fixed_expires_at": null,
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 24,
  "attributes": {
    "plan": "pro",
    "features": ["analytics", "exports"]
  },
  "is_default": false,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:00:00Z"
}`,
    responseFields: [
      { name: "name", type: "string", description: "Slug key used by generation calls." },
      { name: "max_activations", type: "number", description: "Seat limit configured on this slug template." },
      { name: "expiration_type", type: '"forever" | "duration" | "fixed_date"', description: "Expiration policy mode." },
      { name: "offline_enabled", type: "boolean", description: "Whether this slug issues offline JWTs on valid activation/validation." },
      { name: "offline_token_lifetime_hours", type: "number", description: "Maximum lifetime for issued/refreshed offline JWTs, in hours." },
      { name: "product", type: "string | null", description: "Which app this slug's licenses are for. Returned to apps as license.product and written to offline tokens as aud. Empty clears it." },
      { name: "attributes", type: "object", description: "JSON object snapshotted to generated licenses and embedded in offline JWT attributes claim." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid slug name format in path." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Slug not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-slugs-update": {
    key: "management-slugs-update",
    group: "management",
    label: "PATCH /management/slugs/{name}",
    method: "PATCH",
    path: "/management/slugs/{name}",
    title: "Update Slug",
    summary: "Update one or more slug policy fields by slug name.",
    details: [
      "Requires management API key auth.",
      "At least one field must be provided.",
      "attributes can be replaced with a new JSON object; set attributes to {} to clear values.",
      "Changing a slug only affects newly generated licenses. Existing licenses keep snapshotted policy values."
    ],
    curl: `curl -sS -X PATCH http://localhost:8080/management/slugs/pro-monthly \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "max_activations": 5,
    "expiration_type": "fixed_date",
    "fixed_expires_at": "2027-01-01T00:00:00Z",
    "offline_enabled": true,
    "product": "desktop-app",
    "offline_token_lifetime_hours": 48,
    "attributes": {
      "plan": "enterprise",
      "features": ["analytics", "sso", "audit"]
    }
  }'`,
    requestBody: `{
  "max_activations": 5,
  "expiration_type": "fixed_date",
  "fixed_expires_at": "2027-01-01T00:00:00Z",
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 48,
  "attributes": {
    "plan": "enterprise",
    "features": ["analytics", "sso", "audit"]
  }
}`,
    responseBody: `{
  "id": 2,
  "name": "pro-monthly",
  "max_activations": 5,
  "expiration_type": "fixed_date",
  "expiration_days": null,
  "fixed_expires_at": "2027-01-01T00:00:00Z",
  "offline_enabled": true,
  "product": "desktop-app",
  "offline_token_lifetime_hours": 48,
  "attributes": {
    "plan": "enterprise",
    "features": ["analytics", "sso", "audit"]
  },
  "is_default": false,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:30:00Z"
}`,
    responseFields: [
      { name: "updated_at", type: "string", description: "RFC3339 timestamp after update." },
      { name: "max_activations", type: "number", description: "Updated seat limit for future generations." },
      { name: "expiration_type", type: '"forever" | "duration" | "fixed_date"', description: "Updated expiration policy mode." },
      { name: "offline_enabled", type: "boolean", description: "Updated offline JWT policy for this slug." },
      { name: "offline_token_lifetime_hours", type: "number", description: "Updated offline JWT lifetime for this slug, in hours." },
      { name: "product", type: "string | null", description: "Which app this slug's licenses are for. Returned to apps as license.product and written to offline tokens as aud. Empty clears it." },
      { name: "attributes", type: "object", description: "Updated slug attributes for future license snapshots and offline JWT attributes claims." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid path slug, invalid body, missing patch fields, invalid expiration policy values, or invalid attributes shape." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Slug not found." },
      { status: 409, reason: "Slug name already exists." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-slugs-delete": {
    key: "management-slugs-delete",
    group: "management",
    label: "DELETE /management/slugs/{name}",
    method: "DELETE",
    path: "/management/slugs/{name}",
    title: "Delete Slug",
    summary: "Archive a slug policy template when it is safe to remove.",
    details: [
      "Requires management API key auth.",
      "Default slug cannot be deleted.",
      "Deletion is a soft delete: archived slugs are hidden from active usage and listing.",
      "Slugs with non-revoked, non-expired licenses cannot be deleted."
    ],
    curl: `curl -sS -X DELETE http://localhost:8080/management/slugs/legacy-beta \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "deleted": true
}`,
    responseFields: [
      { name: "deleted", type: "boolean", description: "True when deletion succeeds." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid slug name format in path." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Slug not found." },
      { status: 409, reason: "Slug cannot be deleted (default or has non-revoked, non-expired licenses)." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-api-keys-list": {
    key: "management-api-keys-list",
    group: "management",
    label: "GET /management/api-keys",
    method: "GET",
    path: "/management/api-keys",
    title: "List API Keys",
    summary: "List generated API keys and their active/revoked state.",
    details: [
      "Requires management API key auth.",
      "Plaintext API key values are never returned by this endpoint.",
      "Use this for admin listing and revocation workflows."
    ],
    curl: `curl -sS http://localhost:8080/management/api-keys \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "api_keys": [
    {
      "id": 1,
      "name": "stripe-prod",
      "hint": "a94f",
      "created_at": "2026-04-21T10:00:00Z",
      "revoked_at": null
    }
  ]
}`,
    responseFields: [
      { name: "api_keys", type: "array", description: "List of generated API key metadata records." }
    ],
    errorCodes: [
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-api-keys-create": {
    key: "management-api-keys-create",
    group: "management",
    label: "POST /management/api-keys",
    method: "POST",
    path: "/management/api-keys",
    title: "Create API Key",
    summary: "Create a generated server API key for provisioning routes.",
    details: [
      "Requires management API key auth.",
      "Returns plaintext api_key only at creation time.",
      "Created keys are stored hashed at rest."
    ],
    curl: `curl -sS http://localhost:8080/management/api-keys \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "stripe-prod"
  }'`,
    requestBody: `{
  "name": "stripe-prod"
}`,
    responseBody: `{
  "api_key": "a94f2c8b5e11d67e8a03c91fd78122e3f6acbb16de2202aa0f814cc2f71d10be",
  "id": 3,
  "name": "stripe-prod",
  "created_at": "2026-04-21T10:30:00Z"
}`,
    responseFields: [
      { name: "api_key", type: "string", description: "Generated key value, shown only once." },
      { name: "id", type: "number", description: "Stable internal key id." },
      { name: "name", type: "string", description: "Operator-supplied label." },
      { name: "created_at", type: "string", description: "RFC3339 creation timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON payload." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-api-keys-revoke": {
    key: "management-api-keys-revoke",
    group: "management",
    label: "POST /management/api-keys/{id}/revoke",
    method: "POST",
    path: "/management/api-keys/{id}/revoke",
    title: "Revoke API Key",
    summary: "Revoke a generated API key so it can no longer access provisioning routes.",
    details: [
      "Requires management API key auth.",
      "Path id must be a positive integer.",
      "Revocation is idempotent and keeps the same revoked_at once set."
    ],
    curl: `curl -sS -X POST http://localhost:8080/management/api-keys/3/revoke \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "id": 3,
  "name": "stripe-prod",
  "hint": "a94f",
  "created_at": "2026-04-21T10:30:00Z",
  "revoked_at": "2026-04-21T11:00:00Z"
}`,
    responseFields: [
      { name: "id", type: "number", description: "Revoked key id." },
      { name: "revoked_at", type: "string", description: "RFC3339 timestamp when revocation was recorded." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid key id." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "API key not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-webhooks-list": {
    key: "management-webhooks-list",
    group: "management",
    label: "GET /management/webhooks",
    method: "GET",
    path: "/management/webhooks",
    title: "List Webhook Endpoints",
    summary: "List configured webhook endpoints and subscribed events.",
    details: [
      "Requires management API key auth.",
      "Useful for admin overview and configuration screens.",
      "This endpoint only manages configuration; event delivery infrastructure is separate."
    ],
    curl: `curl -sS http://localhost:8080/management/webhooks \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "webhooks": [
    {
      "id": 1,
      "name": "audit-sync",
      "url": "https://example.com/hooks/license",
      "events": ["license.generated", "license.revoked"],
      "enabled": true,
      "created_at": "2026-04-21T12:00:00Z",
      "updated_at": "2026-04-21T12:00:00Z"
    }
  ]
}`,
    responseFields: [
      { name: "webhooks", type: "array", description: "Configured webhook endpoint records." }
    ],
    errorCodes: [
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-webhooks-deliveries": {
    key: "management-webhooks-deliveries",
    group: "management",
    label: "GET /management/webhooks/deliveries",
    method: "GET",
    path: "/management/webhooks/deliveries",
    title: "List Webhook Deliveries",
    summary: "List recent webhook delivery runs and their retry/delivery status.",
    details: [
      "Requires management API key auth.",
      "Returns the latest webhook delivery rows across configured endpoints.",
      "Use the optional limit query parameter to control result count; default is 25 and max is 100.",
      "Delivery rows are created only for endpoints that were enabled and subscribed when the event was emitted."
    ],
    curl: `curl -sS "http://localhost:8080/management/webhooks/deliveries?limit=25" \
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body; optional query parameter: limit)`,
    responseBody: `{
  "deliveries": [
    {
      "id": 42,
      "endpoint_id": 1,
      "endpoint_name": "audit-sync",
      "endpoint_url": "https://example.com/hooks/license",
      "event_type": "license.generated",
      "status": "delivered",
      "attempts": 1,
      "last_response_status": 200,
      "last_error": null,
      "next_attempt_at": "2026-04-21T12:00:00Z",
      "created_at": "2026-04-21T12:00:00Z",
      "updated_at": "2026-04-21T12:00:01Z",
      "delivered_at": "2026-04-21T12:00:01Z"
    }
  ]
}`,
    responseFields: [
      { name: "deliveries", type: "array", description: "Recent webhook delivery run records." },
      { name: "deliveries[].id", type: "number", description: "Stable delivery id used in webhook headers." },
      { name: "deliveries[].endpoint_id", type: "number", description: "Webhook endpoint id." },
      { name: "deliveries[].endpoint_name", type: "string", description: "Endpoint name at read time." },
      { name: "deliveries[].endpoint_url", type: "string", description: "Endpoint URL at read time." },
      { name: "deliveries[].event_type", type: "string", description: "Webhook event type." },
      { name: "deliveries[].status", type: '"pending" | "sending" | "delivered" | "failed"', description: "Current delivery state." },
      { name: "deliveries[].attempts", type: "number", description: "Number of delivery attempts made." },
      { name: "deliveries[].last_response_status", type: "number | null", description: "Last HTTP status received from the endpoint." },
      { name: "deliveries[].last_error", type: "string | null", description: "Last delivery error, if any." },
      { name: "deliveries[].next_attempt_at", type: "string", description: "RFC3339 timestamp for the next retry when pending." },
      { name: "deliveries[].created_at", type: "string", description: "RFC3339 timestamp when the delivery was queued." },
      { name: "deliveries[].updated_at", type: "string", description: "RFC3339 timestamp when delivery state last changed." },
      { name: "deliveries[].delivered_at", type: "string | null", description: "RFC3339 timestamp when delivery succeeded." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid limit query parameter." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-webhooks-create": {
    key: "management-webhooks-create",
    group: "management",
    label: "POST /management/webhooks",
    method: "POST",
    path: "/management/webhooks",
    title: "Create Webhook Endpoint",
    summary: "Create a webhook endpoint configuration for selected event types.",
    details: [
      "Requires management API key auth.",
      "URL must be absolute and use http or https.",
      "Events must be supported values documented in Webhook Concepts.",
      "Outbound webhook requests include an Idempotency-Key header (sls-webhook-<delivery_id>) for receiver-side dedupe."
    ],
    supportedEvents: webhookSupportedEvents,
    curl: `curl -sS http://localhost:8080/management/webhooks \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "audit-sync",
    "url": "https://example.com/hooks/license",
    "events": ["license.generated", "license.revoked"],
    "enabled": true
  }'`,
    requestBody: `{
  "name": "audit-sync",
  "url": "https://example.com/hooks/license",
  "events": ["license.generated", "license.revoked"],
  "enabled": true
}`,
    responseBody: `{
  "id": 1,
  "name": "audit-sync",
  "url": "https://example.com/hooks/license",
  "events": ["license.generated", "license.revoked"],
  "enabled": true,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:00:00Z"
}`,
    responseFields: [
      { name: "id", type: "number", description: "Webhook endpoint id." },
      { name: "name", type: "string", description: "Operator-friendly endpoint name." },
      { name: "url", type: "string", description: "Destination URL for outbound events." },
      { name: "events", type: "array<string>", description: "Subscribed event list." },
      { name: "enabled", type: "boolean", description: "Whether endpoint receives events." },
      { name: "created_at", type: "string", description: "RFC3339 creation timestamp." },
      { name: "updated_at", type: "string", description: "RFC3339 last update timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, invalid URL, or unsupported event values." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-webhooks-update": {
    key: "management-webhooks-update",
    group: "management",
    label: "PATCH /management/webhooks/{id}",
    method: "PATCH",
    path: "/management/webhooks/{id}",
    title: "Update Webhook Endpoint",
    summary: "Update a webhook endpoint name, URL, events, and/or enabled flag.",
    details: [
      "Requires management API key auth.",
      "At least one field must be provided.",
      "Events and URL validations match webhook create behavior."
    ],
    supportedEvents: webhookSupportedEvents,
    curl: `curl -sS -X PATCH http://localhost:8080/management/webhooks/1 \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "enabled": false,
    "events": ["license.revoked"]
  }'`,
    requestBody: `{
  "enabled": false,
  "events": ["license.revoked"]
}`,
    responseBody: `{
  "id": 1,
  "name": "audit-sync",
  "url": "https://example.com/hooks/license",
  "events": ["license.revoked"],
  "enabled": false,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:30:00Z"
}`,
    responseFields: [
      { name: "updated_at", type: "string", description: "RFC3339 update timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid id, invalid patch body, invalid URL, or unsupported event values." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Webhook endpoint not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  "management-webhooks-delete": {
    key: "management-webhooks-delete",
    group: "management",
    label: "DELETE /management/webhooks/{id}",
    method: "DELETE",
    path: "/management/webhooks/{id}",
    title: "Delete Webhook Endpoint",
    summary: "Delete a webhook endpoint configuration.",
    details: [
      "Requires management API key auth.",
      "Deletion removes endpoint configuration immediately.",
      "Returns a small confirmation payload."
    ],
    curl: `curl -sS -X DELETE http://localhost:8080/management/webhooks/1 \\
  -H "Authorization: Bearer management_key_dev_123456"`,
    requestBody: `{} (no request body)`,
    responseBody: `{
  "deleted": true
}`,
    responseFields: [{ name: "deleted", type: "boolean", description: "True when deletion succeeded." }],
    errorCodes: [
      { status: 400, reason: "Invalid id." },
      { status: 401, reason: "Missing or invalid management API key." },
      { status: 404, reason: "Webhook endpoint not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  }
};

export const apiEndpointOrder = [
  "generate",
  "revoke",
  "activate",
  "validate",
  "deactivate",
  "management-licenses-list",
  "management-licenses-get",
  "management-licenses-update",
  "management-licenses-remove-activation",
  "management-slugs-list",
  "management-slugs-create",
  "management-slugs-get",
  "management-slugs-update",
  "management-slugs-delete",
  "management-api-keys-list",
  "management-api-keys-create",
  "management-api-keys-revoke",
  "management-webhooks-list",
  "management-webhooks-deliveries",
  "management-webhooks-create",
  "management-webhooks-update",
  "management-webhooks-delete",
  "management-offline-signing-keys-list",
  "management-offline-signing-keys-create",
  "management-offline-signing-keys-activate",
  "management-offline-signing-keys-retire",
  "management-offline-public-keys-list"
];

export function endpointDocHref(endpointKey) {
  return `/docs/api/${endpointKey}/`;
}
