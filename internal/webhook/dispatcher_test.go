package webhook

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
