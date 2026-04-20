import CodeExample from "../../../../components/CodeExample";

export const metadata = {
  title: "Quickstart",
  description:
    "Quickstart guide for running Simple License Server with Docker Compose and testing initial license generation.",
  alternates: {
    canonical: "/docs/quickstart/"
  }
};

export default function QuickstartPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Quickstart</p>
      <h1>Run the server in a few minutes</h1>
      <p>
        This path gets you from clone to first generated license key quickly, while keeping production-grade defaults in
        mind.
      </p>

      <div className="card" style={{ marginTop: "1rem" }} id="start-infra">
        <h3>1) Start infrastructure</h3>
        <CodeExample language="bash" code="docker compose up --build" />
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="verify-health">
        <h3>2) Verify health</h3>
        <CodeExample language="bash" code="curl -sS http://localhost:8080/healthz" />
      </div>

      <div className="card" style={{ marginTop: "1rem" }} id="generate-first-key">
        <h3>3) Generate first key</h3>
        <CodeExample
          language="bash"
          code={`curl -sS http://localhost:8080/generate \\
  -H "Authorization: Bearer server_key_dev_123" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: evt_123" \\
  -d '{"slug":"default","metadata":{"email":"user@example.com"}}'`}
        />
      </div>
    </section>
  );
}
