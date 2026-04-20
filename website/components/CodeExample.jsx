"use client";

import { useState } from "react";

export default function CodeExample({ language = "bash", code }) {
  const [copied, setCopied] = useState(false);

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div className="code-panel">
      <div className="code-panel-header">
        <div className="code-panel-dots" aria-hidden="true">
          <span></span>
          <span></span>
          <span></span>
        </div>
        <span className="code-panel-language">{language}</span>
        <button type="button" className="code-panel-copy" onClick={onCopy}>
          <i aria-hidden="true" className="bx bx-copy"></i>
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="code-panel-body">{code}</pre>
    </div>
  );
}
