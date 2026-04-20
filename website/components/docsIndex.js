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
    href: "/docs/api/",
    title: "API Endpoints",
    summary: "Provisioning and runtime endpoints with examples.",
    icon: "bx-code-alt",
    keywords: ["api", "activate", "validate", "revoke", "deactivate", "generate"]
  },
  {
    href: "/docs/api/generate/",
    title: "POST /generate",
    summary: "Generate a license from a trusted backend system.",
    icon: "bx-plus-circle",
    keywords: ["generate", "provisioning", "idempotency"]
  },
  {
    href: "/docs/api/revoke/",
    title: "POST /revoke",
    summary: "Revoke an existing license key.",
    icon: "bx-block",
    keywords: ["revoke", "provisioning"]
  },
  {
    href: "/docs/api/activate/",
    title: "POST /activate",
    summary: "Activate a license for a fingerprint.",
    icon: "bx-plug",
    keywords: ["activate", "runtime", "fingerprint"]
  },
  {
    href: "/docs/api/validate/",
    title: "POST /validate",
    summary: "Validate runtime license state.",
    icon: "bx-check-shield",
    keywords: ["validate", "runtime", "status"]
  },
  {
    href: "/docs/api/deactivate/",
    title: "POST /deactivate",
    summary: "Release a seat by deactivating one fingerprint.",
    icon: "bx-log-out-circle",
    keywords: ["deactivate", "runtime", "seats"]
  }
];
