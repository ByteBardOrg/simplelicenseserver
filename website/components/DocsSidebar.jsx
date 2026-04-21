"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
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

const conceptsLinks = [
  { href: "/docs/concepts/slugs/", title: "Slug Concepts" },
  { href: "/docs/concepts/webhooks/", title: "Webhook Concepts" }
];

const managementEndpointGroups = [
  {
    title: "Slug Endpoints",
    endpoints: [
      "management-slugs-list",
      "management-slugs-create",
      "management-slugs-get",
      "management-slugs-update",
      "management-slugs-delete"
    ]
  },
  {
    title: "API Key Endpoints",
    endpoints: [
      "management-api-keys-list",
      "management-api-keys-create",
      "management-api-keys-revoke"
    ]
  },
  {
    title: "Webhook Endpoints",
    endpoints: [
      "management-webhooks-list",
      "management-webhooks-create",
      "management-webhooks-update",
      "management-webhooks-delete"
    ]
  }
];

const migrationLinks = [{ href: "/docs/migration/0-1-0-to-0-2-0/", title: "0.1.0 -> 0.2.0" }];

export default function DocsSidebar() {
  const pathname = usePathname();
  const licenseEndpointKeys = apiEndpointGroups.license.endpoints;
  const managementEndpointKeys = apiEndpointGroups.management.endpoints;

  const isLicenseEndpointPath = licenseEndpointKeys.some((endpointKey) => pathname === endpointDocHref(endpointKey));
  const isManagementEndpointPath = managementEndpointKeys.some((endpointKey) => pathname === endpointDocHref(endpointKey));

  const isLicenseApiSection = pathname === "/docs/api/" || isLicenseEndpointPath;
  const isManagementApiSection = pathname === "/docs/management-api/" || isManagementEndpointPath;
  const isMigrationSection = pathname === "/docs/migration/" || pathname === "/docs/migration/0-1-0-to-0-2-0/";
  const isConceptsSection = pathname === "/docs/concepts/" || conceptsLinks.some((entry) => pathname === entry.href);

  const [openSections, setOpenSections] = useState(() => ({
    "/docs/api/": isLicenseApiSection,
    "/docs/management-api/": isManagementApiSection,
    "/docs/concepts/": isConceptsSection,
    "/docs/migration/": isMigrationSection
  }));

  useEffect(() => {
    setOpenSections((previous) => ({
      ...previous,
      "/docs/api/": previous["/docs/api/"] || isLicenseApiSection,
      "/docs/management-api/": previous["/docs/management-api/"] || isManagementApiSection,
      "/docs/concepts/": previous["/docs/concepts/"] || isConceptsSection,
      "/docs/migration/": previous["/docs/migration/"] || isMigrationSection
    }));
  }, [isLicenseApiSection, isManagementApiSection, isConceptsSection, isMigrationSection]);

  const groupedManagementEndpoints = useMemo(() => {
    const allowedKeys = new Set(managementEndpointKeys);

    return managementEndpointGroups
      .map((group) => ({
        ...group,
        endpoints: group.endpoints.filter((endpointKey) => allowedKeys.has(endpointKey))
      }))
      .filter((group) => group.endpoints.length > 0);
  }, [managementEndpointKeys]);

  const sectionHasSubnav = (href) =>
    href === "/docs/api/" || href === "/docs/management-api/" || href === "/docs/concepts/" || href === "/docs/migration/";

  const toggleSection = (href) => {
    setOpenSections((previous) => ({
      ...previous,
      [href]: !previous[href]
    }));
  };

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
              <div className="docs-side-link-row">
                <Link href={entry.href} className={`docs-side-link ${active ? "docs-side-link-active" : ""}`}>
                  <i aria-hidden="true" className={`bx ${entry.icon}`}></i>
                  <span>{entry.title}</span>
                </Link>

                {sectionHasSubnav(entry.href) ? (
                  <button
                    type="button"
                    className="docs-side-toggle"
                    aria-label={`Toggle ${entry.title} links`}
                    aria-expanded={openSections[entry.href] ? "true" : "false"}
                    aria-controls={`docs-subnav-${entry.href.replace(/[^a-z0-9]/gi, "")}`}
                    onClick={() => toggleSection(entry.href)}
                  >
                    <i
                      aria-hidden="true"
                      className={`bx ${openSections[entry.href] ? "bx-chevron-down" : "bx-chevron-right"}`}
                    ></i>
                  </button>
                ) : null}
              </div>

              {entry.href === "/docs/api/" && openSections[entry.href] ? (
                <div className="docs-side-subnav" id={`docs-subnav-${entry.href.replace(/[^a-z0-9]/gi, "")}`}>
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

              {entry.href === "/docs/management-api/" && openSections[entry.href] ? (
                <div className="docs-side-subnav" id={`docs-subnav-${entry.href.replace(/[^a-z0-9]/gi, "")}`}>
                  {groupedManagementEndpoints.map((group) => (
                    <div key={group.title} className="docs-side-subgroup">
                      <p className="docs-side-subgroup-title">{group.title}</p>
                      {group.endpoints.map((endpointKey) => {
                        const href = endpointDocHref(endpointKey);
                        const endpoint = apiEndpoints[endpointKey];
                        const endpointActive = pathname === href;

                        return (
                          <Link
                            key={endpointKey}
                            href={href}
                            className={`docs-side-sublink docs-side-sublink-management ${endpointActive ? "docs-side-sublink-active" : ""}`}
                          >
                            {endpoint.label}
                          </Link>
                        );
                      })}
                    </div>
                  ))}
                </div>
              ) : null}

              {entry.href === "/docs/concepts/" && openSections[entry.href] ? (
                <div className="docs-side-subnav" id={`docs-subnav-${entry.href.replace(/[^a-z0-9]/gi, "")}`}>
                  {conceptsLinks.map((concept) => (
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

              {entry.href === "/docs/migration/" && openSections[entry.href] ? (
                <div className="docs-side-subnav" id={`docs-subnav-${entry.href.replace(/[^a-z0-9]/gi, "")}`}>
                  {migrationLinks.map((migrationEntry) => (
                    <Link
                      key={migrationEntry.href}
                      href={migrationEntry.href}
                      className={`docs-side-sublink ${pathname === migrationEntry.href ? "docs-side-sublink-active" : ""}`}
                    >
                      {migrationEntry.title}
                    </Link>
                  ))}
                </div>
              ) : null}
            </div>
          );
        })}
      </nav>
    </aside>
  );
}
