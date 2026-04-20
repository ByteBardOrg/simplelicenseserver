"use client";

import { useState } from "react";
import ApiTabs from "./ApiTabs";
import { apiEndpointGroups, apiEndpoints } from "./apiReferenceData";

export default function ApiDocsExplorer() {
  const [activeGroup, setActiveGroup] = useState("provisioning");
  const groups = Object.values(apiEndpointGroups);

  return (
    <>
      <div className="api-group-grid" id="endpoint-groups">
        {groups.map((group) => (
          <button
            type="button"
            key={group.key}
            className={`card api-group-card ${activeGroup === group.key ? "api-group-card-active" : ""}`}
            onClick={() => setActiveGroup(group.key)}
          >
            <h3>{group.title}</h3>
            <p>{group.summary}</p>
            <ul>
              {group.endpoints.map((endpointKey) => (
                <li key={endpointKey}>{apiEndpoints[endpointKey].label}</li>
              ))}
            </ul>
          </button>
        ))}
      </div>

      <div style={{ marginTop: "1rem" }} id="examples">
        <ApiTabs activeGroup={activeGroup} />
      </div>
    </>
  );
}
