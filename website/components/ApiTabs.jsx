"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import CodeExample from "./CodeExample";
import { apiEndpointGroups, apiEndpoints, endpointDocHref } from "./apiReferenceData";

export default function ApiTabs({ activeGroup = "provisioning" }) {
  const group = apiEndpointGroups[activeGroup] ?? apiEndpointGroups.provisioning;
  const [activeEndpoint, setActiveEndpoint] = useState(group.endpoints[0]);

  useEffect(() => {
    setActiveEndpoint(group.endpoints[0]);
  }, [group]);

  const selectedEndpointKey = group.endpoints.includes(activeEndpoint) ? activeEndpoint : group.endpoints[0];
  const currentEndpoint = useMemo(() => apiEndpoints[selectedEndpointKey], [selectedEndpointKey]);

  return (
    <div className="card api-tabs-card">
      <p className="api-group-description">{group.description}</p>

      <div className="tabs">
        {group.endpoints.map((endpointKey) => (
          <button
            type="button"
            key={endpointKey}
            className={`tab ${endpointKey === activeEndpoint ? "tab-active" : ""}`}
            onClick={() => setActiveEndpoint(endpointKey)}
          >
            {apiEndpoints[endpointKey].label}
          </button>
        ))}
      </div>

      <div className="api-endpoint-prose">
        <div className="api-endpoint-headline">
          <span className="api-endpoint-method">{currentEndpoint.method}</span>
          <code className="api-endpoint-path">{currentEndpoint.path}</code>
        </div>
        <p>{currentEndpoint.summary}</p>
        <ul>
          {currentEndpoint.details.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
        <p style={{ marginTop: "0.65rem" }}>
          <Link className="api-endpoint-link" href={endpointDocHref(currentEndpoint.key)}>
            Open full endpoint page
          </Link>
        </p>
      </div>

      <CodeExample language="bash" code={currentEndpoint.curl} />
    </div>
  );
}
