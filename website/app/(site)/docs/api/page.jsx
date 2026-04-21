import Link from "next/link";
import ApiTabs from "../../../../components/ApiTabs";

export const metadata = {
  title: "License API",
  description:
    "License API endpoint guide for generation, revocation, activation, validation, and deactivation.",
  alternates: {
    canonical: "/docs/api/"
  }
};

export default function ApiPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">API Reference</p>
      <h1>License API</h1>
      <p>
        These endpoints cover license issuance and runtime lifecycle checks. Use this section for generate, revoke,
        activate, validate, and deactivate behavior.
      </p>
      <p>
        Need operator routes instead? See the dedicated{" "}
        <Link href="/docs/management-api/">Management API</Link> page.
      </p>

      <div style={{ marginTop: "1rem" }}>
        <ApiTabs activeGroup="license" />
      </div>
    </section>
  );
}
