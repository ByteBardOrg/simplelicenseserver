import Link from "next/link";
import DocsSearchBar from "./DocsSearchBar";

export default function TopHeader() {
  return (
    <header className="top-nav">
      <div className="container nav-inner">
        <div className="nav-main">
          <Link href="/" className="brand">
            Simple License Server
          </Link>

          <nav className="nav-links" aria-label="Main">
            <Link className="nav-link" href="/docs/">
              Documentation
            </Link>
            <Link className="nav-link" href="/docs/api/">
              API
            </Link>
          </nav>
        </div>

        <div className="nav-actions">
          <DocsSearchBar />
          <Link className="button" href="/docs/quickstart/">
            Get Started
          </Link>
        </div>
      </div>
    </header>
  );
}
