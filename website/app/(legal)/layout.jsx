import Link from "next/link";
import TopHeader from "../../components/TopHeader";

export default function LegalLayout({ children }) {
  return (
    <div className="shell">
      <TopHeader />
      <main className="main-content">{children}</main>
      <footer className="footer">
        <div className="container footer-inner">
          <div>© 2026 ByteBard (CVR 38774468)</div>
          <Link href="/">Back to homepage</Link>
        </div>
      </footer>
    </div>
  );
}
