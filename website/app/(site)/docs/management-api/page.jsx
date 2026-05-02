import Link from "next/link";
import { apiEndpointGroups, apiEndpoints, endpointDocHref } from "../../../../components/apiReferenceData";

export const metadata = {
  title: "Management API",
  description:
    "Management API endpoint guide for slug policy management, API key lifecycle, and webhook endpoint configuration.",
  alternates: {
    canonical: "/docs/management-api/"
  }
};

export default function ManagementApiPage() {
  const managementConceptSections = [
    {
      title: "License Management",
      endpointKeys: ["management-licenses-list"]
    },
    {
      title: "Slug Management",
      endpointKeys: [
        "management-slugs-list",
        "management-slugs-create",
        "management-slugs-get",
        "management-slugs-update",
        "management-slugs-delete"
      ]
    },
    {
      title: "API Key Management",
      endpointKeys: ["management-api-keys-list", "management-api-keys-create", "management-api-keys-revoke"]
    },
    {
      title: "Webhook Management",
      endpointKeys: [
        "management-webhooks-list",
        "management-webhooks-create",
        "management-webhooks-update",
        "management-webhooks-delete"
      ]
    }
  ];

  const enabledManagementEndpoints = new Set(apiEndpointGroups.management.endpoints);

  return (
    <section className="docs-content">
      <p className="eyebrow">API Reference</p>
      <h1>Management API</h1>
      <p>
        These routes are for internal tooling: license inspection, slug policies, generated server API keys, and webhook
        endpoint configuration. License generation and revocation stay on <code>/generate</code> and <code>/revoke</code>.
      </p>

      {managementConceptSections.map((section) => {
        const rows = section.endpointKeys
          .filter((endpointKey) => enabledManagementEndpoints.has(endpointKey))
          .map((endpointKey) => apiEndpoints[endpointKey]);

        if (rows.length === 0) {
          return null;
        }

        return (
          <div key={section.title}>
            <h2 className="docs-reference-group-title">{section.title}</h2>
            <div className="docs-endpoint-list" role="list" aria-label={`${section.title} endpoints`}>
              {rows.map((endpoint) => (
                <Link
                  key={endpoint.key}
                  href={endpointDocHref(endpoint.key)}
                  className="docs-endpoint-row"
                  role="listitem"
                >
                  <span className={`docs-endpoint-method docs-endpoint-method-${endpoint.method.toLowerCase()}`}>
                    {endpoint.method}
                  </span>
                  <code>{endpoint.path}</code>
                </Link>
              ))}
            </div>
          </div>
        );
      })}

      <p className="docs-reference-footnote">
        Looking for runtime/license lifecycle routes? See <Link href="/docs/api/">License API</Link>.
      </p>
    </section>
  );
}
