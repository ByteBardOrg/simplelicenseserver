import Link from "next/link";
import { notFound } from "next/navigation";
import CodeExample from "../../../../../components/CodeExample";
import { apiEndpointGroups, apiEndpointOrder, apiEndpoints } from "../../../../../components/apiReferenceData";

export const dynamicParams = false;

export function generateStaticParams() {
  return apiEndpointOrder.map((endpoint) => ({ endpoint }));
}

export async function generateMetadata({ params }) {
  const { endpoint: endpointKey } = await params;
  const endpoint = apiEndpoints[endpointKey];
  if (!endpoint) {
    return { title: "API Endpoint" };
  }

  return {
    title: `${endpoint.label}`,
    description: `${endpoint.title} endpoint reference with curl example, request and response schema, and error code behavior.`,
    alternates: {
      canonical: `/docs/api/${endpoint.key}/`
    }
  };
}

export default async function EndpointPage({ params }) {
  const { endpoint: endpointKey } = await params;
  const endpoint = apiEndpoints[endpointKey];
  if (!endpoint) {
    notFound();
  }

  const group = apiEndpointGroups[endpoint.group];

  return (
    <section className="docs-content">
      <p className="eyebrow">API Endpoint</p>
      <h1>{endpoint.label}</h1>
      <p>{endpoint.summary}</p>
      <p>
        <span className="api-endpoint-method">{endpoint.method}</span>{" "}
        <code className="api-endpoint-path">{endpoint.path}</code>
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Endpoint context</h3>
        <p>{group.description}</p>
        <ul>
          {endpoint.details.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      </div>

      <CodeExample language="bash" code={endpoint.curl} />

      <div className="grid" style={{ marginTop: "1rem" }}>
        <article className="card">
          <h3>Request body</h3>
          <CodeExample language="json" code={endpoint.requestBody} />
        </article>
        <article className="card">
          <h3>Response body</h3>
          <CodeExample language="json" code={endpoint.responseBody} />
        </article>
      </div>

      <article className="card" style={{ marginTop: "1rem" }}>
        <h3>Error codes</h3>
        <div className="api-errors-list">
          {endpoint.errorCodes.map((error) => (
            <div className="api-error-item" key={`${error.status}-${error.reason}`}>
              <span className="api-error-status">{error.status}</span>
              <p>{error.reason}</p>
            </div>
          ))}
        </div>
      </article>

      <p style={{ marginTop: "1rem" }}>
        <Link className="button button-secondary" href="/docs/api/">
          Back to API overview
        </Link>
      </p>
    </section>
  );
}
