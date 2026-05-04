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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"simple-license-server/internal/offlinejwt"
	"simple-license-server/internal/storage"
)

type stubService struct {
	enqueueWebhookEventFn      func(ctx context.Context, eventType string, payload map[string]any) error
	isAuthorizedServerAPIKeyFn func(ctx context.Context, candidate string) (bool, error)
	pingFn                     func(ctx context.Context) error
	generateLicenseFn          func(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error)
	generateIdempotentFn       func(ctx context.Context, endpoint, idemKey, requestHash, slugName string, metadata map[string]any) (storage.GeneratedLicense, json.RawMessage, bool, error)
	listLicensesFn             func(ctx context.Context, params storage.LicenseListParams) (storage.LicenseListResult, error)
	revokeLicenseFn            func(ctx context.Context, licenseKey string) (storage.RevokeResult, error)
	activateLicenseFn          func(ctx context.Context, licenseKey, fingerprint string, metadata map[string]any) (storage.ActivationResult, error)
	validateLicenseFn          func(ctx context.Context, licenseKey, fingerprint string) (storage.ValidationResult, error)
	deactivateLicenseFn        func(ctx context.Context, licenseKey, fingerprint, reason string) (storage.DeactivationResult, error)
	listSlugsFn                func(ctx context.Context, includeArchived bool) ([]storage.SlugRecord, error)
	getSlugByNameFn            func(ctx context.Context, name string) (storage.SlugRecord, error)
	createSlugFn               func(ctx context.Context, params storage.CreateSlugParams) (storage.SlugRecord, error)
	updateSlugByNameFn         func(ctx context.Context, name string, params storage.UpdateSlugParams) (storage.SlugRecord, error)
	deleteSlugByNameFn         func(ctx context.Context, name string) error
	listAPIKeysFn              func(ctx context.Context) ([]storage.APIKeyRecord, error)
	createAPIKeyFn             func(ctx context.Context, params storage.CreateAPIKeyParams) (storage.CreatedAPIKey, error)
	revokeAPIKeyFn             func(ctx context.Context, id int64) (storage.APIKeyRecord, error)
	listWebhooksFn             func(ctx context.Context) ([]storage.WebhookEndpoint, error)
	listWebhookDeliveriesFn    func(ctx context.Context, limit int) ([]storage.WebhookDeliveryLog, error)
	createWebhookFn            func(ctx context.Context, params storage.CreateWebhookEndpointParams) (storage.WebhookEndpoint, error)
	updateWebhookFn            func(ctx context.Context, id int64, params storage.UpdateWebhookEndpointParams) (storage.WebhookEndpoint, error)
	deleteWebhookFn            func(ctx context.Context, id int64) error
	listSigningKeysFn          func(ctx context.Context) ([]storage.SigningKeyRecord, error)
	listPublicSigningKeysFn    func(ctx context.Context) ([]storage.SigningKeyRecord, error)
	getActiveSigningKeyFn      func(ctx context.Context) (storage.SigningKeyRecord, error)
	createSigningKeyFn         func(ctx context.Context, params storage.CreateSigningKeyParams) (storage.SigningKeyRecord, error)
	activateSigningKeyFn       func(ctx context.Context, id int64) (storage.SigningKeyRecord, error)
	retireSigningKeyFn         func(ctx context.Context, id int64) (storage.SigningKeyRecord, error)
}

func (s stubService) EnqueueWebhookEvent(ctx context.Context, eventType string, payload map[string]any) error {
	if s.enqueueWebhookEventFn != nil {
		return s.enqueueWebhookEventFn(ctx, eventType, payload)
	}

	return nil
}

func (s stubService) IsAuthorizedServerAPIKey(ctx context.Context, candidate string) (bool, error) {
	if s.isAuthorizedServerAPIKeyFn != nil {
		return s.isAuthorizedServerAPIKeyFn(ctx, candidate)
	}
	return false, nil
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

func (s stubService) ListLicenses(ctx context.Context, params storage.LicenseListParams) (storage.LicenseListResult, error) {
	if s.listLicensesFn != nil {
		return s.listLicensesFn(ctx, params)
	}
	return storage.LicenseListResult{}, nil
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

func (s stubService) ListSlugs(ctx context.Context, includeArchived bool) ([]storage.SlugRecord, error) {
	if s.listSlugsFn != nil {
		return s.listSlugsFn(ctx, includeArchived)
	}
	return []storage.SlugRecord{}, nil
}

func (s stubService) GetSlugByName(ctx context.Context, name string) (storage.SlugRecord, error) {
	if s.getSlugByNameFn != nil {
		return s.getSlugByNameFn(ctx, name)
	}
	return storage.SlugRecord{}, nil
}

func (s stubService) CreateSlug(ctx context.Context, params storage.CreateSlugParams) (storage.SlugRecord, error) {
	if s.createSlugFn != nil {
		return s.createSlugFn(ctx, params)
	}
	return storage.SlugRecord{}, nil
}

func (s stubService) UpdateSlugByName(ctx context.Context, name string, params storage.UpdateSlugParams) (storage.SlugRecord, error) {
	if s.updateSlugByNameFn != nil {
		return s.updateSlugByNameFn(ctx, name, params)
	}
	return storage.SlugRecord{}, nil
}

func (s stubService) DeleteSlugByName(ctx context.Context, name string) error {
	if s.deleteSlugByNameFn != nil {
		return s.deleteSlugByNameFn(ctx, name)
	}
	return nil
}

func (s stubService) ListAPIKeys(ctx context.Context) ([]storage.APIKeyRecord, error) {
	if s.listAPIKeysFn != nil {
		return s.listAPIKeysFn(ctx)
	}
	return []storage.APIKeyRecord{}, nil
}

func (s stubService) CreateAPIKey(ctx context.Context, params storage.CreateAPIKeyParams) (storage.CreatedAPIKey, error) {
	if s.createAPIKeyFn != nil {
		return s.createAPIKeyFn(ctx, params)
	}
	return storage.CreatedAPIKey{}, nil
}

func (s stubService) RevokeAPIKey(ctx context.Context, id int64) (storage.APIKeyRecord, error) {
	if s.revokeAPIKeyFn != nil {
		return s.revokeAPIKeyFn(ctx, id)
	}
	return storage.APIKeyRecord{}, nil
}

func (s stubService) ListWebhookEndpoints(ctx context.Context) ([]storage.WebhookEndpoint, error) {
	if s.listWebhooksFn != nil {
		return s.listWebhooksFn(ctx)
	}
	return []storage.WebhookEndpoint{}, nil
}

func (s stubService) ListWebhookDeliveries(ctx context.Context, limit int) ([]storage.WebhookDeliveryLog, error) {
	if s.listWebhookDeliveriesFn != nil {
		return s.listWebhookDeliveriesFn(ctx, limit)
	}
	return []storage.WebhookDeliveryLog{}, nil
}

func (s stubService) CreateWebhookEndpoint(ctx context.Context, params storage.CreateWebhookEndpointParams) (storage.WebhookEndpoint, error) {
	if s.createWebhookFn != nil {
		return s.createWebhookFn(ctx, params)
	}
	return storage.WebhookEndpoint{}, nil
}

func (s stubService) UpdateWebhookEndpoint(ctx context.Context, id int64, params storage.UpdateWebhookEndpointParams) (storage.WebhookEndpoint, error) {
	if s.updateWebhookFn != nil {
		return s.updateWebhookFn(ctx, id, params)
	}
	return storage.WebhookEndpoint{}, nil
}

func (s stubService) DeleteWebhookEndpoint(ctx context.Context, id int64) error {
	if s.deleteWebhookFn != nil {
		return s.deleteWebhookFn(ctx, id)
	}
	return nil
}

func (s stubService) ListSigningKeys(ctx context.Context) ([]storage.SigningKeyRecord, error) {
	if s.listSigningKeysFn != nil {
		return s.listSigningKeysFn(ctx)
	}
	return []storage.SigningKeyRecord{}, nil
}

func (s stubService) ListPublicSigningKeys(ctx context.Context) ([]storage.SigningKeyRecord, error) {
	if s.listPublicSigningKeysFn != nil {
		return s.listPublicSigningKeysFn(ctx)
	}
	return []storage.SigningKeyRecord{}, nil
}

func (s stubService) GetActiveSigningKey(ctx context.Context) (storage.SigningKeyRecord, error) {
	if s.getActiveSigningKeyFn != nil {
		return s.getActiveSigningKeyFn(ctx)
	}
	return storage.SigningKeyRecord{}, storage.ErrNotFound
}

func (s stubService) CreateSigningKey(ctx context.Context, params storage.CreateSigningKeyParams) (storage.SigningKeyRecord, error) {
	if s.createSigningKeyFn != nil {
		return s.createSigningKeyFn(ctx, params)
	}
	return storage.SigningKeyRecord{}, nil
}

func (s stubService) ActivateSigningKey(ctx context.Context, id int64) (storage.SigningKeyRecord, error) {
	if s.activateSigningKeyFn != nil {
		return s.activateSigningKeyFn(ctx, id)
	}
	return storage.SigningKeyRecord{}, nil
}

func (s stubService) RetireSigningKey(ctx context.Context, id int64) (storage.SigningKeyRecord, error) {
	if s.retireSigningKeyFn != nil {
		return s.retireSigningKeyFn(ctx, id)
	}
	return storage.SigningKeyRecord{}, nil
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
		isAuthorizedServerAPIKeyFn: func(ctx context.Context, candidate string) (bool, error) {
			return candidate == "server_key_dev_123", nil
		},
		generateLicenseFn: func(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error) {
			called = true
			if _, ok := ctx.Deadline(); ok {
				deadlineSeen = true
			}

			return storage.GeneratedLicense{
				LicenseKey: "A1B2C3-D4E5F6-AB12CD-34EF56-7890AB",
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

func TestGenerateAcceptsManagementKeyForProvisioning(t *testing.T) {
	createdAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	called := false
	s := NewServer(stubService{
		isAuthorizedServerAPIKeyFn: func(ctx context.Context, candidate string) (bool, error) {
			return candidate == "server_key_db_123456", nil
		},
		generateLicenseFn: func(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error) {
			called = true
			return storage.GeneratedLicense{
				LicenseKey: "LNC-8821-X99B-4421",
				Slug:       slugName,
				Status:     "inactive",
				Metadata:   metadata,
				CreatedAt:  createdAt,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(`{"slug":"default","metadata":{"email":"user@example.com"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer management_key_dev_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatalf("expected service GenerateLicense to be called")
	}
}

func TestRevokeAcceptsManagementKeyForProvisioning(t *testing.T) {
	revokedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	called := false
	s := NewServer(stubService{
		isAuthorizedServerAPIKeyFn: func(ctx context.Context, candidate string) (bool, error) {
			return candidate == "server_key_db_123456", nil
		},
		revokeLicenseFn: func(ctx context.Context, licenseKey string) (storage.RevokeResult, error) {
			called = true
			return storage.RevokeResult{
				Valid:      false,
				Status:     "revoked",
				LicenseKey: licenseKey,
				RevokedAt:  revokedAt,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(`{"license_key":"LNC-8821-X99B-4421"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer management_key_dev_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatalf("expected service RevokeLicense to be called")
	}
}

func TestValidateRefreshesOfflineTokenWhenSlugAllowsOffline(t *testing.T) {
	secret := "test_offline_signing_secret_32_chars"
	keyPair, err := offlinejwt.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	encryptedPrivateKey, err := offlinejwt.EncryptPrivateKey(keyPair.PrivatePEM, secret)
	if err != nil {
		t.Fatalf("encrypt private key: %v", err)
	}

	s := NewServerWithOptions(stubService{
		validateLicenseFn: func(ctx context.Context, licenseKey, fingerprint string) (storage.ValidationResult, error) {
			return storage.ValidationResult{
				Valid:                       true,
				Status:                      "active",
				LicenseID:                   "8c50f4f7-761d-4bf1-8e09-27a5f7ea0a12",
				Slug:                        "default",
				OfflineEnabled:              true,
				OfflineTokenLifetimeSeconds: 86400,
			}, nil
		},
		getActiveSigningKeyFn: func(ctx context.Context) (storage.SigningKeyRecord, error) {
			return storage.SigningKeyRecord{
				Kid:                 keyPair.Kid,
				Algorithm:           keyPair.Algorithm,
				Status:              "active",
				PrivateKeyEncrypted: encryptedPrivateKey,
				PublicKeyPEM:        keyPair.PublicPEM,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, Options{
		RequestTimeout:     15 * time.Second,
		RateLimitEnabled:   false,
		OfflineSigningKey:  secret,
		OfflineTokenIssuer: "test-issuer",
	})

	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(`{"license_key":"LNC-8821-X99B-4421","fingerprint":"device-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var response validateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("expected refreshed offline token")
	}
	if strings.Count(response.Token, ".") != 2 {
		t.Fatalf("expected JWT-shaped token, got %q", response.Token)
	}
}

func TestManagementEndpointRequiresManagementKey(t *testing.T) {
	s := NewServer(stubService{}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/management/api-keys", nil)
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestManagementEndpointWithManagementKey(t *testing.T) {
	s := NewServer(stubService{}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/management/api-keys", nil)
	req.Header.Set("Authorization", "Bearer management_key_dev_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "api_keys") {
		t.Fatalf("expected api keys payload, got %s", rr.Body.String())
	}
}

func TestListLicensesUsesPaginationSearchAndStatus(t *testing.T) {
	createdAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	var gotParams storage.LicenseListParams
	s := NewServer(stubService{
		listLicensesFn: func(ctx context.Context, params storage.LicenseListParams) (storage.LicenseListResult, error) {
			gotParams = params
			return storage.LicenseListResult{
				Licenses: []storage.LicenseRow{
					{
						ID:             "license-id-1",
						Key:            "LNC-8821-X99B-4421",
						Status:         "active",
						SlugName:       "default",
						Metadata:       map[string]any{"email": "user@example.com"},
						MaxActivations: 3,
						ActiveSeats:    2,
						CreatedAt:      createdAt,
					},
				},
				Total: 42,
				Counts: storage.LicenseStatusCounts{
					Total:    1284,
					Active:   942,
					Inactive: 330,
					Revoked:  12,
				},
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/management/licenses?page=2&page_size=25&q=LNC&status=active", nil)
	req.Header.Set("Authorization", "Bearer management_key_dev_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if gotParams.Page != 2 || gotParams.PageSize != 25 || gotParams.Search != "LNC" || gotParams.Status != "active" {
		t.Fatalf("unexpected list params: %+v", gotParams)
	}
	if !strings.Contains(rr.Body.String(), "LNC-8821-X99B-4421") || !strings.Contains(rr.Body.String(), `"total_pages":2`) {
		t.Fatalf("expected license list payload, got %s", rr.Body.String())
	}
}

func TestListLicensesRejectsInvalidStatus(t *testing.T) {
	s := NewServer(stubService{}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/management/licenses?status=pending", nil)
	req.Header.Set("Authorization", "Bearer management_key_dev_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestListWebhookDeliveriesUsesLimit(t *testing.T) {
	now := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	statusCode := 500
	lastError := "webhook responded with status 500"
	var gotLimit int
	s := NewServer(stubService{
		listWebhookDeliveriesFn: func(ctx context.Context, limit int) ([]storage.WebhookDeliveryLog, error) {
			gotLimit = limit
			return []storage.WebhookDeliveryLog{
				{
					ID:                 42,
					EndpointID:         7,
					EndpointName:       "audit-sync",
					EndpointURL:        "https://example.com/hooks/license",
					EventType:          "license.generated",
					Status:             "failed",
					Attempts:           3,
					LastResponseStatus: &statusCode,
					LastError:          &lastError,
					NextAttemptAt:      now,
					CreatedAt:          now,
					UpdatedAt:          now,
				},
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), map[string]struct{}{"management_key_dev_123456": {}}, 15*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/management/webhooks/deliveries?limit=5", nil)
	req.Header.Set("Authorization", "Bearer management_key_dev_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if gotLimit != 5 {
		t.Fatalf("expected limit 5, got %d", gotLimit)
	}
	if !strings.Contains(rr.Body.String(), "audit-sync") || !strings.Contains(rr.Body.String(), "license.generated") {
		t.Fatalf("expected delivery log payload, got %s", rr.Body.String())
	}
}

func TestManagementEndpointRejectsProvisioningKey(t *testing.T) {
	s := NewServer(
		stubService{
			isAuthorizedServerAPIKeyFn: func(ctx context.Context, candidate string) (bool, error) {
				return candidate == "server_key_db_123456", nil
			},
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		map[string]struct{}{"management_key_dev_123456": {}},
		15*time.Second,
	)

	req := httptest.NewRequest(http.MethodGet, "/management/api-keys", nil)
	req.Header.Set("Authorization", "Bearer server_key_db_123456")
	rr := httptest.NewRecorder()

	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d body=%s", rr.Code, rr.Body.String())
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

func TestUIRoutesDisabledByDefault(t *testing.T) {
	s := NewServerWithOptions(
		stubService{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		map[string]struct{}{"management_key_dev_123456": {}},
		Options{RequestTimeout: 15 * time.Second},
	)

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rr := httptest.NewRecorder()
	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 when UI disabled, got %d", rr.Code)
	}
}

func TestUIRoutesServeIndexWhenEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html><body>ui ok</body></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	s := NewServerWithOptions(
		stubService{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		map[string]struct{}{"management_key_dev_123456": {}},
		Options{RequestTimeout: 15 * time.Second, UIEnabled: true, UIDistDir: tmpDir},
	)

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rr := httptest.NewRecorder()
	s.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "ui ok") {
		t.Fatalf("expected UI index content, got: %s", rr.Body.String())
	}
}
