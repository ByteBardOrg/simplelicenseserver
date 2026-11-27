export const metadata = {
  title: "Privacy Policy",
  description: "Privacy Policy for Simple License Server.",
  alternates: {
    canonical: "/privacy/"
  }
};

export default function PrivacyPage() {
  return (
    <section className="container content-single">
      <p className="eyebrow">Legal</p>
      <h1>Privacy Policy</h1>
      <p>Last updated: 2026-04-20</p>

      <p>This website and project are operated by ByteBard (CVR 38774468), a company registered in Denmark.</p>

      <h2>Data controller context</h2>
      <p>
        For this public website, ByteBard acts as data controller. For self-hosted product deployments, the operator of
        the deployment is normally the data controller for license and activation metadata processed by their instance.
      </p>

      <h2>What data may be processed</h2>
      <p>
        Depending on your integration, metadata can include items such as customer email, order id, machine
        fingerprint, and timestamps related to license events.
      </p>

      <h2>How data is used</h2>
      <p>
        Data is used strictly for license lifecycle operations, including generation, activation, validation,
        revocation, and audit history.
      </p>

      <h2>Website analytics</h2>
      <p>
        We use self-hosted Umami analytics on the public website to understand page traffic and improve site content.
        This tracking applies to the website only and does not apply to the Simple License Server software or
        self-hosted instances run by operators.
      </p>

      <h2>GDPR rights (EU/EEA)</h2>
      <p>
        If GDPR applies to your data, you may have rights to access, rectification, erasure, restriction, objection,
        and data portability. Requests should be directed to the controller responsible for the relevant data processing
        context.
      </p>

      <h2>Contact</h2>
      <p>For privacy questions, contact ByteBard directly through your standard business contact channel.</p>
    </section>
  );
}
