package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"simple-license-server/internal/storage"
	"simple-license-server/internal/version"
)

type DeliveryStore interface {
	ClaimWebhookDeliveries(ctx context.Context, limit int) ([]storage.WebhookDelivery, error)
	MarkWebhookDeliveryDelivered(ctx context.Context, deliveryID int64, statusCode int) error
	MarkWebhookDeliveryFailed(ctx context.Context, deliveryID int64, nextAttemptAt time.Time, statusCode int, lastError string, permanent bool) error
}

type Options struct {
	PollInterval   time.Duration
	BatchSize      int
	RequestTimeout time.Duration
	MaxAttempts    int
	BackoffMin     time.Duration
	BackoffMax     time.Duration
}

func DefaultOptions() Options {
	return Options{
		PollInterval:   1 * time.Second,
		BatchSize:      20,
		RequestTimeout: 10 * time.Second,
		MaxAttempts:    8,
		BackoffMin:     2 * time.Second,
		BackoffMax:     5 * time.Minute,
	}
}

type Dispatcher struct {
	store  DeliveryStore
	logger *slog.Logger
	client *http.Client
	opts   Options
}

func NewDispatcher(store DeliveryStore, logger *slog.Logger, opts Options) *Dispatcher {
	defaults := DefaultOptions()
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaults.PollInterval
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaults.BatchSize
	}
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = defaults.RequestTimeout
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = defaults.MaxAttempts
	}
	if opts.BackoffMin <= 0 {
		opts.BackoffMin = defaults.BackoffMin
	}
	if opts.BackoffMax <= 0 {
		opts.BackoffMax = defaults.BackoffMax
	}
	if opts.BackoffMax < opts.BackoffMin {
		opts.BackoffMax = opts.BackoffMin
	}

	return &Dispatcher{
		store:  store,
		logger: logger,
		client: &http.Client{Timeout: opts.RequestTimeout},
		opts:   opts,
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.opts.PollInterval)
	defer ticker.Stop()

	d.processBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.processBatch(ctx)
		}
	}
}

func (d *Dispatcher) processBatch(ctx context.Context) {
	deliveries, err := d.store.ClaimWebhookDeliveries(ctx, d.opts.BatchSize)
	if err != nil {
		d.logger.Error("failed to claim webhook deliveries", "error", err)
		return
	}

	for _, delivery := range deliveries {
		d.processDelivery(ctx, delivery)
	}
}

func (d *Dispatcher) processDelivery(ctx context.Context, delivery storage.WebhookDelivery) {
	if strings.TrimSpace(delivery.EndpointURL) == "" {
		err := d.store.MarkWebhookDeliveryFailed(ctx, delivery.ID, time.Now().UTC(), 0, "webhook endpoint URL missing", true)
		if err != nil {
			d.logger.Error("failed to mark delivery with missing endpoint", "delivery_id", delivery.ID, "error", err)
		}
		return
	}

	requestBody, err := json.Marshal(map[string]any{
		"id":          delivery.ID,
		"type":        delivery.EventType,
		"attempt":     delivery.Attempts,
		"occurred_at": delivery.CreatedAt,
		"data":        delivery.Payload,
	})
	if err != nil {
		d.handleDeliveryFailure(ctx, delivery, 0, fmt.Errorf("marshal webhook body: %w", err), true)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.EndpointURL, bytes.NewReader(requestBody))
	if err != nil {
		d.handleDeliveryFailure(ctx, delivery, 0, fmt.Errorf("build webhook request: %w", err), true)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "simple-license-server-webhook/"+version.Current)
	req.Header.Set("Idempotency-Key", webhookIdempotencyKey(delivery.ID))
	req.Header.Set("X-Webhook-Delivery-Id", strconv.FormatInt(delivery.ID, 10))
	req.Header.Set("X-Webhook-Event", delivery.EventType)

	resp, err := d.client.Do(req)
	if err != nil {
		d.handleDeliveryFailure(ctx, delivery, 0, err, false)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := d.store.MarkWebhookDeliveryDelivered(ctx, delivery.ID, resp.StatusCode); err != nil {
			d.logger.Error("failed to mark webhook delivery delivered", "delivery_id", delivery.ID, "error", err)
		}
		return
	}

	permanent := isPermanentHTTPStatus(resp.StatusCode)
	d.handleDeliveryFailure(ctx, delivery, resp.StatusCode, fmt.Errorf("webhook responded with status %d", resp.StatusCode), permanent)
}

func (d *Dispatcher) handleDeliveryFailure(ctx context.Context, delivery storage.WebhookDelivery, statusCode int, cause error, permanent bool) {
	permanentFailure := permanent || delivery.Attempts >= d.opts.MaxAttempts
	nextAttemptAt := time.Now().UTC().Add(d.backoffForAttempt(delivery.Attempts))
	err := d.store.MarkWebhookDeliveryFailed(ctx, delivery.ID, nextAttemptAt, statusCode, cause.Error(), permanentFailure)
	if err != nil {
		d.logger.Error("failed to mark webhook delivery failed", "delivery_id", delivery.ID, "error", err)
		return
	}

	attrs := []any{
		"delivery_id", delivery.ID,
		"event_type", delivery.EventType,
		"attempt", delivery.Attempts,
		"status_code", statusCode,
		"error", cause.Error(),
	}

	if permanentFailure {
		d.logger.Error("webhook delivery permanently failed", attrs...)
		return
	}

	attrs = append(attrs, "next_attempt_at", nextAttemptAt)
	d.logger.Warn("webhook delivery failed; will retry", attrs...)
}

func (d *Dispatcher) backoffForAttempt(attempt int) time.Duration {
	if attempt <= 1 {
		return d.opts.BackoffMin
	}

	backoff := d.opts.BackoffMin
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= d.opts.BackoffMax {
			return d.opts.BackoffMax
		}
	}

	return backoff
}

func isPermanentHTTPStatus(status int) bool {
	if status == 408 || status == 429 {
		return false
	}

	return status >= 400 && status < 500
}

func webhookIdempotencyKey(deliveryID int64) string {
	return fmt.Sprintf("sls-webhook-%d", deliveryID)
}

var _ DeliveryStore = (*storage.Store)(nil)
