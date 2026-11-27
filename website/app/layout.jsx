import "./globals.css";

const siteUrl = "https://simplelicenseserver.com";
const siteDescription =
  "Simple License Server is a self-hosted license key API you can run in Docker. Generate, activate, validate, revoke, and deactivate software licenses with a simple JSON API.";

export const metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Simple License Server | Self-Hosted License API for Developers",
    template: "%s | Simple License Server"
  },
  description: siteDescription,
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
      "max-snippet": -1,
      "max-video-preview": -1
    }
  },
  openGraph: {
    type: "website",
    locale: "en_US",
    url: siteUrl,
    siteName: "Simple License Server",
    title: "Simple License Server | Self-Hosted License API for Developers",
    description: siteDescription
  },
  twitter: {
    card: "summary",
    title: "Simple License Server | Self-Hosted License API for Developers",
    description: siteDescription
  },
  alternates: {
    canonical: "/"
  }
};

export const viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#0f131d"
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" className="dark">
      <head>
        <link rel="preconnect" href="https://fonts.bunny.net" />
        <link
          rel="stylesheet"
          href="https://fonts.bunny.net/css?family=inter:400,500,600,700,800|space-grotesk:400,500,600,700"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
