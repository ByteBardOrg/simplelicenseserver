"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { apiEndpointOrder, apiEndpoints, endpointDocHref } from "./apiReferenceData";

const baseLinks = [
  { href: "/docs/", title: "Overview", icon: "bx-home-alt-2" },
  { href: "/docs/quickstart/", title: "Quickstart", icon: "bx-rocket" },
  { href: "/docs/api/", title: "API Endpoints", icon: "bx-code-alt" }
];

export default function DocsSidebar() {
  const pathname = usePathname();

  const isApiSection = pathname.startsWith("/docs/api/");

  return (
    <aside className="docs-sidebar" aria-label="Documentation sections">
      <div className="docs-sidebar-header">
        <div className="docs-sidebar-icon">
          <i aria-hidden="true" className="bx bx-terminal"></i>
        </div>
        <div>
          <h2>Documentation</h2>
          <p>v1 static docs</p>
        </div>
      </div>

      <nav className="docs-sidebar-nav">
        {baseLinks.map((entry) => {
          const active = entry.href === "/docs/api/" ? isApiSection : pathname === entry.href;

          return (
            <div key={entry.href}>
              <Link href={entry.href} className={`docs-side-link ${active ? "docs-side-link-active" : ""}`}>
                <i aria-hidden="true" className={`bx ${entry.icon}`}></i>
                <span>{entry.title}</span>
              </Link>

              {entry.href === "/docs/api/" ? (
                <div className="docs-side-subnav">
                  {apiEndpointOrder.map((endpointKey) => {
                    const href = endpointDocHref(endpointKey);
                    const endpoint = apiEndpoints[endpointKey];
                    const endpointActive = pathname === href;

                    return (
                      <Link key={endpointKey} href={href} className={`docs-side-sublink ${endpointActive ? "docs-side-sublink-active" : ""}`}>
                        {endpoint.label}
                      </Link>
                    );
                  })}
                </div>
              ) : null}
            </div>
          );
        })}
      </nav>

      <Link className="docs-sidebar-footer" href="/docs/">
        <i aria-hidden="true" className="bx bx-book-open"></i>
        Documentation Home
      </Link>
    </aside>
  );
}
