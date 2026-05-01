import Link from "next/link";
import { apiEndpointGroups, apiEndpoints, endpointDocHref } from "../../../../components/apiReferenceData";

export const metadata = {
  title: "License API",
  description:
    "License API endpoint guide for generation, revocation, activation, validation, and deactivation.",
  alternates: {
    canonical: "/docs/api/"
  }
};

export default function ApiPage() {
  const licenseRows = apiEndpointGroups.license.endpoints.map((endpointKey) => apiEndpoints[endpointKey]);

  const managementRows = [
    apiEndpoints["management-slugs-list"],
    apiEndpoints["management-api-keys-list"],
    apiEndpoints["management-webhooks-list"]
  ];

  return (
    <section className="docs-content">
      <p className="eyebrow">API Reference</p>
      <h1>API Reference</h1>
      <p>
        Simple License Server has a small API surface: license lifecycle routes, management routes, and one health
        endpoint.
      </p>

      <h2 className="docs-reference-group-title">License Management</h2>
      <div className="docs-endpoint-list" role="list" aria-label="License management endpoints">
        {licenseRows.map((endpoint) => (
          <Link key={endpoint.key} href={endpointDocHref(endpoint.key)} className="docs-endpoint-row" role="listitem">
            <span className={`docs-endpoint-method docs-endpoint-method-${endpoint.method.toLowerCase()}`}>
              {endpoint.method}
            </span>
            <code>{endpoint.path}</code>
          </Link>
        ))}
      </div>

      <h2 className="docs-reference-group-title">Server Info</h2>
      <div className="docs-endpoint-list" role="list" aria-label="Server information endpoints">
        <Link href="/docs/quickstart/#verify-health" className="docs-endpoint-row" role="listitem">
          <span className="docs-endpoint-method docs-endpoint-method-get">GET</span>
          <code>/healthz</code>
        </Link>
      </div>

      <h2 className="docs-reference-group-title">Management API</h2>
      <div className="docs-endpoint-list" role="list" aria-label="Management API endpoints">
        {managementRows.map((endpoint) => (
          <Link key={endpoint.key} href={endpointDocHref(endpoint.key)} className="docs-endpoint-row" role="listitem">
            <span className={`docs-endpoint-method docs-endpoint-method-${endpoint.method.toLowerCase()}`}>
              {endpoint.method}
            </span>
            <code>{endpoint.path}</code>
          </Link>
        ))}
      </div>

      <p className="docs-reference-footnote">
        Need the full operator endpoint catalog? Open <Link href="/docs/management-api/">Management API</Link>.
      </p>
    </section>
  );
}
