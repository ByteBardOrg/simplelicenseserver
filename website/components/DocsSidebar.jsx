"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { apiEndpointGroups, apiEndpoints, endpointDocHref } from "./apiReferenceData";

const baseLinks = [
  { href: "/docs/", title: "Overview", icon: "bx-home-alt-2" },
  { href: "/docs/quickstart/", title: "Quickstart", icon: "bx-rocket" },
  { href: "/docs/concepts/", title: "Concepts", icon: "bx-bulb" },
  { href: "/docs/configuration/", title: "Configuration", icon: "bx-slider-alt" },
  { href: "/docs/guides/stripe/", title: "Stripe Example", icon: "bx-purchase-tag" },
  { href: "/docs/api/", title: "License API", icon: "bx-code-alt" },
  { href: "/docs/management-api/", title: "Management API", icon: "bx-wrench" },
  { href: "/docs/migration/", title: "Migration", icon: "bx-transfer" }
];

export default function DocsSidebar() {
  const pathname = usePathname();
  const licenseEndpointKeys = apiEndpointGroups.license.endpoints;
  const managementEndpointKeys = apiEndpointGroups.management.endpoints;

  const isLicenseEndpointPath = licenseEndpointKeys.some((endpointKey) => pathname === endpointDocHref(endpointKey));
  const isManagementEndpointPath = managementEndpointKeys.some((endpointKey) => pathname === endpointDocHref(endpointKey));

  const isLicenseApiSection = pathname === "/docs/api/" || isLicenseEndpointPath;
  const isManagementApiSection = pathname === "/docs/management-api/" || isManagementEndpointPath;
  const isMigrationSection = pathname === "/docs/migration/" || pathname === "/docs/migration/0-1-0-to-0-2-0/";
  const isConceptsSection = pathname === "/docs/concepts/" || pathname === "/docs/concepts/webhooks/";

  return (
    <aside className="docs-sidebar" aria-label="Documentation sections">
      <div className="docs-sidebar-header">
        <div className="docs-sidebar-icon">
          <i aria-hidden="true" className="bx bx-terminal"></i>
        </div>
        <div>
          <h2>Documentation</h2>
        </div>
      </div>

      <nav className="docs-sidebar-nav">
        {baseLinks.map((entry) => {
          let active = pathname === entry.href;
          if (entry.href === "/docs/api/") {
            active = isLicenseApiSection;
          }
          if (entry.href === "/docs/management-api/") {
            active = isManagementApiSection;
          }
          if (entry.href === "/docs/concepts/") {
            active = isConceptsSection;
          }
          if (entry.href === "/docs/migration/") {
            active = isMigrationSection;
          }

          return (
            <div key={entry.href}>
              <Link href={entry.href} className={`docs-side-link ${active ? "docs-side-link-active" : ""}`}>
                <i aria-hidden="true" className={`bx ${entry.icon}`}></i>
                <span>{entry.title}</span>
              </Link>

              {entry.href === "/docs/api/" ? (
                <div className="docs-side-subnav">
                  {licenseEndpointKeys.map((endpointKey) => {
                    const href = endpointDocHref(endpointKey);
                    const endpoint = apiEndpoints[endpointKey];
                    const endpointActive = pathname === href;

                    return (
                      <Link
                        key={endpointKey}
                        href={href}
                        className={`docs-side-sublink ${endpointActive ? "docs-side-sublink-active" : ""}`}
                      >
                        {endpoint.label}
                      </Link>
                    );
                  })}
                </div>
              ) : null}

              {entry.href === "/docs/management-api/" ? (
                <div className="docs-side-subnav">
                  {managementEndpointKeys.map((endpointKey) => {
                    const href = endpointDocHref(endpointKey);
                    const endpoint = apiEndpoints[endpointKey];
                    const endpointActive = pathname === href;

                    return (
                      <Link
                        key={endpointKey}
                        href={href}
                        className={`docs-side-sublink ${endpointActive ? "docs-side-sublink-active" : ""}`}
                      >
                        {endpoint.label}
                      </Link>
                    );
                  })}
                </div>
              ) : null}

              {entry.href === "/docs/concepts/" ? (
                <div className="docs-side-subnav">
                  <Link
                    href="/docs/concepts/webhooks/"
                    className={`docs-side-sublink ${pathname === "/docs/concepts/webhooks/" ? "docs-side-sublink-active" : ""}`}
                  >
                    Webhook Concepts
                  </Link>
                </div>
              ) : null}

              {entry.href === "/docs/migration/" ? (
                <div className="docs-side-subnav">
                  <Link
                    href="/docs/migration/0-1-0-to-0-2-0/"
                    className={`docs-side-sublink ${pathname === "/docs/migration/0-1-0-to-0-2-0/" ? "docs-side-sublink-active" : ""}`}
                  >
                    0.1.0 -&gt; 0.2.0
                  </Link>
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
