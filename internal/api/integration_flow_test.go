package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	generateRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/generate", generateRequest{
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
	if generated.ID == "" {
		t.Fatalf("expected generate to return the license id")
	}
	if generated.Status != "inactive" {
		t.Fatalf("expected generated status inactive, got %q", generated.Status)
	}

	activateRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/activate", activateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
		Metadata:    map[string]any{"source": "integration-test"},
	}, nil)
	if activateRes.Code != http.StatusOK {
		t.Fatalf("expected activate status %d, got %d body=%s", http.StatusOK, activateRes.Code, activateRes.Body.String())
	}

	activated := decodeJSONResponse[publicLicenseResponse](t, activateRes)
	if !activated.Valid {
		t.Fatalf("expected activation to be valid, got %+v", activated)
	}
	if activated.License == nil || activated.License.Status != "active" || activated.License.Seats.Used != 1 {
		t.Fatalf("expected active license with one seat used, got %s", activateRes.Body.String())
	}
	if strings.Contains(activateRes.Body.String(), "metadata") {
		t.Fatalf("public response must not expose metadata: %s", activateRes.Body.String())
	}

	validateBeforeRevokeRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/validate", validateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
	}, nil)
	if validateBeforeRevokeRes.Code != http.StatusOK {
		t.Fatalf("expected validate-before-revoke status %d, got %d body=%s", http.StatusOK, validateBeforeRevokeRes.Code, validateBeforeRevokeRes.Body.String())
	}

	validatedBeforeRevoke := decodeJSONResponse[publicLicenseResponse](t, validateBeforeRevokeRes)
	if !validatedBeforeRevoke.Valid || validatedBeforeRevoke.License.Status != "active" {
		t.Fatalf("expected valid active license before revoke, got %+v", validatedBeforeRevoke)
	}

	revokeRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/revoke", revokeRequest{
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

	validateAfterRevokeRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/validate", validateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
	}, nil)
	if validateAfterRevokeRes.Code != http.StatusOK {
		t.Fatalf("expected validate-after-revoke status %d, got %d body=%s", http.StatusOK, validateAfterRevokeRes.Code, validateAfterRevokeRes.Body.String())
	}

	validatedAfterRevoke := decodeJSONResponse[publicLicenseResponse](t, validateAfterRevokeRes)
	if validatedAfterRevoke.Valid {
		t.Fatalf("expected revoked license to be invalid, got %+v", validatedAfterRevoke)
	}
	if validatedAfterRevoke.License == nil || validatedAfterRevoke.License.Status != "revoked" {
		t.Fatalf("expected revoked status after revoke, got %s", validateAfterRevokeRes.Body.String())
	}
	if validatedAfterRevoke.Reason == nil || *validatedAfterRevoke.Reason != "revoked" {
		t.Fatalf("expected revoked reason after revoke, got %s", validateAfterRevokeRes.Body.String())
	}

	adminHeaders := map[string]string{"Authorization": "Bearer " + managementKey}
	licensePath := "/management/licenses/" + generated.ID

	// Restore the revoked license and raise its seat count; the existing device comes back with it.
	restoreRes := doJSONRequest(t, handler, http.MethodPatch, licensePath, map[string]any{
		"revoked":         false,
		"max_activations": 5,
		"expires_at":      nil,
	}, adminHeaders)
	if restoreRes.Code != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d body=%s", restoreRes.Code, restoreRes.Body.String())
	}
	restored := decodeJSONResponse[licenseDetailResponse](t, restoreRes)
	if restored.Status != "active" || restored.MaxActivations != 5 || restored.RevokedAt != nil || restored.ExpiresAt != nil {
		t.Fatalf("unexpected restored license: %s", restoreRes.Body.String())
	}
	if len(restored.Activations) != 1 || restored.Activations[0].Fingerprint != fingerprint {
		t.Fatalf("expected the original activation to be listed, got %s", restoreRes.Body.String())
	}

	// An operator frees the device's seat; the device is then no longer valid.
	removeRes := doJSONRequest(t, handler, http.MethodDelete, fmt.Sprintf("%s/activations/%d", licensePath, restored.Activations[0].ID), nil, adminHeaders)
	if removeRes.Code != http.StatusOK {
		t.Fatalf("expected remove activation status 200, got %d body=%s", removeRes.Code, removeRes.Body.String())
	}
	afterRemove := decodeJSONResponse[licenseDetailResponse](t, removeRes)
	if afterRemove.Status != "inactive" || afterRemove.ActiveSeats != 0 || len(afterRemove.Activations) != 0 {
		t.Fatalf("expected inactive license with no seats, got %s", removeRes.Body.String())
	}

	validateAfterRemoveRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/validate", validateRequest{
		LicenseKey:  generated.LicenseKey,
		Fingerprint: fingerprint,
	}, nil)
	validatedAfterRemove := decodeJSONResponse[publicLicenseResponse](t, validateAfterRemoveRes)
	if validatedAfterRemove.Valid || validatedAfterRemove.Reason == nil || *validatedAfterRemove.Reason != "fingerprint_not_active" {
		t.Fatalf("expected fingerprint_not_active after removal, got %s", validateAfterRemoveRes.Body.String())
	}

	// A slug's product is returned to apps so they can reject keys meant for another product.
	productSlug := "integration-product-" + strings.ToLower(flowSuffix)
	createSlugRes := doJSONRequest(t, handler, http.MethodPost, "/management/slugs", map[string]any{
		"name": productSlug, "max_activations": 1, "expiration_type": "forever", "product": "integration-app",
	}, adminHeaders)
	if createSlugRes.Code != http.StatusCreated {
		t.Fatalf("expected create slug 201, got %d body=%s", createSlugRes.Code, createSlugRes.Body.String())
	}
	productLicenseRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/generate", generateRequest{Slug: productSlug}, adminHeaders)
	productLicense := decodeJSONResponse[generateResponse](t, productLicenseRes)
	productActivateRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/activate", activateRequest{
		LicenseKey:  productLicense.LicenseKey,
		Fingerprint: fingerprint,
	}, nil)
	productActivated := decodeJSONResponse[publicLicenseResponse](t, productActivateRes)
	if productActivated.License == nil || productActivated.License.Product == nil || *productActivated.License.Product != "integration-app" {
		t.Fatalf("expected license.product integration-app, got %s", productActivateRes.Body.String())
	}

	// Webhook endpoints get a signing secret that is shown once and can be rotated.
	createHookRes := doJSONRequest(t, handler, http.MethodPost, "/management/webhooks", map[string]any{
		"name": "integration-" + flowSuffix, "url": "https://example.com/hooks", "events": []string{"license.generated"}, "enabled": false,
	}, adminHeaders)
	createdHook := decodeJSONResponse[webhookEndpointResponse](t, createHookRes)
	if createHookRes.Code != http.StatusCreated || !strings.HasPrefix(createdHook.Secret, "whsec_") {
		t.Fatalf("expected a whsec_ secret on create, got %d %s", createHookRes.Code, createHookRes.Body.String())
	}
	listHooksRes := doJSONRequest(t, handler, http.MethodGet, "/management/webhooks", nil, adminHeaders)
	if strings.Contains(listHooksRes.Body.String(), createdHook.Secret) || strings.Contains(listHooksRes.Body.String(), `"secret"`) {
		t.Fatalf("webhook secrets must never be listed: %s", listHooksRes.Body.String())
	}
	rotateRes := doJSONRequest(t, handler, http.MethodPost, fmt.Sprintf("/management/webhooks/%d/rotate-secret", createdHook.ID), nil, adminHeaders)
	rotated := decodeJSONResponse[rotateWebhookSecretResponse](t, rotateRes)
	if rotateRes.Code != http.StatusOK || !strings.HasPrefix(rotated.Secret, "whsec_") || rotated.Secret == createdHook.Secret {
		t.Fatalf("expected a new secret from rotate, got %d %s", rotateRes.Code, rotateRes.Body.String())
	}
	doJSONRequest(t, handler, http.MethodDelete, fmt.Sprintf("/management/webhooks/%d", createdHook.ID), nil, adminHeaders)

	// Unknown keys get the same 200 shape with reason not_found.
	unknownRes := doJSONRequest(t, handler, http.MethodPost, "/api/v1/validate", validateRequest{
		LicenseKey:  "DOES-NOT-EXIST-" + flowSuffix,
		Fingerprint: fingerprint,
	}, nil)
	if unknownRes.Code != http.StatusOK {
		t.Fatalf("expected unknown key status 200, got %d", unknownRes.Code)
	}
	unknown := decodeJSONResponse[publicLicenseResponse](t, unknownRes)
	if unknown.Valid || unknown.License != nil || unknown.Reason == nil || *unknown.Reason != "not_found" {
		t.Fatalf("expected not_found response, got %s", unknownRes.Body.String())
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
