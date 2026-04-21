import Link from "next/link";
import ApiTabs from "../../../../components/ApiTabs";

export const metadata = {
  title: "Management API",
  description:
    "Management API endpoint guide for API key lifecycle and webhook endpoint configuration.",
  alternates: {
    canonical: "/docs/management-api/"
  }
};

export default function ManagementApiPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">API Reference</p>
      <h1>Management API</h1>
      <p>
        These endpoints are for operator and internal tooling flows: generated API keys, webhook endpoint
        configuration, and delivery controls.
      </p>
      <p>
        Looking for license lifecycle routes? See <Link href="/docs/api/">License API</Link>.
      </p>

      <div style={{ marginTop: "1rem" }}>
        <ApiTabs activeGroup="management" />
      </div>
    </section>
  );
}
