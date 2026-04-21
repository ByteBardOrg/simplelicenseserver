export const docsIndex = [
  {
    href: "/docs/",
    title: "Overview",
    summary: "Architecture, navigation, and docs map.",
    icon: "bx-home-alt-2",
    keywords: ["overview", "documentation", "intro", "architecture"]
  },
  {
    href: "/docs/quickstart/",
    title: "Quickstart",
    summary: "Run the server, verify health, and generate your first key.",
    icon: "bx-rocket",
    keywords: ["docker", "compose", "setup", "install"]
  },
  {
    href: "/docs/concepts/",
    title: "Concepts",
    summary: "Understand slugs, seats, activation flow, and license lifecycle behavior.",
    icon: "bx-bulb",
    keywords: ["concepts", "slug", "seats", "licenses", "activation", "lifecycle"]
  },
  {
    href: "/docs/concepts/webhooks/",
    title: "Webhook Concepts",
    summary: "Delivery model, idempotency headers, and webhook payload contracts.",
    icon: "bx-broadcast",
    keywords: ["webhooks", "idempotency", "payload", "events", "concepts"]
  },
  {
    href: "/docs/concepts/slugs/",
    title: "Slug Concepts",
    summary: "Slug naming rules, policy fields, and license snapshot behavior at generation time.",
    icon: "bx-label",
    keywords: ["slug", "policies", "max activations", "expiration", "concepts"]
  },
  {
    href: "/docs/configuration/",
    title: "Configuration",
    summary: "All environment variables, defaults, and validation behavior.",
    icon: "bx-slider-alt",
    keywords: ["configuration", "env", "variables", "defaults", "options", "settings"]
  },
  {
    href: "/docs/guides/stripe/",
    title: "Stripe Example",
    summary: "Stripe + Vercel webhook example for automatic license generation.",
    icon: "bx-purchase-tag",
    keywords: ["stripe", "example", "vercel", "webhook", "checkout", "generate"]
  },
  {
    href: "/docs/api/",
    title: "License API",
    summary: "License lifecycle endpoints with request/response examples.",
    icon: "bx-code-alt",
    keywords: ["api", "license api", "generate", "activate", "revoke"]
  },
  {
    href: "/docs/management-api/",
    title: "Management API",
    summary: "Management endpoints for slugs, API keys, and webhook configuration.",
    icon: "bx-wrench",
    keywords: ["management api", "slugs", "api keys", "webhooks", "events"]
  },
  {
    href: "/docs/api/management-slugs-list/",
    title: "GET /management/slugs",
    summary: "List slug policy templates.",
    icon: "bx-list-ul",
    keywords: ["management api", "slugs", "list"]
  },
  {
    href: "/docs/api/management-slugs-create/",
    title: "POST /management/slugs",
    summary: "Create a slug policy template.",
    icon: "bx-plus-circle",
    keywords: ["management api", "slugs", "create"]
  },
  {
    href: "/docs/api/management-slugs-get/",
    title: "GET /management/slugs/{name}",
    summary: "Get a slug policy template by name.",
    icon: "bx-search-alt",
    keywords: ["management api", "slugs", "get"]
  },
  {
    href: "/docs/api/management-slugs-update/",
    title: "PATCH /management/slugs/{name}",
    summary: "Update a slug policy template.",
    icon: "bx-edit",
    keywords: ["management api", "slugs", "update"]
  },
  {
    href: "/docs/api/management-slugs-delete/",
    title: "DELETE /management/slugs/{name}",
    summary: "Delete a slug policy template.",
    icon: "bx-trash",
    keywords: ["management api", "slugs", "delete"]
  },
  {
    href: "/docs/migration/",
    title: "Migration",
    summary: "Upgrade guides across released versions.",
    icon: "bx-transfer",
    keywords: ["migration", "upgrade", "version"]
  },
  {
    href: "/docs/migration/0-1-0-to-0-2-0/",
    title: "0.1.0 -> 0.2.0",
    summary: "Breaking changes and migration checklist for 0.2.0.",
    icon: "bx-git-compare",
    keywords: ["0.2.0", "0.1.0", "migration", "breaking changes"]
  },
  {
    href: "/docs/api/generate/",
    title: "POST /generate",
    summary: "Generate a license using a generated server API key.",
    icon: "bx-plus-circle",
    keywords: ["generate", "license api", "idempotency"]
  },
  {
    href: "/docs/api/revoke/",
    title: "POST /revoke",
    summary: "Revoke an existing license key.",
    icon: "bx-block",
    keywords: ["revoke", "license api"]
  },
  {
    href: "/docs/api/activate/",
    title: "POST /activate",
    summary: "Activate a license for a fingerprint.",
    icon: "bx-plug",
    keywords: ["activate", "license api", "fingerprint"]
  },
  {
    href: "/docs/api/validate/",
    title: "POST /validate",
    summary: "Validate runtime license state.",
    icon: "bx-check-shield",
    keywords: ["validate", "license api", "status"]
  },
  {
    href: "/docs/api/deactivate/",
    title: "POST /deactivate",
    summary: "Release a seat by deactivating one fingerprint.",
    icon: "bx-log-out-circle",
    keywords: ["deactivate", "license api", "seats"]
  },
  {
    href: "/docs/api/management-api-keys-list/",
    title: "GET /management/api-keys",
    summary: "List generated API keys for provisioning.",
    icon: "bx-list-ul",
    keywords: ["management api", "api keys", "list"]
  },
  {
    href: "/docs/api/management-api-keys-create/",
    title: "POST /management/api-keys",
    summary: "Create a generated server API key.",
    icon: "bx-key",
    keywords: ["management api", "api keys", "create"]
  },
  {
    href: "/docs/api/management-api-keys-revoke/",
    title: "POST /management/api-keys/{id}/revoke",
    summary: "Revoke a generated API key.",
    icon: "bx-key-minus",
    keywords: ["management api", "api keys", "revoke"]
  },
  {
    href: "/docs/api/management-webhooks-list/",
    title: "GET /management/webhooks",
    summary: "List configured webhook endpoints.",
    icon: "bx-link-alt",
    keywords: ["management api", "webhooks", "list"]
  },
  {
    href: "/docs/api/management-webhooks-create/",
    title: "POST /management/webhooks",
    summary: "Create a webhook endpoint configuration.",
    icon: "bx-plus-circle",
    keywords: ["management api", "webhooks", "create"]
  },
  {
    href: "/docs/api/management-webhooks-update/",
    title: "PATCH /management/webhooks/{id}",
    summary: "Update webhook endpoint configuration.",
    icon: "bx-edit",
    keywords: ["management api", "webhooks", "update"]
  },
  {
    href: "/docs/api/management-webhooks-delete/",
    title: "DELETE /management/webhooks/{id}",
    summary: "Delete a webhook endpoint.",
    icon: "bx-trash",
    keywords: ["management api", "webhooks", "delete"]
  }
];
