import Script from "next/script";

export default function AnalyticsScript() {
  return (
    <Script
      src="https://analytics.bytebard.org/script.js"
      data-website-id="398f43de-6497-4620-a7a5-7b31527a71fc"
      strategy="afterInteractive"
    />
  );
}
