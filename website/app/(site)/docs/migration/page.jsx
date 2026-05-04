import Link from "next/link";

export const metadata = {
  title: "Migration",
  description: "Migration guides for Simple License Server API versions.",
  alternates: {
    canonical: "/docs/migration/"
  }
};

export default function MigrationPage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Migration</p>
      <h1>Upgrade guides</h1>
      <p>Use these guides when moving between published API versions.</p>

      <div className="docs-list" style={{ marginTop: "1rem" }}>
        <Link className="docs-item" href="/docs/migration/0-2-0-to-0-3-0/">
          <div className="eyebrow">Migration</div>
          <h3>0.2.0 -&gt; 0.3.0</h3>
          <p>Bundled management UI, license registry endpoint, and webhook delivery run log.</p>
        </Link>

        <Link className="docs-item" href="/docs/migration/0-1-0-to-0-2-0/">
          <div className="eyebrow">Migration</div>
          <h3>0.1.0 -&gt; 0.2.0</h3>
          <p>Management API introduction, key model split, and docs/navigation updates.</p>
        </Link>
      </div>
    </section>
  );
}
