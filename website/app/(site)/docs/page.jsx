import Link from "next/link";
import CodeExample from "../../../components/CodeExample";
import { docsIndex } from "../../../components/docsIndex";

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
      <h1>Build your license flow with confidence</h1>

      <div className="docs-list">
        {docsIndex.map((entry) => (
          <Link href={entry.href} className="docs-item" key={entry.href}>
            <div className="eyebrow">Documentation</div>
            <h3>{entry.title}</h3>
            <p>{entry.summary}</p>
          </Link>
        ))}
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
