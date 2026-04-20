export const apiEndpointGroups = {
  provisioning: {
    key: "provisioning",
    title: "Provisioning API",
    summary:
      "Used by trusted backend systems such as payment webhook handlers, internal admin tooling, and service integrations.",
    description:
      "Used by trusted backend systems such as payment webhook handlers, internal admin tooling, and service integrations.",
    endpoints: ["generate", "revoke"]
  },
  runtime: {
    key: "runtime",
    title: "Client Runtime API",
    summary:
      "Used by your shipped client application after a license has been issued. These routes use license_key plus fingerprint.",
    description:
      "Used by your shipped client application after a license has been issued. These routes use license_key plus fingerprint.",
    endpoints: ["activate", "validate", "deactivate"]
  }
};

export const apiEndpoints = {
  generate: {
    key: "generate",
    group: "provisioning",
    label: "POST /generate",
    method: "POST",
    path: "/generate",
    title: "Generate License",
    summary: "Create a new license for a slug. This is the route to call after successful payment.",
    details: [
      "Requires a server API key in the Authorization header.",
      "Supports Idempotency-Key header for safe webhook retries.",
      "Resolves slug policy at generation time and persists expires_at on the license."
    ],
    curl: `curl -X POST http://localhost:8080/generate \\
  -H "Authorization: Bearer server_key_dev_123" \\
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
  "license_key": "XXXX-XXXX-XXXX-XXXX",
  "slug": "my-product",
  "status": "inactive",
  "metadata": {
    "email": "user@example.com",
    "external_customer_id": "cus_123",
    "external_order_id": "ord_456"
  },
  "expires_at": "2026-10-20T12:00:00Z",
  "created_at": "2026-04-20T12:00:00Z"
}`,
    responseFields: [
      { name: "license_key", type: "string", description: "Generated license key." },
      { name: "slug", type: "string", description: "Product slug used at generation time." },
      { name: "status", type: '"inactive" | "active" | "revoked"', description: "Current license status." },
      { name: "metadata", type: "object", description: "Arbitrary metadata object that was stored with the license." },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null for non-expiring slug policies."
      },
      { name: "created_at", type: "string", description: "RFC3339 creation timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing slug, oversized fields, or invalid metadata structure." },
      { status: 401, reason: "Missing or invalid server API key." },
      { status: 404, reason: "Referenced slug does not exist." },
      { status: 409, reason: "Idempotency key conflict or in-progress idempotent request." },
      { status: 500, reason: "Unexpected storage or server error." }
    ]
  },
  revoke: {
    key: "revoke",
    group: "provisioning",
    label: "POST /revoke",
    method: "POST",
    path: "/revoke",
    title: "Revoke License",
    summary: "Revoke an existing license. Revoked licenses cannot be activated or validated.",
    details: [
      "Requires a server API key in the Authorization header.",
      "Revocation is license-level, not seat-level.",
      "After revocation, runtime checks return invalid/revoked state."
    ],
    curl: `curl -X POST http://localhost:8080/revoke \\
  -H "Authorization: Bearer server_key_dev_123" \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXX-XXXX-XXXX-XXXX"
  }'`,
    requestBody: `{
  "license_key": "XXXX-XXXX-XXXX-XXXX"
}`,
    responseBody: `{
  "valid": false,
  "status": "revoked",
  "license_key": "XXXX-XXXX-XXXX-XXXX",
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
      { status: 401, reason: "Missing or invalid server API key." },
      { status: 404, reason: "License key was not found." },
      { status: 500, reason: "Unexpected storage or server error." }
    ]
  },
  activate: {
    key: "activate",
    group: "runtime",
    label: "POST /activate",
    method: "POST",
    path: "/activate",
    title: "Activate License",
    summary: "Bind a license key to a fingerprint. Usually the first runtime call after key entry.",
    details: [
      "Checks revoked/expired state and activation limits.",
      "Creates or confirms an activation for the fingerprint.",
      "May return a non-valid status with a reason when activation is denied."
    ],
    curl: `curl -X POST http://localhost:8080/activate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "metadata": {
      "email": "user@example.com",
      "platform": "macos"
    }
  }'`,
    requestBody: `{
  "license_key": "XXXX-XXXX-XXXX-XXXX",
  "fingerprint": "abc123-machine-fingerprint",
  "metadata": {
    "email": "user@example.com",
    "platform": "macos"
  }
}`,
    responseBody: `{
  "valid": true,
  "status": "active",
  "license_key": "XXXX-XXXX-XXXX-XXXX",
  "fingerprint": "abc123-machine-fingerprint",
  "expires_at": "2026-10-20T12:00:00Z",
  "reason": ""
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "Whether activation succeeded for this fingerprint." },
      {
        name: "status",
        type: '"inactive" | "active" | "revoked"',
        description: "License status returned by the activation decision."
      },
      { name: "license_key", type: "string", description: "License key provided in the request." },
      { name: "fingerprint", type: "string", description: "Fingerprint evaluated for activation." },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null when the license does not expire."
      },
      {
        name: "reason",
        type: "string (optional)",
        description: "Present on denied activations (for example revoked, expired, or activation_limit_reached)."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing required fields, or invalid metadata." },
      { status: 404, reason: "License key was not found." },
      { status: 500, reason: "Unexpected storage or server error." }
    ]
  },
  validate: {
    key: "validate",
    group: "runtime",
    label: "POST /validate",
    method: "POST",
    path: "/validate",
    title: "Validate License",
    summary: "High-frequency runtime check to confirm license status for a fingerprint.",
    details: [
      "Verifies license state and fingerprint binding.",
      "Returns status and expires_at for runtime decisions.",
      "Can be called often as your online validation heartbeat."
    ],
    curl: `curl -X POST http://localhost:8080/validate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "fingerprint": "abc123-machine-fingerprint"
  }'`,
    requestBody: `{
  "license_key": "XXXX-XXXX-XXXX-XXXX",
  "fingerprint": "abc123-machine-fingerprint"
}`,
    responseBody: `{
  "valid": true,
  "status": "active",
  "expires_at": "2026-10-20T12:00:00Z",
  "reason": ""
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "Whether the license is currently valid for the fingerprint." },
      {
        name: "status",
        type: '"inactive" | "active" | "revoked"',
        description: "License status returned by runtime validation."
      },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null for non-expiring licenses."
      },
      {
        name: "reason",
        type: "string (optional)",
        description: "Present on invalid responses (for example revoked, expired, or fingerprint_not_active)."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON or missing required fields." },
      { status: 404, reason: "License key was not found." },
      { status: 500, reason: "Unexpected storage or server error." }
    ]
  },
  deactivate: {
    key: "deactivate",
    group: "runtime",
    label: "POST /deactivate",
    method: "POST",
    path: "/deactivate",
    title: "Deactivate Seat",
    summary: "Release one fingerprint binding so a seat can be reused on another device.",
    details: [
      "Useful for reinstall, hardware replacement, and seat transfer flows.",
      "Behavior is idempotent by state: repeated calls do not double-release seats.",
      "Header-based Idempotency-Key is not required for this route."
    ],
    curl: `curl -X POST http://localhost:8080/deactivate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "reason": "user_device_replaced"
  }'`,
    requestBody: `{
  "license_key": "XXXX-XXXX-XXXX-XXXX",
  "fingerprint": "abc123-machine-fingerprint",
  "reason": "user_device_replaced"
}`,
    responseBody: `{
  "valid": true,
  "released": true,
  "status": "inactive",
  "active_seats": 0,
  "max_activations": 3,
  "expires_at": "2026-10-20T12:00:00Z"
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "Whether the request completed successfully." },
      { name: "released", type: "boolean", description: "True when a seat was released for the fingerprint." },
      {
        name: "status",
        type: '"inactive" | "active" | "revoked"',
        description: "License status after deactivation logic is applied."
      },
      { name: "active_seats", type: "number", description: "Current number of active seats after this request." },
      { name: "max_activations", type: "number", description: "Seat limit configured by the license slug policy." },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null when the license does not expire."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON or missing required fields." },
      { status: 404, reason: "License key was not found." },
      { status: 500, reason: "Unexpected storage or server error." }
    ]
  }
};

export const apiEndpointOrder = ["generate", "revoke", "activate", "validate", "deactivate"];

export function endpointDocHref(endpointKey) {
  return `/docs/api/${endpointKey}/`;
}
