import Link from "next/link";
import CodeExample from "../../../components/CodeExample";

export const metadata = {
  title: "Documentation",
  description:
    "Developer documentation for Simple License Server, including quickstart setup, API examples, and licensing flow guidance.",
  alternates: {
    canonical: "/docs/"
  }
};

export default function DocsHomePage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Documentation</p>
      <h1>Simple License Server docs</h1>

      <p>
        Start with Quickstart, confirm auth, then use API Reference for endpoint-level details. The goal is to get you
        to a working license flow with minimal reading.
      </p>

      <div className="docs-list docs-list-compact">
        <Link href="/docs/quickstart/" className="docs-item">
          <h3>Quickstart</h3>
          <p>Run the server and generate your first license.</p>
        </Link>
        <Link href="/docs/authentication/" className="docs-item">
          <h3>Authentication</h3>
          <p>Understand which key type is used for each route.</p>
        </Link>
        <Link href="/docs/api/" className="docs-item">
          <h3>API Reference</h3>
          <p>Browse all core endpoints by method and path.</p>
        </Link>
      </div>

      <h2 style={{ marginTop: "2rem" }}>Try the API now</h2>
      <CodeExample
        language="bash"
        code={`curl -sS http://localhost:8080/generate \\
  -H "Authorization: Bearer <generated_server_api_key>" \\
  -H "Content-Type: application/json" \\
  -d '{"slug":"default","metadata":{"email":"user@example.com"}}'`}
      />
    </section>
  );
}
