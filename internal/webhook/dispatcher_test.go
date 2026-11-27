package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"simple-license-server/internal/storage"
)

type fakeDeliveryStore struct {
	mu              sync.Mutex
	markDeliveredID int64
	markStatusCode  int
}

func (f *fakeDeliveryStore) ClaimWebhookDeliveries(ctx context.Context, limit int) ([]storage.WebhookDelivery, error) {
	return nil, nil
}

func (f *fakeDeliveryStore) MarkWebhookDeliveryDelivered(ctx context.Context, deliveryID int64, statusCode int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markDeliveredID = deliveryID
	f.markStatusCode = statusCode
	return nil
}

func (f *fakeDeliveryStore) MarkWebhookDeliveryFailed(ctx context.Context, deliveryID int64, nextAttemptAt time.Time, statusCode int, lastError string, permanent bool) error {
	return nil
}

func TestProcessDeliverySetsIdempotencyHeader(t *testing.T) {
	var (
		idemHeader       string
		deliveryIDHeader string
		eventHeader      string
	)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idemHeader = r.Header.Get("Idempotency-Key")
		deliveryIDHeader = r.Header.Get("X-Webhook-Delivery-Id")
		eventHeader = r.Header.Get("X-Webhook-Event")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	store := &fakeDeliveryStore{}
	d := NewDispatcher(store, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultOptions())

	delivery := storage.WebhookDelivery{
		ID:          42,
		EndpointURL: target.URL,
		EventType:   "license.generated",
		Payload:     map[string]any{"license_key": "A1B2C3-D4E5F6-AB12CD-34EF56-7890AB"},
		Attempts:    1,
		CreatedAt:   time.Now().UTC(),
	}

	d.processDelivery(context.Background(), delivery)

	if idemHeader != "sls-webhook-42" {
		t.Fatalf("expected idempotency header sls-webhook-42, got %q", idemHeader)
	}

	if deliveryIDHeader != "42" {
		t.Fatalf("expected delivery id header 42, got %q", deliveryIDHeader)
	}

	if eventHeader != "license.generated" {
		t.Fatalf("expected webhook event header license.generated, got %q", eventHeader)
	}

	if store.markDeliveredID != 42 {
		t.Fatalf("expected delivery 42 marked delivered, got %d", store.markDeliveredID)
	}
}

func TestWebhookIdempotencyKeyStableAcrossAttempts(t *testing.T) {
	if got := webhookIdempotencyKey(99); got != "sls-webhook-99" {
		t.Fatalf("unexpected idempotency key: %q", got)
	}

	if got := webhookIdempotencyKey(99); got != "sls-webhook-99" {
		t.Fatalf("idempotency key should be stable, got %q", got)
	}
}

func TestProcessDeliverySignsBody(t *testing.T) {
	var (
		signature string
		body      []byte
	)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature = r.Header.Get(SignatureHeader)
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	d := NewDispatcher(&fakeDeliveryStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultOptions())
	d.processDelivery(context.Background(), storage.WebhookDelivery{
		ID: 7, EndpointURL: target.URL, EndpointSecret: "whsec_test", EventType: "license.generated",
		Payload: map[string]any{"license_key": "K"}, Attempts: 1, CreatedAt: time.Now().UTC(),
	})

	parts := strings.SplitN(signature, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "t=") || !strings.HasPrefix(parts[1], "v1=") {
		t.Fatalf("unexpected signature header %q", signature)
	}
	ts, err := strconv.ParseInt(strings.TrimPrefix(parts[0], "t="), 10, 64)
	if err != nil || time.Since(time.Unix(ts, 0)) > time.Minute {
		t.Fatalf("expected a current timestamp, got %q", parts[0])
	}

	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "."))
	mac.Write(body)
	if want := hex.EncodeToString(mac.Sum(nil)); strings.TrimPrefix(parts[1], "v1=") != want {
		t.Fatalf("signature does not verify against the received body")
	}
}

func TestSignIsDeterministic(t *testing.T) {
	got := Sign("secret", 1700000000, []byte(`{"a":1}`))
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(`1700000000.{"a":1}`))
	if want := "t=1700000000,v1=" + hex.EncodeToString(mac.Sum(nil)); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	followed := false
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed = true }))
	defer final.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	store := &fakeDeliveryStore{}
	d := NewDispatcher(store, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultOptions())
	d.processDelivery(context.Background(), storage.WebhookDelivery{
		ID: 8, EndpointURL: redirector.URL, EventType: "license.generated", Payload: map[string]any{}, Attempts: 1,
	})

	if followed {
		t.Fatalf("redirect was followed")
	}
	if store.markDeliveredID == 8 {
		t.Fatalf("a redirect must not count as delivered")
	}
}

func TestCheckWebhookAddress(t *testing.T) {
	cases := map[string]bool{
		"169.254.169.254:80":          false,
		"[fe80::1]:443":               false,
		"[fd00:ec2::254]:80":          false,
		"100.100.100.200:80":          false,
		"[::ffff:169.254.169.254]:80": false,
		"127.0.0.1:8080":              true,
		"10.1.2.3:443":                true,
		"192.168.1.10:443":            true,
		"93.184.216.34:443":           true,
	}
	for address, allowed := range cases {
		err := checkWebhookAddress("tcp", address, nil)
		if allowed && err != nil {
			t.Fatalf("%s: expected allowed, got %v", address, err)
		}
		if !allowed && err == nil {
			t.Fatalf("%s: expected blocked", address)
		}
	}
}
