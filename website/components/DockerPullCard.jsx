"use client";

import { useState } from "react";

const command = "docker pull bytebardorg/simplelicenseserver";

export default function DockerPullCard() {
  const [copied, setCopied] = useState(false);

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div className="command-shell">
      <div className="command-card">
        <div className="command">
          <span className="pill">$</span>
          <code>{command}</code>
        </div>
        <button type="button" className="command-copy" onClick={onCopy}>
          <i aria-hidden="true" className={`bx ${copied ? "bx-check" : "bx-copy"}`}></i>
          <span className="command-copy-label">{copied ? "Copied" : "Copy"}</span>
        </button>
      </div>
    </div>
  );
}
