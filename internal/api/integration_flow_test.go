package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"simple-license-server/internal/storage"
)

func TestIntegrationLicenseLifecycleFlow(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run integration tests")
	}

	databaseURL := strings.TrimSpace(os.Getenv("INTEGRATION_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("set INTEGRATION_DATABASE_URL (or DATABASE_URL) to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := storage.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(store.Close)

	managementKey := "management_key_integration_123456"
	server := NewServerWithOptions(
		store,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		map[string]struct{}{managementKey: {}},
		Options{RequestTimeout: 15 * time.Second, RateLimitEnabled: false},
	)
	handler := server.Routes()

	flowSuffix := strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", "")
	fingerprint := "integration-fp-" + flowSuffix

	createAPIKeyRes := doJSONRequest(t, handler, http.MethodPost, "/management/api-keys", createAPIKeyRequest{
		Name: "integration-flow-" + flowSuffix,
	}, map[string]string{
		"Authorization": "Bearer " + managementKey,
	})
	if createAPIKeyRes.Code != http.StatusCreated {
		t.Fatalf("expected create api key status %d, got %d body=%s", http.StatusCreated, createAPIKeyRes.Code, createAPIKeyRes.Body.String())
	}

	createdAPIKey := decodeJSONResponse[createAPIKeyResponse](t, createAPIKeyRes)
	if strings.TrimSpace(createdAPIKey.APIKey) == "" {
		t.Fatalf("expected non-empty generated api key")
	}

	generateRes := doJSONRequest(t, handler, http.MethodPost, "/generate", generateRequest{
		Slug:     "default",
		Metadata: map[string]any{"test_case": "integration_lifecycle_flow"},
	}, map[string]string{
		"Authorization": "Bearer " + createdAPIKey.APIKey,
	})
	if generateRes.Code != http.StatusOK {
		t.Fatalf("expected generate status %d, got %d body=%s", http.StatusOK, generateRes.Code, generateRes.Body.String())
	}

	generated := decodeJSONResponse[generateResponse](t, generateRes)
	if strings.TrimSpace(generated.LicenseKey) == "" {
		t.Fatalf("expected non-empty license key")
	}
	if generated.Status != "inactive" {
		t.Fatalf("expected generated status inactive, got %q", generated.Status)
	}

	activateRes := doJSONRequest(t, handler, http.MethodPost, "/activate", activateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
		Metadata:    map[string]any{"source": "integration-test"},
	}, nil)
	if activateRes.Code != http.StatusOK {
		t.Fatalf("expected activate status %d, got %d body=%s", http.StatusOK, activateRes.Code, activateRes.Body.String())
	}

	activated := decodeJSONResponse[activateResponse](t, activateRes)
	if !activated.Valid {
		t.Fatalf("expected activation to be valid, got %+v", activated)
	}
	if activated.Status != "active" {
		t.Fatalf("expected activated status active, got %q", activated.Status)
	}

	validateBeforeRevokeRes := doJSONRequest(t, handler, http.MethodPost, "/validate", validateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
	}, nil)
	if validateBeforeRevokeRes.Code != http.StatusOK {
		t.Fatalf("expected validate-before-revoke status %d, got %d body=%s", http.StatusOK, validateBeforeRevokeRes.Code, validateBeforeRevokeRes.Body.String())
	}

	validatedBeforeRevoke := decodeJSONResponse[validateResponse](t, validateBeforeRevokeRes)
	if !validatedBeforeRevoke.Valid || validatedBeforeRevoke.Status != "active" {
		t.Fatalf("expected valid active license before revoke, got %+v", validatedBeforeRevoke)
	}

	revokeRes := doJSONRequest(t, handler, http.MethodPost, "/revoke", revokeRequest{
		LicenseKey: generated.LicenseKey,
	}, map[string]string{
		"Authorization": "Bearer " + createdAPIKey.APIKey,
	})
	if revokeRes.Code != http.StatusOK {
		t.Fatalf("expected revoke status %d, got %d body=%s", http.StatusOK, revokeRes.Code, revokeRes.Body.String())
	}

	revoked := decodeJSONResponse[revokeResponse](t, revokeRes)
	if revoked.Valid {
		t.Fatalf("expected revoked valid=false, got %+v", revoked)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked status revoked, got %q", revoked.Status)
	}

	validateAfterRevokeRes := doJSONRequest(t, handler, http.MethodPost, "/validate", validateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
	}, nil)
	if validateAfterRevokeRes.Code != http.StatusOK {
		t.Fatalf("expected validate-after-revoke status %d, got %d body=%s", http.StatusOK, validateAfterRevokeRes.Code, validateAfterRevokeRes.Body.String())
	}

	validatedAfterRevoke := decodeJSONResponse[validateResponse](t, validateAfterRevokeRes)
	if validatedAfterRevoke.Valid {
		t.Fatalf("expected revoked license to be invalid, got %+v", validatedAfterRevoke)
	}
	if validatedAfterRevoke.Status != "revoked" {
		t.Fatalf("expected revoked status after revoke, got %q", validatedAfterRevoke.Status)
	}
	if validatedAfterRevoke.Reason != "revoked" {
		t.Fatalf("expected revoked reason after revoke, got %q", validatedAfterRevoke.Reason)
	}
}

func doJSONRequest(t *testing.T, handler http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		requestBody = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, requestBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeJSONResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v body=%s", err, rr.Body.String())
	}

	return payload
}
