import Link from "next/link";
import DockerPullCard from "../../components/DockerPullCard";
import { codeLanguageClass, highlightCode } from "../../components/codeHighlight";

const homeApiExample = `curl -X POST http://localhost:8080/activate \\
  -H "Content-Type: application/json" \\
  -d '{
    "license_key": "C1B6DE-39A6E3-DE1529-8559A0-4AF593",
    "fingerprint": "machine-uuid-9876",
    "metadata": {
      "os": "linux",
      "version": "1.0"
    }
  }'`;

const homeStructuredData = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": "https://simplelicenseserver.com/#website",
      url: "https://simplelicenseserver.com/",
      name: "Simple License Server",
      description:
        "Simple License Server is a self-hosted license key API you can run in Docker. Generate, activate, validate, revoke, and deactivate software licenses with a simple JSON API.",
      inLanguage: "en-US"
    },
    {
      "@type": "SoftwareApplication",
      "@id": "https://simplelicenseserver.com/#application",
      name: "Simple License Server",
      description:
        "A self-hosted license server for SaaS and desktop developers. Run in Docker and manage software keys with a simple JSON API.",
      applicationCategory: "DeveloperApplication",
      operatingSystem: "Linux, macOS, Windows",
      url: "https://simplelicenseserver.com/"
    }
  ]
};

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
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(homeStructuredData) }} />

      <section className="container hero">
        <h1>
          Effortless License <br />
          <span className="gradient">Management</span>
        </h1>
        <p>
          Simple License Server is a self-hosted license server for SaaS and desktop developers who need a reliable license key API. Run it in Docker, store data in Postgres, and integrate in your own infrastructure.
        </p>
        <DockerPullCard />
      </section>

      <section className="container section" aria-labelledby="features-heading" style={{ marginTop: "5rem" }}>
        <h2 id="features-heading">It's this easy to use.</h2>
        <div className="easy-grid">
          <article className="easy-code-card">
            <div className="easy-code-head">
              <span className="eyebrow">API Example</span>
              <span className="easy-code-endpoint">POST /activate</span>
            </div>
            <pre className="easy-code-body">
              <code
                className={codeLanguageClass("bash")}
                dangerouslySetInnerHTML={{ __html: highlightCode(homeApiExample, "bash") }}
              ></code>
            </pre>
          </article>

          <div className="easy-feature-list">
            <article className="easy-feature-item">
              <span className="easy-feature-icon" aria-hidden="true">
                <i className="bx bx-git-branch"></i>
              </span>
              <div>
                <h3>Made for API workflows</h3>
                <p>
                  Use it behind payment webhooks, admin tools, and internal services with predictable endpoint behavior.
                </p>
              </div>
            </article>

            <article className="easy-feature-item">
              <span className="easy-feature-icon" aria-hidden="true">
                <i className="bx bx-check-shield"></i>
              </span>
              <div>
                <h3>Production-friendly defaults</h3>
                <p>Request timeouts, shutdown control, and API protections are available out of the box.</p>
              </div>
            </article>

            <article className="easy-feature-item">
              <span className="easy-feature-icon" aria-hidden="true">
                <i className="bx bx-server"></i>
              </span>
              <div>
                <h3>Built for self-hosting</h3>
                <p>
                  Your keys and license metadata remain under your control instead of a third-party licensing SaaS.
                </p>
              </div>
            </article>
          </div>
        </div>
      </section>
    </>
  );
}
