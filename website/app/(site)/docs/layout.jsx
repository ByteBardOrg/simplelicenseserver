import DocsSidebar from "../../../components/DocsSidebar";

export default function DocsLayout({ children }) {
  return (
    <div className="docs-page">
      <div className="docs-shell">
        <DocsSidebar />
        <div className="docs-main">{children}</div>
      </div>
    </div>
  );
}
