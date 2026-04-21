import Link from "next/link";
import CodeExample from "../../../../../components/CodeExample";

export const metadata = {
  title: "Stripe Example",
  description:
    "Stripe example integration for Simple License Server using a Vercel serverless webhook intermediary.",
  alternates: {
    canonical: "/docs/guides/stripe/"
  }
};

const envExample = `NEXT_PUBLIC_APP_URL=https://your-app.example.com
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
STRIPE_PRICE_ID=price_123
LICENSE_SERVER_URL=https://licenses.example.com
LICENSE_SERVER_SERVER_API_KEY=a94f2c8b5e11d67e8a03c91fd78122e3f6acbb16de2202aa0f814cc2f71d10be`;

const checkoutSessionExample = `import Stripe from "stripe"

export const runtime = "nodejs"

const stripeSecretKey = process.env.STRIPE_SECRET_KEY
const appUrl = process.env.NEXT_PUBLIC_APP_URL
const priceId = process.env.STRIPE_PRICE_ID

if (!stripeSecretKey || !appUrl || !priceId) {
  throw new Error("Missing STRIPE_SECRET_KEY, NEXT_PUBLIC_APP_URL, or STRIPE_PRICE_ID")
}

const stripe = new Stripe(stripeSecretKey)

export async function POST() {
  const slug = "default"

  const session = await stripe.checkout.sessions.create({
    mode: "payment",
    line_items: [{ price: priceId, quantity: 1 }],
    success_url: appUrl + "/success?session_id={CHECKOUT_SESSION_ID}",
    cancel_url: appUrl + "/pricing",
    metadata: { slug },
  })

  return Response.json({ checkout_url: session.url })
}`;

const webhookExample = `import Stripe from "stripe"

export const runtime = "nodejs"

const stripeSecretKey = process.env.STRIPE_SECRET_KEY
const stripeWebhookSecret = process.env.STRIPE_WEBHOOK_SECRET
const licenseServerUrl = process.env.LICENSE_SERVER_URL
const licenseServerApiKey = process.env.LICENSE_SERVER_SERVER_API_KEY

  if (!stripeSecretKey || !stripeWebhookSecret || !licenseServerUrl || !licenseServerApiKey) {
  throw new Error("Missing one or more required environment variables")
}

const stripe = new Stripe(stripeSecretKey)

async function generateLicense(input: {
  slug: string
  eventId: string
  email: string | null
  customerId: string | null
  checkoutSessionId: string
}) {
  const response = await fetch(licenseServerUrl + "/generate", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer " + licenseServerApiKey,
      "Idempotency-Key": input.eventId,
    },
    body: JSON.stringify({
      slug: input.slug,
      metadata: {
        provider: "stripe",
        provider_event_id: input.eventId,
        provider_customer_id: input.customerId,
        provider_checkout_session_id: input.checkoutSessionId,
        email: input.email,
      },
    }),
  })

  if (!response.ok) {
    const details = await response.text()
    throw new Error("generate failed (" + response.status + "): " + details)
  }

  return (await response.json()) as { license_key: string }
}

export async function POST(req: Request) {
  const signature = req.headers.get("stripe-signature")
  if (!signature) return new Response("missing stripe-signature", { status: 400 })

  const rawBody = await req.text()

  let stripeEvent: Stripe.Event
  try {
    stripeEvent = stripe.webhooks.constructEvent(rawBody, signature, stripeWebhookSecret)
  } catch {
    return new Response("invalid signature", { status: 400 })
  }

  if (stripeEvent.type !== "checkout.session.completed") {
    return new Response("ignored", { status: 200 })
  }

  const session = stripeEvent.data.object as Stripe.Checkout.Session

  if (session.payment_status !== "paid") {
    return new Response("payment not completed", { status: 200 })
  }

  const slug = session.metadata?.slug?.trim()
  if (!slug) return new Response("missing slug metadata", { status: 400 })

  const customerId =
    typeof session.customer === "string"
      ? session.customer
      : session.customer?.id || null

  try {
    const generated = await generateLicense({
      slug,
      eventId: stripeEvent.id,
      email: session.customer_details?.email || null,
      customerId,
      checkoutSessionId: session.id,
    })

    // Persist your own support mapping table.
    // await db.licenseMap.upsert({ stripeSessionId: session.id, licenseKey: generated.license_key, ... })

    return Response.json({ ok: true, license_key: generated.license_key })
  } catch (error) {
    console.error("license generation failed", error)
    return new Response("generate failed", { status: 500 })
  }
}`;

const stripeCliExample = `stripe listen --forward-to localhost:3000/api/webhooks/stripe

stripe trigger checkout.session.completed`;

const generateCurlExample = `curl -sS https://licenses.example.com/generate \\
  -H "Authorization: Bearer a94f2c8b5e11d67e8a03c91fd78122e3f6acbb16de2202aa0f814cc2f71d10be" \\
  -H "Idempotency-Key: evt_123" \\
  -H "Content-Type: application/json" \\
  -d '{
    "slug": "default",
    "metadata": {
      "provider": "stripe",
      "provider_event_id": "evt_123",
      "provider_checkout_session_id": "cs_test_123",
      "email": "user@example.com"
    }
  }'`;

export default function StripeGuidePage() {
  return (
    <section className="docs-content">
      <p className="eyebrow">Example</p>
      <h1>Stripe example: checkout -&gt; Vercel webhook -&gt; license generation</h1>
      <p>
        This guide shows one reliable production pattern: Stripe handles checkout, a Vercel serverless route verifies
        webhook signatures, and your backend bridge calls <code>POST /generate</code> on Simple License Server.
      </p>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>What happens step by step</h3>
        <p>
          Think of the flow as two lanes: the browser lane (customer checkout) and the trusted backend lane (webhook +
          license generation).
        </p>
        <ol>
          <li>
            Your app calls <code>app/api/checkout/route.ts</code> to create a Stripe Checkout Session and includes
            <code> metadata.slug</code> so you know which product policy to generate.
          </li>
          <li>
            The customer is redirected to Stripe Checkout and pays there; no license server keys are exposed to the
            browser.
          </li>
          <li>
            Stripe marks the session complete and sends <code>checkout.session.completed</code> to your Vercel webhook
            endpoint.
          </li>
          <li>
            Your webhook verifies the Stripe signature, checks event type and payment status, and rejects anything that
            does not pass validation.
          </li>
          <li>
            For valid events, your webhook calls <code>POST /generate</code> with the server API key and
            <code> Idempotency-Key: event.id</code> so Stripe retries cannot create duplicate licenses.
          </li>
          <li>
            Simple License Server returns a license key; your app stores Stripe-to-license mapping, sends delivery
            email or portal update, and then returns <code>2xx</code> to Stripe.
          </li>
        </ol>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>1) Set environment variables</h3>
        <p>
          <code>LICENSE_SERVER_SERVER_API_KEY</code> should be a generated server key created via
          <code> POST /management/api-keys</code>.
        </p>
        <CodeExample language="bash" code={envExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>2) Create a Checkout Session route in your app</h3>
        <p>
          File: <code>app/api/checkout/route.ts</code>
        </p>
        <p>
          The critical detail is setting <code>metadata.slug</code>, because your webhook uses this to call
          <code> POST /generate</code>.
        </p>
        <CodeExample language="typescript" code={checkoutSessionExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>3) Add the Stripe webhook intermediary route</h3>
        <p>
          File: <code>app/api/webhooks/stripe/route.ts</code>
        </p>
        <p>
          This route verifies Stripe signatures and uses <code>Idempotency-Key: event.id</code> to make webhook
          retries safe.
        </p>
        <CodeExample language="typescript" code={webhookExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>4) Configure Stripe webhook destination</h3>
        <ul>
          <li>Endpoint URL: <code>https://your-app.example.com/api/webhooks/stripe</code></li>
          <li>Event to subscribe: <code>checkout.session.completed</code></li>
          <li>Copy the endpoint signing secret into <code>STRIPE_WEBHOOK_SECRET</code></li>
        </ul>
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>5) Test locally with Stripe CLI</h3>
        <CodeExample language="bash" code={stripeCliExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>6) Verify your /generate contract</h3>
        <p>
          Your intermediary should send this shape to Simple License Server (server key required on
          <code> /generate</code>).
        </p>
        <CodeExample language="bash" code={generateCurlExample} />
      </div>

      <div className="card" style={{ marginTop: "1rem" }}>
        <h3>Production notes</h3>
        <ul>
          <li>Keep Stripe and license server keys only in server-side environment storage.</li>
          <li>Return non-2xx on generation failures so Stripe retries the event.</li>
          <li>Persist your own mapping table for support and key re-delivery workflows.</li>
          <li>Do not generate licenses from success-page redirects alone.</li>
        </ul>
      </div>

      <p style={{ marginTop: "1rem" }}>
        <Link className="button button-secondary" href="/docs/">
          Back to docs overview
        </Link>
      </p>
    </section>
  );
}
