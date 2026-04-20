package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"simple-license-server/internal/storage"
)

type stubService struct {
	pingFn               func(ctx context.Context) error
	generateLicenseFn    func(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error)
	generateIdempotentFn func(ctx context.Context, endpoint, idemKey, requestHash, slugName string, metadata map[string]any) (storage.GeneratedLicense, json.RawMessage, bool, error)
	revokeLicenseFn      func(ctx context.Context, licenseKey string) (storage.RevokeResult, error)
	activateLicenseFn    func(ctx context.Context, licenseKey, fingerprint string, metadata map[string]any) (storage.ActivationResult, error)
	validateLicenseFn    func(ctx context.Context, licenseKey, fingerprint string) (storage.ValidationResult, error)
	deactivateLicenseFn  func(ctx context.Context, licenseKey, fingerprint, reason string) (storage.DeactivationResult, error)
}

func (s stubService) Ping(ctx context.Context) error {
	if s.pingFn != nil {
		return s.pingFn(ctx)
	}
	return nil
}

func (s stubService) GenerateLicenseIdempotent(ctx context.Context, endpoint, idemKey, requestHash, slugName string, metadata map[string]any) (storage.GeneratedLicense, json.RawMessage, bool, error) {
	if s.generateIdempotentFn != nil {
		return s.generateIdempotentFn(ctx, endpoint, idemKey, requestHash, slugName, metadata)
	}
	return storage.GeneratedLicense{}, nil, false, nil
}

func (s stubService) GenerateLicense(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error) {
	if s.generateLicenseFn != nil {
		return s.generateLicenseFn(ctx, slugName, metadata)
	}
	return storage.GeneratedLicense{}, nil
}

func (s stubService) RevokeLicense(ctx context.Context, licenseKey string) (storage.RevokeResult, error) {
	if s.revokeLicenseFn != nil {
		return s.revokeLicenseFn(ctx, licenseKey)
	}
	return storage.RevokeResult{}, nil
}

func (s stubService) ActivateLicense(ctx context.Context, licenseKey, fingerprint string, metadata map[string]any) (storage.ActivationResult, error) {
	if s.activateLicenseFn != nil {
		return s.activateLicenseFn(ctx, licenseKey, fingerprint, metadata)
	}
	return storage.ActivationResult{}, nil
}

func (s stubService) ValidateLicense(ctx context.Context, licenseKey, fingerprint string) (storage.ValidationResult, error) {
	if s.validateLicenseFn != nil {
		return s.validateLicenseFn(ctx, licenseKey, fingerprint)
	}
	return storage.ValidationResult{}, nil
}

func (s stubService) DeactivateLicense(ctx context.Context, licenseKey, fingerprint, reason string) (storage.DeactivationResult, error) {
	if s.deactivateLicenseFn != nil {
		return s.deactivateLicenseFn(ctx, licenseKey, fingerprint, reason)
	}
	return storage.DeactivationResult{}, nil
}

func TestGenerateRequiresServerKey(t *testing.T) {
	s := NewServer(stubService{}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"server_key_dev_123": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(`{"slug":"default"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestGenerateWithBearerKey(t *testing.T) {
	called := false
	deadlineSeen := false
	createdAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

	s := NewServer(stubService{
		generateLicenseFn: func(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error) {
			called = true
			if _, ok := ctx.Deadline(); ok {
				deadlineSeen = true
			}

			return storage.GeneratedLicense{
				LicenseKey: "AAAA-BBBB-CCCC-DDDD",
				Slug:       slugName,
				Status:     "inactive",
				Metadata:   metadata,
				CreatedAt:  createdAt,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"server_key_dev_123": {}}, 15*time.Second)

	body := `{"slug":"default","metadata":{"email":"user@example.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer server_key_dev_123")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	if !called {
		t.Fatalf("expected service GenerateLicense to be called")
	}

	if !deadlineSeen {
		t.Fatalf("expected request timeout middleware to set context deadline")
	}
}

func TestExtractAPIKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test_key")

	if got := extractAPIKey(req); got != "test_key" {
		t.Fatalf("expected bearer key, got %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-API-Key", "x_key")

	if got := extractAPIKey(req2); got != "x_key" {
		t.Fatalf("expected X-API-Key key, got %q", got)
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	tooLarge := bytes.Repeat([]byte("a"), maxBodyBytes+2)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tooLarge))
	req.Header.Set("Content-Type", "application/json")

	var dst map[string]any
	err := decodeJSON(req, &dst)
	if err == nil {
		t.Fatalf("expected oversized body error")
	}

	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeJSONRejectsMissingContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"slug":"default"}`))

	var dst map[string]any
	err := decodeJSON(req, &dst)
	if err == nil {
		t.Fatalf("expected content-type validation error")
	}

	if !strings.Contains(err.Error(), "Content-Type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSecurityHeadersApplied(t *testing.T) {
	s := NewServer(stubService{}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"server_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}

	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected no-store cache header, got %q", got)
	}
}

func TestRateLimitMiddlewareReturns429(t *testing.T) {
	s := NewServerWithOptions(
		stubService{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		map[string]struct{}{"server_key_dev_123456": {}},
		Options{
			RequestTimeout:        15 * time.Second,
			RateLimitEnabled:      true,
			RateLimitGlobalRPS:    1,
			RateLimitGlobalBurst:  1,
			RateLimitPerIPRPS:     1,
			RateLimitPerIPBurst:   1,
			RateLimitIPTTL:        time.Minute,
			RateLimitMaxIPEntries: 10,
			TrustProxyHeaders:     false,
		},
	)

	req1 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	rr1 := httptest.NewRecorder()
	s.Routes().ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	rr2 := httptest.NewRecorder()
	s.Routes().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be rate limited, got %d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestHashGenerateRequestStableAcrossMetadataOrder(t *testing.T) {
	reqA := generateRequest{Slug: "default", Metadata: map[string]any{"a": 1, "b": 2}}
	reqB := generateRequest{Slug: "default", Metadata: map[string]any{"b": 2, "a": 1}}

	hashA, err := hashGenerateRequest(reqA)
	if err != nil {
		t.Fatalf("hash reqA: %v", err)
	}

	hashB, err := hashGenerateRequest(reqB)
	if err != nil {
		t.Fatalf("hash reqB: %v", err)
	}

	if hashA != hashB {
		t.Fatalf("expected stable hashes, got %q != %q", hashA, hashB)
	}
}
