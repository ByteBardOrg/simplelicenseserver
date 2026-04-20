export const metadata = {
  title: "Terms of Use",
  description: "Terms of Use for Simple License Server.",
  alternates: {
    canonical: "/terms/"
  }
};

export default function TermsPage() {
  return (
    <section className="container content-single">
      <p className="eyebrow">Legal</p>
      <h1>Terms of Use</h1>
      <p>Last updated: 2026-04-20</p>

      <p>These terms are provided by ByteBard (CVR 38774468), a company registered in Denmark.</p>

      <p>
        Simple License Server is provided as-is without warranties of any kind. You are responsible for deployment,
        configuration, security hardening, backups, and compliance in your environment.
      </p>

      <h2>Acceptable use</h2>
      <p>You agree to use the software lawfully and not in ways that violate regulations, contracts, or third-party rights.</p>

      <h2>Liability</h2>
      <p>
        To the maximum extent permitted by law, project maintainers are not liable for any direct or indirect damages
        resulting from use of this software.
      </p>

      <h2>Governing law</h2>
      <p>
        Unless mandatory local consumer protections state otherwise, these terms are governed by Danish law and
        disputes are subject to the competent courts in Denmark.
      </p>

      <h2>Contact</h2>
      <p>For legal or usage questions, contact ByteBard directly through your standard business contact channel.</p>
    </section>
  );
}
