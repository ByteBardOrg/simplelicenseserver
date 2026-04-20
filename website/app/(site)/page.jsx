import Link from "next/link";
import DockerPullCard from "../../components/DockerPullCard";

export const metadata = {
  title: "Simple License Server | Self-Hosted License API for Developers",
  description:
    "A self-hosted license server for SaaS and desktop developers. Run in Docker and manage software keys with a simple JSON API.",
  alternates: {
    canonical: "/"
  }
};

export default function HomePage() {
  return (
    <>
      <section className="container hero">
        <h1>
          Effortless License <br />
          <span className="gradient">Management</span>
        </h1>
        <p>
          A dead-simple licensing backend that lives in your infrastructure. Issue keys, manage fingerprints, and
          validate licenses through a universal JSON API designed for production use.
        </p>
        <DockerPullCard />
      </section>

      <section className="container section" aria-labelledby="features-heading">
        <h2 id="features-heading">It's this easy to use.</h2>
        <div className="easy-grid">
          <article className="easy-code-card">
            <div className="easy-code-head">
              <span className="eyebrow">API Example</span>
              <span className="easy-code-endpoint">POST /activate</span>
            </div>
            <pre className="easy-code-body">{`curl -X POST http://localhost:8080/activate \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <your_api_key>" \\
  -d '{
    "license_key": "ABCD-1234-EFGH-5678",
    "fingerprint": "machine-uuid-9876",
    "metadata": {
      "os": "linux",
      "version": "1.0"
    }
  }'`}</pre>
          </article>

          <div className="easy-feature-list">
            <article className="easy-feature-item">
              <span className="easy-feature-icon" aria-hidden="true">
                <i className="bx bxl-docker"></i>
              </span>
              <div>
                <h3>Simple to Deploy</h3>
                <p>
                  Packaged as a single Docker container. Zero complex dependencies, external databases, or heavy
                  configurations required.
                </p>
              </div>
            </article>

            <article className="easy-feature-item">
              <span className="easy-feature-icon" aria-hidden="true">
                <i className="bx bx-code-alt"></i>
              </span>
              <div>
                <h3>Universal API</h3>
                <p>Integrate with any language or framework using standard REST endpoints and JSON payloads.</p>
              </div>
            </article>

            <article className="easy-feature-item">
              <span className="easy-feature-icon" aria-hidden="true">
                <i className="bx bx-tachometer"></i>
              </span>
              <div>
                <h3>Built for Performance</h3>
                <p>
                  Written in Go. Validates thousands of licenses per second with minimal memory footprint and
                  sub-millisecond latency.
                </p>
              </div>
            </article>
          </div>
        </div>
      </section>

      <section className="container section" aria-labelledby="seo-content">
        <h2 id="seo-content">Self-hosted software licensing, without the bloat</h2>
        <p className="section-intro">
          Simple License Server is a self-hosted license server for SaaS and desktop developers who need a reliable
          license key API. Run it in Docker, store data in Postgres, and integrate generation, activation, validation,
          revocation, and deactivation in your own infrastructure.
        </p>

        <div className="grid">
          <article className="card">
            <h3>Made for API workflows</h3>
            <p>Use it behind payment webhooks, admin tools, and internal services with predictable endpoint behavior.</p>
          </article>
          <article className="card">
            <h3>Production-friendly defaults</h3>
            <p>Request timeouts, shutdown control, and API protections are available out of the box.</p>
          </article>
          <article className="card">
            <h3>Built for self-hosting</h3>
            <p>Your keys and license metadata remain under your control instead of a third-party licensing SaaS.</p>
          </article>
        </div>
      </section>

      <section className="container section" aria-labelledby="faq-heading">
        <h2 id="faq-heading">Frequently asked questions</h2>
        <div className="grid">
          <article className="card">
            <h3>Can I run this with Docker Compose?</h3>
            <p>Yes. The repo includes a compose setup for API and Postgres, making local and server deploys quick.</p>
          </article>
          <article className="card">
            <h3>Does it support custom activation flows?</h3>
            <p>Yes. Activation and validation are fingerprint-aware, so seat and device policies are easy to extend.</p>
          </article>
          <article className="card">
            <h3>Is this a hosted licensing service?</h3>
            <p>No. The project is self-hosted first. You own uptime, storage, and integration logic.</p>
          </article>
          <article className="card">
            <h3>Where are the docs?</h3>
            <p>Visit the docs section for quickstart and API guides, plus implementation examples.</p>
          </article>
        </div>
        <p style={{ marginTop: "1rem" }}>
          <Link href="/docs/" className="button button-secondary">
            Open Documentation
          </Link>
        </p>
      </section>
    </>
  );
}
