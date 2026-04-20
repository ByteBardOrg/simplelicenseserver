import ApiDocsExplorer from "../../../../components/ApiDocsExplorer";

export const metadata = {
  title: "API Endpoints",
  description:
    "Endpoint guide for Simple License Server covering generate, activate, validate, revoke, and deactivate requests.",
  alternates: {
    canonical: "/docs/api/"
  }
};

export default function ApiPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">API Reference</p>
      <h1>License lifecycle endpoints</h1>
      <p>
        Provisioning endpoints are for trusted backend systems and require a server API key. Runtime endpoints are for
        your shipped app, using license_key plus fingerprint.
      </p>
      <p>
        Pick a group card, then choose an endpoint tab to see practical behavior notes and an exact curl example.
      </p>

      <ApiDocsExplorer />
    </section>
  );
}
