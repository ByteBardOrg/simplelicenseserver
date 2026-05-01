"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { apiEndpointGroups, apiEndpoints, endpointDocHref } from "./apiReferenceData";

const topLinks = [
  { href: "/docs/", title: "Overview", icon: "bx-home-alt-2" },
  { href: "/docs/quickstart/", title: "Quickstart", icon: "bx-rocket" },
  { href: "/docs/authentication/", title: "Authentication", icon: "bx-lock-alt" }
];

const referenceExtras = [{ href: "/docs/management-api/", title: "Management API" }];

const supportLinks = [
  { href: "/docs/configuration/", title: "Configuration" },
  { href: "/docs/guides/stripe/", title: "Stripe Example" },
  { href: "/docs/migration/", title: "Migration" }
];

const conceptLinks = [
  { href: "/docs/concepts/slugs/", title: "Slug Concepts" },
  { href: "/docs/concepts/webhooks/", title: "Webhook Concepts" }
];

export default function DocsSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const licenseEndpointKeys = apiEndpointGroups.license.endpoints;
  const managementEndpointKeys = apiEndpointGroups.management.endpoints;

  const isLicenseEndpointPath = licenseEndpointKeys.some((endpointKey) => pathname === endpointDocHref(endpointKey));
  const isManagementEndpointPath = managementEndpointKeys.some((endpointKey) => pathname === endpointDocHref(endpointKey));

  const isApiReferenceSection =
    pathname === "/docs/api/" || pathname === "/docs/management-api/" || isLicenseEndpointPath || isManagementEndpointPath;

  const isConceptsSection = pathname === "/docs/concepts/" || pathname === "/docs/concepts/slugs/" || pathname === "/docs/concepts/webhooks/";

  const [openHeader, setOpenHeader] = useState(null);

  useEffect(() => {
    if (isApiReferenceSection) {
      setOpenHeader("api");
      return;
    }

    if (isConceptsSection) {
      setOpenHeader("concepts");
      return;
    }

    setOpenHeader(null);
  }, [pathname, isApiReferenceSection, isConceptsSection]);

  const collapseHeaders = () => {
    setOpenHeader(null);
  };

  const toggleHeader = (key, overviewHref) => {
    setOpenHeader((current) => {
      const next = current === key ? null : key;
      if (next === key) {
        router.push(overviewHref);
      }
      return next;
    });
  };

  const apiHeaderActive = openHeader ? openHeader === "api" : isApiReferenceSection;
  const conceptsHeaderActive = openHeader ? openHeader === "concepts" : isConceptsSection;

  const isLinkActive = (href) => {
    if (href === "/docs/migration/") {
      return pathname.startsWith("/docs/migration/");
    }
    if (href === "/docs/concepts/") {
      return pathname.startsWith("/docs/concepts/");
    }
    return pathname === href;
  };

  return (
    <aside className="docs-sidebar" aria-label="Documentation sections">
      <div className="docs-sidebar-header">
        <h2>Documentation</h2>
      </div>

      <nav className="docs-sidebar-nav">
        {topLinks.map((entry) => (
          <Link
            key={entry.href}
            href={entry.href}
            className={`docs-side-link ${isLinkActive(entry.href) ? "docs-side-link-active" : ""}`}
            onClick={collapseHeaders}
          >
            <i aria-hidden="true" className={`bx ${entry.icon}`}></i>
            <span>{entry.title}</span>
          </Link>
        ))}

        <button
          type="button"
          className={`docs-side-link docs-side-header-button ${apiHeaderActive ? "docs-side-link-active" : ""}`}
          aria-expanded={openHeader === "api" ? "true" : "false"}
          aria-controls="docs-subnav-apireference"
          onClick={() => toggleHeader("api", "/docs/api/")}
        >
          <i aria-hidden="true" className="bx bx-code-alt"></i>
          <span>API Reference</span>
        </button>

        {openHeader === "api" ? (
          <div className="docs-side-subnav" id="docs-subnav-apireference">
            <Link href="/docs/api/" className={`docs-side-sublink ${pathname === "/docs/api/" ? "docs-side-sublink-active" : ""}`}>
              Overview
            </Link>
            <p className="docs-side-subgroup-title">License Management</p>
            {licenseEndpointKeys.map((endpointKey) => {
              const href = endpointDocHref(endpointKey);
              const endpoint = apiEndpoints[endpointKey];
              const endpointActive = pathname === href;

              return (
                <Link key={endpointKey} href={href} className={`docs-side-sublink ${endpointActive ? "docs-side-sublink-active" : ""}`}>
                  <span className="docs-side-method">{endpoint.method}</span>
                  <span>{endpoint.path}</span>
                </Link>
              );
            })}

            {referenceExtras.map((entry) => (
              <Link
                key={entry.href}
                href={entry.href}
                className={`docs-side-sublink ${
                  (entry.href === "/docs/management-api/" && (pathname === "/docs/management-api/" || isManagementEndpointPath)) ||
                  pathname === entry.href
                    ? "docs-side-sublink-active"
                    : ""
                }`}
              >
                {entry.title}
              </Link>
            ))}
          </div>
        ) : null}

        <div className="docs-side-divider" aria-hidden="true"></div>

        <button
          type="button"
          className={`docs-side-link docs-side-link-secondary docs-side-header-button ${conceptsHeaderActive ? "docs-side-link-active" : ""}`}
          aria-expanded={openHeader === "concepts" ? "true" : "false"}
          aria-controls="docs-subnav-concepts"
          onClick={() => toggleHeader("concepts", "/docs/concepts/")}
        >
          <span>Concepts</span>
        </button>

        {openHeader === "concepts" ? (
          <div className="docs-side-subnav" id="docs-subnav-concepts">
            <Link href="/docs/concepts/" className={`docs-side-sublink ${pathname === "/docs/concepts/" ? "docs-side-sublink-active" : ""}`}>
              Overview
            </Link>
            {conceptLinks.map((concept) => (
              <Link
                key={concept.href}
                href={concept.href}
                className={`docs-side-sublink ${pathname === concept.href ? "docs-side-sublink-active" : ""}`}
              >
                {concept.title}
              </Link>
            ))}
          </div>
        ) : null}

        {supportLinks.map((entry) => (
          <Link
            key={entry.href}
            href={entry.href}
            className={`docs-side-link docs-side-link-secondary ${isLinkActive(entry.href) ? "docs-side-link-active" : ""}`}
            onClick={collapseHeaders}
          >
            <span>{entry.title}</span>
          </Link>
        ))}
      </nav>
    </aside>
  );
}
