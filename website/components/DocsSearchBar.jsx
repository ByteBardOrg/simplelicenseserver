"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { docsIndex } from "./docsIndex";

function matches(entry, query) {
  const value = query.toLowerCase().trim();
  if (!value) {
    return false;
  }

  return [entry.title, entry.summary, ...(entry.keywords || [])].some((item) =>
    item.toLowerCase().includes(value)
  );
}

export default function DocsSearchBar() {
  const router = useRouter();
  const inputRef = useRef(null);
  const [query, setQuery] = useState("");

  useEffect(() => {
    const onKeyDown = (event) => {
      if (event.key !== "/") {
        return;
      }

      const target = event.target;
      const isTyping =
        target instanceof HTMLElement &&
        (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable);

      if (isTyping) {
        return;
      }

      event.preventDefault();
      inputRef.current?.focus();
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const results = useMemo(() => docsIndex.filter((entry) => matches(entry, query)).slice(0, 6), [query]);

  const openResult = (href) => {
    setQuery("");
    router.push(href);
  };

  const onSubmit = (event) => {
    event.preventDefault();

    const first = results[0];
    if (first) {
      openResult(first.href);
      return;
    }

    router.push("/docs/");
  };

  const showResults = query.trim().length > 0;

  return (
    <form className="docs-search" role="search" onSubmit={onSubmit}>
      <i aria-hidden="true" className="bx bx-search-alt-2 docs-search-icon"></i>
      <input
        ref={inputRef}
        type="search"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        className="docs-search-input"
        placeholder="Search docs..."
        aria-label="Search documentation"
      />
      <span className="docs-search-hint">/</span>

      {showResults ? (
        <div className="docs-search-results" role="listbox" aria-label="Documentation search results">
          {results.length === 0 ? (
            <div className="docs-search-empty">No direct match. Press Enter to open docs overview.</div>
          ) : (
            results.map((entry) => (
              <button
                type="button"
                key={entry.href}
                className="docs-search-result"
                onClick={() => openResult(entry.href)}
              >
                <span className="docs-search-title">{entry.title}</span>
                <span className="docs-search-summary">{entry.summary}</span>
              </button>
            ))
          )}
        </div>
      ) : null}
    </form>
  );
}
