const webhookSupportedEvents = [
  "license.generated",
  "license.activated",
  "license.deactivated",
  "license.validated",
  "license.validation_failed",
  "license.revoked"
];

export const apiEndpointGroups = {
  license: {
    key: "license",
    title: "License API",
    summary:
      "License lifecycle routes for generation, revocation, activation, validation, and seat release.",
    description:
      "Use these endpoints for license lifecycle operations. /generate and /revoke require a generated active server API key. Runtime endpoints use license_key plus fingerprint.",
    endpoints: ["generate", "revoke", "activate", "validate", "deactivate"]
  },
  management: {
    key: "management",
    title: "Management API",
    summary:
      "Operator-facing routes for slug policy management, API key lifecycle, and webhook endpoint configuration.",
    description:
      "Use these endpoints from internal tooling or your frontend backend to manage slug policy templates, provisioning API keys, and webhook endpoint configuration. Management routes require the management bootstrap key configured via MANAGEMENT_API_KEYS or MANAGEMENT_API_KEY.",
    endpoints: [
      "management-slugs-list",
      "management-slugs-create",
      "management-slugs-get",
      "management-slugs-update",
      "management-slugs-delete",
      "management-api-keys-list",
      "management-api-keys-create",
      "management-api-keys-revoke",
      "management-webhooks-list",
      "management-webhooks-create",
      "management-webhooks-update",
      "management-webhooks-delete"
    ]
  }
};

export const apiEndpoints = {
  generate: {
    key: "generate",
    group: "license",
    label: "POST /generate",
    method: "POST",
    path: "/generate",
    title: "Generate License",
    summary: "Create a new license for a slug. Intended for trusted backend systems after payment success.",
    details: [
      "Requires a generated active server API key.",
      "Supports Idempotency-Key for safe webhook retries.",
      "Resolves slug expiration policy at generation time and stores expires_at on the license."
    ],
    curl: `curl -X POST http://localhost:8080/generate \\
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
  "expires_at": "2026-10-20T12:00:00Z",
  "created_at": "2026-04-20T12:00:00Z"
}`,
    responseFields: [
      { name: "license_key", type: "string", description: "Generated license key." },
      { name: "slug", type: "string", description: "Slug used at generation time." },
      { name: "status", type: '"inactive" | "active" | "revoked"', description: "Current license status." },
      { name: "metadata", type: "object", description: "Arbitrary metadata stored with the license." },
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
    label: "POST /revoke",
    method: "POST",
    path: "/revoke",
    title: "Revoke License",
    summary: "Revoke a license key. Revoked licenses cannot be activated or validated.",
    details: [
      "Requires a generated active server API key.",
      "Revocation is license-level, not seat-level.",
      "After revocation, runtime checks return invalid/revoked state."
    ],
    curl: `curl -X POST http://localhost:8080/revoke \\
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
    label: "POST /activate",
    method: "POST",
    path: "/activate",
    title: "Activate License",
    summary: "Bind a license key to a fingerprint. Usually the first runtime call after key entry.",
    details: [
      "Checks revoked/expired state and activation limits.",
      "Creates or confirms an activation for the fingerprint.",
      "Returns a non-valid status with reason when activation is denied."
    ],
    curl: `curl -X POST http://localhost:8080/activate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "metadata": {
      "email": "user@example.com",
      "platform": "macos"
    }
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "fingerprint": "abc123-machine-fingerprint",
  "metadata": {
    "email": "user@example.com",
    "platform": "macos"
  }
}`,
    responseBody: `{
  "valid": true,
  "status": "active",
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "fingerprint": "abc123-machine-fingerprint",
  "expires_at": "2026-10-20T12:00:00Z",
  "reason": ""
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "Whether activation succeeded for this fingerprint." },
      {
        name: "status",
        type: '"inactive" | "active" | "revoked"',
        description: "License status returned by activation logic."
      },
      { name: "license_key", type: "string", description: "License key provided in the request." },
      { name: "fingerprint", type: "string", description: "Fingerprint evaluated for activation." },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null when it does not expire."
      },
      {
        name: "reason",
        type: "string (optional)",
        description: "Present on denied activations (for example revoked, expired, activation_limit_reached)."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, missing required fields, or invalid metadata." },
      { status: 404, reason: "License not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  validate: {
    key: "validate",
    group: "license",
    label: "POST /validate",
    method: "POST",
    path: "/validate",
    title: "Validate License",
    summary: "High-frequency runtime check to confirm license status for a fingerprint.",
    details: [
      "Verifies license state and fingerprint binding.",
      "Returns status and expires_at for runtime decisions.",
      "Designed for repeated online validation heartbeats."
    ],
    curl: `curl -X POST http://localhost:8080/validate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint"
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
  "fingerprint": "abc123-machine-fingerprint"
}`,
    responseBody: `{
  "valid": true,
  "status": "active",
  "expires_at": "2026-10-20T12:00:00Z",
  "reason": ""
}`,
    responseFields: [
      { name: "valid", type: "boolean", description: "Whether the license is valid for this fingerprint." },
      {
        name: "status",
        type: '"inactive" | "active" | "revoked"',
        description: "License status returned by validation logic."
      },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null for non-expiring licenses."
      },
      {
        name: "reason",
        type: "string (optional)",
        description: "Present on invalid responses (for example revoked, expired, fingerprint_not_active)."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON or missing required fields." },
      { status: 404, reason: "License not found." },
      { status: 500, reason: "Unexpected server or storage error." }
    ]
  },
  deactivate: {
    key: "deactivate",
    group: "license",
    label: "POST /deactivate",
    method: "POST",
    path: "/deactivate",
    title: "Deactivate Seat",
    summary: "Release one fingerprint binding so a seat can be reused on another device.",
    details: [
      "Useful for reinstalls, hardware replacement, and seat transfer flows.",
      "Behavior is idempotent by state: repeated calls do not double-release seats.",
      "Idempotency-Key header is not required for this endpoint."
    ],
    curl: `curl -X POST http://localhost:8080/deactivate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
    "fingerprint": "abc123-machine-fingerprint",
    "reason": "user_device_replaced"
  }'`,
    requestBody: `{
  "license_key": "XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX",
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
      { name: "released", type: "boolean", description: "True when a seat was released for this fingerprint." },
      {
        name: "status",
        type: '"inactive" | "active" | "revoked"',
        description: "License status after deactivation logic is applied."
      },
      { name: "active_seats", type: "number", description: "Current active seat count after this request." },
      { name: "max_activations", type: "number", description: "Seat limit from slug policy." },
      {
        name: "expires_at",
        type: "string | null",
        description: "RFC3339 timestamp when the license expires, or null when it does not expire."
      }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON or missing required fields." },
      { status: 404, reason: "License not found." },
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
      "Returns slug records including expiration policy and default slug flag.",
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
      "is_default": true,
      "created_at": "2026-04-21T09:00:00Z",
      "updated_at": "2026-04-21T09:00:00Z"
    }
  ]
}`,
    responseFields: [
      { name: "slugs", type: "array", description: "Slug policy records available for generation." }
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
      "Expiration policy must be one of forever, duration, or fixed_date."
    ],
    curl: `curl -sS http://localhost:8080/management/slugs \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "pro-monthly",
    "max_activations": 3,
    "expiration_type": "duration",
    "expiration_days": 30
  }'`,
    requestBody: `{
  "name": "pro-monthly",
  "max_activations": 3,
  "expiration_type": "duration",
  "expiration_days": 30
}`,
    responseBody: `{
  "id": 2,
  "name": "pro-monthly",
  "max_activations": 3,
  "expiration_type": "duration",
  "expiration_days": 30,
  "fixed_expires_at": null,
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
      { name: "is_default", type: "boolean", description: "True only for the default slug." },
      { name: "created_at", type: "string", description: "RFC3339 creation timestamp." },
      { name: "updated_at", type: "string", description: "RFC3339 update timestamp." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid JSON, invalid slug name, or invalid expiration policy fields." },
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
  "is_default": false,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:00:00Z"
}`,
    responseFields: [
      { name: "name", type: "string", description: "Slug key used by generation calls." },
      { name: "max_activations", type: "number", description: "Seat limit configured on this slug template." },
      { name: "expiration_type", type: '"forever" | "duration" | "fixed_date"', description: "Expiration policy mode." }
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
      "Changing a slug only affects newly generated licenses. Existing licenses keep snapshotted policy values."
    ],
    curl: `curl -sS -X PATCH http://localhost:8080/management/slugs/pro-monthly \\
  -H "Authorization: Bearer management_key_dev_123456" \\
  -H "Content-Type: application/json" \\
  -d '{
    "max_activations": 5,
    "expiration_type": "fixed_date",
    "fixed_expires_at": "2027-01-01T00:00:00Z"
  }'`,
    requestBody: `{
  "max_activations": 5,
  "expiration_type": "fixed_date",
  "fixed_expires_at": "2027-01-01T00:00:00Z"
}`,
    responseBody: `{
  "id": 2,
  "name": "pro-monthly",
  "max_activations": 5,
  "expiration_type": "fixed_date",
  "expiration_days": null,
  "fixed_expires_at": "2027-01-01T00:00:00Z",
  "is_default": false,
  "created_at": "2026-04-21T12:00:00Z",
  "updated_at": "2026-04-21T12:30:00Z"
}`,
    responseFields: [
      { name: "updated_at", type: "string", description: "RFC3339 timestamp after update." },
      { name: "max_activations", type: "number", description: "Updated seat limit for future generations." },
      { name: "expiration_type", type: '"forever" | "duration" | "fixed_date"', description: "Updated expiration policy mode." }
    ],
    errorCodes: [
      { status: 400, reason: "Invalid path slug, invalid body, missing patch fields, or invalid expiration policy values." },
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
    summary: "Delete a slug policy template when it is safe to remove.",
    details: [
      "Requires management API key auth.",
      "Default slug cannot be deleted.",
      "Slugs with existing licenses cannot be deleted."
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
      { status: 409, reason: "Slug cannot be deleted (default or in-use)." },
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
  "management-slugs-list",
  "management-slugs-create",
  "management-slugs-get",
  "management-slugs-update",
  "management-slugs-delete",
  "management-api-keys-list",
  "management-api-keys-create",
  "management-api-keys-revoke",
  "management-webhooks-list",
  "management-webhooks-create",
  "management-webhooks-update",
  "management-webhooks-delete"
];

export function endpointDocHref(endpointKey) {
  return `/docs/api/${endpointKey}/`;
}
