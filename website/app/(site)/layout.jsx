import Link from "next/link";
import AnalyticsScript from "../../components/AnalyticsScript";
import TopHeader from "../../components/TopHeader";

export default function SiteLayout({ children }) {
  return (
    <div className="shell">
      <AnalyticsScript />
      <TopHeader />

      <main className="main-content">{children}</main>

      <footer className="footer">
        <div className="container footer-inner">
          <div>
            <div>Simple License Server</div>
            <small>© 2026 ByteBard (CVR 38774468)</small>
          </div>
          <nav className="footer-links" aria-label="Legal">
            <Link href="/privacy/">Privacy</Link>
            <Link href="/terms/">Terms</Link>
          </nav>
        </div>
      </footer>
    </div>
  );
}
