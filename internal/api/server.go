package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	slugdomain "simple-license-server/internal/domain/slug"
	"simple-license-server/internal/storage"
	"simple-license-server/internal/version"

	"golang.org/x/crypto/bcrypt"
)

const (
	maxBodyBytes            = 1 << 20
	maxSlugLength           = 128
	maxLicenseKeyLength     = 128
	maxFingerprintLength    = 256
	maxReasonLength         = 256
	maxIdempotencyKeyLength = 255
	maxMetadataEntries      = 64
	maxMetadataDepth        = 5
	maxMetadataNodes        = 512
	maxAPIKeyNameLength     = 128
	maxWebhookNameLength    = 128
	maxWebhookURLLength     = 2048
	generateEndpointKey     = "/generate"
)

const (
	webhookEventLicenseGenerated        = "license.generated"
	webhookEventLicenseActivated        = "license.activated"
	webhookEventLicenseDeactivated      = "license.deactivated"
	webhookEventLicenseValidated        = "license.validated"
	webhookEventLicenseValidationFailed = "license.validation_failed"
	webhookEventLicenseRevoked          = "license.revoked"
)

var supportedWebhookEvents = []string{
	webhookEventLicenseGenerated,
	webhookEventLicenseActivated,
	webhookEventLicenseDeactivated,
	webhookEventLicenseValidated,
	webhookEventLicenseValidationFailed,
	webhookEventLicenseRevoked,
}

var supportedWebhookEventSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(supportedWebhookEvents))
	for _, event := range supportedWebhookEvents {
		m[event] = struct{}{}
	}
	return m
}()

var supportedWebhookEventOrder = func() map[string]int {
	m := make(map[string]int, len(supportedWebhookEvents))
	for idx, event := range supportedWebhookEvents {
		m[event] = idx
	}
	return m
}()

type Server struct {
	service                licenseService
	logger                 *slog.Logger
	managementAPIKeyHashes [][]byte
	requestTimeout         time.Duration
	rateLimiter            *ipRateLimiter
}

type Options struct {
	RequestTimeout        time.Duration
	RateLimitEnabled      bool
	RateLimitGlobalRPS    float64
	RateLimitGlobalBurst  int
	RateLimitPerIPRPS     float64
	RateLimitPerIPBurst   int
	RateLimitIPTTL        time.Duration
	RateLimitMaxIPEntries int
	TrustProxyHeaders     bool
}

func DefaultOptions() Options {
	return Options{
		RequestTimeout:        15 * time.Second,
		RateLimitEnabled:      true,
		RateLimitGlobalRPS:    100,
		RateLimitGlobalBurst:  200,
		RateLimitPerIPRPS:     20,
		RateLimitPerIPBurst:   40,
		RateLimitIPTTL:        10 * time.Minute,
		RateLimitMaxIPEntries: 10000,
		TrustProxyHeaders:     false,
	}
}

type licenseService interface {
	Ping(ctx context.Context) error
	EnqueueWebhookEvent(ctx context.Context, eventType string, payload map[string]any) error
	IsAuthorizedServerAPIKey(ctx context.Context, candidate string) (bool, error)
	GenerateLicenseIdempotent(ctx context.Context, endpoint, idemKey, requestHash, slugName string, metadata map[string]any) (storage.GeneratedLicense, json.RawMessage, bool, error)
	GenerateLicense(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error)
	RevokeLicense(ctx context.Context, licenseKey string) (storage.RevokeResult, error)
	ActivateLicense(ctx context.Context, licenseKey, fingerprint string, metadata map[string]any) (storage.ActivationResult, error)
	ValidateLicense(ctx context.Context, licenseKey, fingerprint string) (storage.ValidationResult, error)
	DeactivateLicense(ctx context.Context, licenseKey, fingerprint, reason string) (storage.DeactivationResult, error)
	ListSlugs(ctx context.Context) ([]storage.SlugRecord, error)
	GetSlugByName(ctx context.Context, name string) (storage.SlugRecord, error)
	CreateSlug(ctx context.Context, params storage.CreateSlugParams) (storage.SlugRecord, error)
	UpdateSlugByName(ctx context.Context, name string, params storage.UpdateSlugParams) (storage.SlugRecord, error)
	DeleteSlugByName(ctx context.Context, name string) error
	ListAPIKeys(ctx context.Context) ([]storage.APIKeyRecord, error)
	CreateAPIKey(ctx context.Context, params storage.CreateAPIKeyParams) (storage.CreatedAPIKey, error)
	RevokeAPIKey(ctx context.Context, id int64) (storage.APIKeyRecord, error)
	ListWebhookEndpoints(ctx context.Context) ([]storage.WebhookEndpoint, error)
	CreateWebhookEndpoint(ctx context.Context, params storage.CreateWebhookEndpointParams) (storage.WebhookEndpoint, error)
	UpdateWebhookEndpoint(ctx context.Context, id int64, params storage.UpdateWebhookEndpointParams) (storage.WebhookEndpoint, error)
	DeleteWebhookEndpoint(ctx context.Context, id int64) error
}

func NewServer(service licenseService, logger *slog.Logger, managementAPIKeys map[string]struct{}, requestTimeout time.Duration) *Server {
	opts := DefaultOptions()
	opts.RequestTimeout = requestTimeout
	return NewServerWithOptions(service, logger, managementAPIKeys, opts)
}

func NewServerWithOptions(service licenseService, logger *slog.Logger, managementAPIKeys map[string]struct{}, opts Options) *Server {
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = DefaultOptions().RequestTimeout
	}

	if opts.RateLimitGlobalRPS <= 0 {
		opts.RateLimitGlobalRPS = DefaultOptions().RateLimitGlobalRPS
	}

	if opts.RateLimitGlobalBurst <= 0 {
		opts.RateLimitGlobalBurst = DefaultOptions().RateLimitGlobalBurst
	}

	if opts.RateLimitPerIPRPS <= 0 {
		opts.RateLimitPerIPRPS = DefaultOptions().RateLimitPerIPRPS
	}

	if opts.RateLimitPerIPBurst <= 0 {
		opts.RateLimitPerIPBurst = DefaultOptions().RateLimitPerIPBurst
	}

	if opts.RateLimitIPTTL <= 0 {
		opts.RateLimitIPTTL = DefaultOptions().RateLimitIPTTL
	}

	if opts.RateLimitMaxIPEntries <= 0 {
		opts.RateLimitMaxIPEntries = DefaultOptions().RateLimitMaxIPEntries
	}

	return &Server{
		service:                service,
		logger:                 logger,
		managementAPIKeyHashes: hashManagementAPIKeys(managementAPIKeys),
		requestTimeout:         opts.RequestTimeout,
		rateLimiter: newIPRateLimiter(rateLimiterConfig{
			enabled:           opts.RateLimitEnabled,
			globalRPS:         opts.RateLimitGlobalRPS,
			globalBurst:       opts.RateLimitGlobalBurst,
			perIPRPS:          opts.RateLimitPerIPRPS,
			perIPBurst:        opts.RateLimitPerIPBurst,
			ipTTL:             opts.RateLimitIPTTL,
			maxIPEntries:      opts.RateLimitMaxIPEntries,
			trustProxyHeaders: opts.TrustProxyHeaders,
		}),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.Handle("POST /generate", s.requireProvisioningKey(http.HandlerFunc(s.handleGenerate)))
	mux.Handle("POST /revoke", s.requireProvisioningKey(http.HandlerFunc(s.handleRevoke)))
	mux.HandleFunc("POST /activate", s.handleActivate)
	mux.HandleFunc("POST /validate", s.handleValidate)
	mux.HandleFunc("POST /deactivate", s.handleDeactivate)
	mux.Handle("GET /management/slugs", s.requireManagementKey(http.HandlerFunc(s.handleListSlugs)))
	mux.Handle("POST /management/slugs", s.requireManagementKey(http.HandlerFunc(s.handleCreateSlug)))
	mux.Handle("GET /management/slugs/{name}", s.requireManagementKey(http.HandlerFunc(s.handleGetSlug)))
	mux.Handle("PATCH /management/slugs/{name}", s.requireManagementKey(http.HandlerFunc(s.handleUpdateSlug)))
	mux.Handle("DELETE /management/slugs/{name}", s.requireManagementKey(http.HandlerFunc(s.handleDeleteSlug)))
	mux.Handle("GET /management/api-keys", s.requireManagementKey(http.HandlerFunc(s.handleListAPIKeys)))
	mux.Handle("POST /management/api-keys", s.requireManagementKey(http.HandlerFunc(s.handleCreateAPIKey)))
	mux.Handle("POST /management/api-keys/{id}/revoke", s.requireManagementKey(http.HandlerFunc(s.handleRevokeAPIKey)))
	mux.Handle("GET /management/webhooks", s.requireManagementKey(http.HandlerFunc(s.handleListWebhooks)))
	mux.Handle("POST /management/webhooks", s.requireManagementKey(http.HandlerFunc(s.handleCreateWebhook)))
	mux.Handle("PATCH /management/webhooks/{id}", s.requireManagementKey(http.HandlerFunc(s.handleUpdateWebhook)))
	mux.Handle("DELETE /management/webhooks/{id}", s.requireManagementKey(http.HandlerFunc(s.handleDeleteWebhook)))

	handler := s.requestTimeoutMiddleware(mux)
	handler = s.rateLimitMiddleware(handler)
	handler = s.loggingMiddleware(handler)
	handler = s.recoverMiddleware(handler)
	handler = s.securityHeadersMiddleware(handler)

	return handler
}

type generateRequest struct {
	Slug     string         `json:"slug"`
	Metadata map[string]any `json:"metadata"`
}

type generateResponse struct {
	LicenseKey string         `json:"license_key"`
	Slug       string         `json:"slug"`
	Status     string         `json:"status"`
	Metadata   map[string]any `json:"metadata"`
	ExpiresAt  *time.Time     `json:"expires_at"`
	CreatedAt  time.Time      `json:"created_at"`
}

type revokeRequest struct {
	LicenseKey string `json:"license_key"`
}

type revokeResponse struct {
	Valid      bool      `json:"valid"`
	Status     string    `json:"status"`
	LicenseKey string    `json:"license_key"`
	RevokedAt  time.Time `json:"revoked_at"`
}

type activateRequest struct {
	LicenseKey  string         `json:"license_key"`
	Fingerprint string         `json:"fingerprint"`
	Metadata    map[string]any `json:"metadata"`
}

type activateResponse struct {
	Valid       bool       `json:"valid"`
	Status      string     `json:"status"`
	LicenseKey  string     `json:"license_key"`
	Fingerprint string     `json:"fingerprint"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Reason      string     `json:"reason,omitempty"`
}

type validateRequest struct {
	LicenseKey  string `json:"license_key"`
	Fingerprint string `json:"fingerprint"`
}

type validateResponse struct {
	Valid     bool       `json:"valid"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
	Reason    string     `json:"reason,omitempty"`
}

type deactivateRequest struct {
	LicenseKey  string `json:"license_key"`
	Fingerprint string `json:"fingerprint"`
	Reason      string `json:"reason"`
}

type deactivateResponse struct {
	Valid          bool       `json:"valid"`
	Released       bool       `json:"released"`
	Status         string     `json:"status"`
	ActiveSeats    int        `json:"active_seats"`
	MaxActivations int        `json:"max_activations"`
	ExpiresAt      *time.Time `json:"expires_at"`
}

type apiKeyResponse struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Hint      string     `json:"hint"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

type listAPIKeysResponse struct {
	APIKeys []apiKeyResponse `json:"api_keys"`
}

type createAPIKeyRequest struct {
	Name string `json:"name"`
}

type createAPIKeyResponse struct {
	APIKey    string    `json:"api_key"`
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type webhookEndpointResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type listWebhookEndpointsResponse struct {
	Webhooks []webhookEndpointResponse `json:"webhooks"`
}

type createWebhookRequest struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

type updateWebhookRequest struct {
	Name    *string   `json:"name"`
	URL     *string   `json:"url"`
	Events  *[]string `json:"events"`
	Enabled *bool     `json:"enabled"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := s.service.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "database unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": version.Current})
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	if err := requireField(req.Slug, "slug"); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.Slug, "slug", maxSlugLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateSlugName(req.Slug); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if err := validateMetadata(req.Metadata); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if err := validateOptionalFieldLength(idempotencyKey, "idempotency key", maxIdempotencyKeyLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	requestHash, err := hashGenerateRequest(req)
	if err != nil {
		s.writeUnexpectedError(w, "failed to hash generate request", err)
		return
	}

	if idempotencyKey != "" {
		generated, responseBody, replayed, err := s.service.GenerateLicenseIdempotent(
			r.Context(),
			generateEndpointKey,
			idempotencyKey,
			requestHash,
			req.Slug,
			req.Metadata,
		)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "slug not found"})
			case errors.Is(err, storage.ErrConflict):
				writeJSON(w, http.StatusConflict, errorResponse{Error: "idempotency key reuse with different payload"})
			case errors.Is(err, storage.ErrInProgress):
				writeJSON(w, http.StatusConflict, errorResponse{Error: "idempotent request is still in progress"})
			default:
				s.writeUnexpectedError(w, "failed idempotent license generation", err)
			}
			return
		}

		if !replayed {
			s.emitWebhookEvent(r.Context(), webhookEventLicenseGenerated, map[string]any{
				"license_key": generated.LicenseKey,
				"slug":        generated.Slug,
				"status":      generated.Status,
				"metadata":    generated.Metadata,
				"expires_at":  generated.ExpiresAt,
				"created_at":  generated.CreatedAt,
			})
		}

		writeRawJSON(w, http.StatusOK, responseBody)
		return
	}

	generated, err := s.service.GenerateLicense(r.Context(), req.Slug, req.Metadata)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "slug not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to generate license", err)
		return
	}

	resp := generateResponse{
		LicenseKey: generated.LicenseKey,
		Slug:       generated.Slug,
		Status:     generated.Status,
		Metadata:   generated.Metadata,
		ExpiresAt:  generated.ExpiresAt,
		CreatedAt:  generated.CreatedAt,
	}

	responseBody, err := json.Marshal(resp)
	if err != nil {
		s.writeUnexpectedError(w, "failed to marshal generate response", err)
		return
	}

	s.emitWebhookEvent(r.Context(), webhookEventLicenseGenerated, map[string]any{
		"license_key": generated.LicenseKey,
		"slug":        generated.Slug,
		"status":      generated.Status,
		"metadata":    generated.Metadata,
		"expires_at":  generated.ExpiresAt,
		"created_at":  generated.CreatedAt,
	})

	writeRawJSON(w, http.StatusOK, responseBody)
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	var req revokeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	if err := requireField(req.LicenseKey, "license_key"); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.LicenseKey, "license_key", maxLicenseKeyLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	revoked, err := s.service.RevokeLicense(r.Context(), req.LicenseKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "license not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to revoke license", err)
		return
	}

	s.emitWebhookEvent(r.Context(), webhookEventLicenseRevoked, map[string]any{
		"license_key": revoked.LicenseKey,
		"status":      revoked.Status,
		"revoked_at":  revoked.RevokedAt,
	})

	writeJSON(w, http.StatusOK, revokeResponse{
		Valid:      revoked.Valid,
		Status:     revoked.Status,
		LicenseKey: revoked.LicenseKey,
		RevokedAt:  revoked.RevokedAt,
	})
}

func (s *Server) handleActivate(w http.ResponseWriter, r *http.Request) {
	var req activateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.Fingerprint = strings.TrimSpace(req.Fingerprint)
	if err := requireFields(
		requiredField{name: "license_key", value: req.LicenseKey},
		requiredField{name: "fingerprint", value: req.Fingerprint},
	); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.LicenseKey, "license_key", maxLicenseKeyLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.Fingerprint, "fingerprint", maxFingerprintLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if err := validateMetadata(req.Metadata); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.service.ActivateLicense(r.Context(), req.LicenseKey, req.Fingerprint, req.Metadata)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "license not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to activate license", err)
		return
	}

	if result.Valid {
		s.emitWebhookEvent(r.Context(), webhookEventLicenseActivated, map[string]any{
			"license_key": result.LicenseKey,
			"fingerprint": result.Fingerprint,
			"status":      result.Status,
			"expires_at":  result.ExpiresAt,
		})
	}

	writeJSON(w, http.StatusOK, activateResponse{
		Valid:       result.Valid,
		Status:      result.Status,
		LicenseKey:  result.LicenseKey,
		Fingerprint: result.Fingerprint,
		ExpiresAt:   result.ExpiresAt,
		Reason:      result.Reason,
	})
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.Fingerprint = strings.TrimSpace(req.Fingerprint)
	if err := requireFields(
		requiredField{name: "license_key", value: req.LicenseKey},
		requiredField{name: "fingerprint", value: req.Fingerprint},
	); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.LicenseKey, "license_key", maxLicenseKeyLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.Fingerprint, "fingerprint", maxFingerprintLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.service.ValidateLicense(r.Context(), req.LicenseKey, req.Fingerprint)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "license not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to validate license", err)
		return
	}

	eventType := webhookEventLicenseValidationFailed
	if result.Valid {
		eventType = webhookEventLicenseValidated
	}
	s.emitWebhookEvent(r.Context(), eventType, map[string]any{
		"license_key": req.LicenseKey,
		"fingerprint": req.Fingerprint,
		"valid":       result.Valid,
		"status":      result.Status,
		"expires_at":  result.ExpiresAt,
		"reason":      result.Reason,
	})

	writeJSON(w, http.StatusOK, validateResponse{
		Valid:     result.Valid,
		Status:    result.Status,
		ExpiresAt: result.ExpiresAt,
		Reason:    result.Reason,
	})
}

func (s *Server) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	var req deactivateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.Fingerprint = strings.TrimSpace(req.Fingerprint)
	if err := requireFields(
		requiredField{name: "license_key", value: req.LicenseKey},
		requiredField{name: "fingerprint", value: req.Fingerprint},
	); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.LicenseKey, "license_key", maxLicenseKeyLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.Fingerprint, "fingerprint", maxFingerprintLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if err := validateOptionalFieldLength(req.Reason, "reason", maxReasonLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.service.DeactivateLicense(r.Context(), req.LicenseKey, req.Fingerprint, req.Reason)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "license not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to deactivate license", err)
		return
	}

	s.emitWebhookEvent(r.Context(), webhookEventLicenseDeactivated, map[string]any{
		"license_key":     req.LicenseKey,
		"fingerprint":     req.Fingerprint,
		"reason":          req.Reason,
		"released":        result.Released,
		"status":          result.Status,
		"active_seats":    result.ActiveSeats,
		"max_activations": result.MaxActivations,
		"expires_at":      result.ExpiresAt,
	})

	writeJSON(w, http.StatusOK, deactivateResponse{
		Valid:          result.Valid,
		Released:       result.Released,
		Status:         result.Status,
		ActiveSeats:    result.ActiveSeats,
		MaxActivations: result.MaxActivations,
		ExpiresAt:      result.ExpiresAt,
	})
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.service.ListAPIKeys(r.Context())
	if err != nil {
		s.writeUnexpectedError(w, "failed to list api keys", err)
		return
	}

	response := make([]apiKeyResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, mapAPIKeyResponse(key))
	}

	writeJSON(w, http.StatusOK, listAPIKeysResponse{APIKeys: response})
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	if err := validateOptionalFieldLength(req.Name, "name", maxAPIKeyNameLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	created, err := s.service.CreateAPIKey(r.Context(), storage.CreateAPIKeyParams{
		Name:    req.Name,
		KeyType: "server",
	})
	if err != nil {
		s.writeUnexpectedError(w, "failed to create api key", err)
		return
	}

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{
		APIKey:    created.APIKey,
		ID:        created.Record.ID,
		Name:      created.Record.Name,
		CreatedAt: created.Record.CreatedAt,
	})
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	key, err := s.service.RevokeAPIKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "api key not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to revoke api key", err)
		return
	}

	writeJSON(w, http.StatusOK, mapAPIKeyResponse(key))
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	endpoints, err := s.service.ListWebhookEndpoints(r.Context())
	if err != nil {
		s.writeUnexpectedError(w, "failed to list webhook endpoints", err)
		return
	}

	response := make([]webhookEndpointResponse, 0, len(endpoints))
	for _, endpoint := range endpoints {
		response = append(response, mapWebhookEndpointResponse(endpoint))
	}

	writeJSON(w, http.StatusOK, listWebhookEndpointsResponse{Webhooks: response})
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)
	if err := requireFields(
		requiredField{name: "name", value: req.Name},
		requiredField{name: "url", value: req.URL},
	); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.Name, "name", maxWebhookNameLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateFieldLength(req.URL, "url", maxWebhookURLLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := validateWebhookURL(req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	events, err := normalizeWebhookEvents(req.Events)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	created, err := s.service.CreateWebhookEndpoint(r.Context(), storage.CreateWebhookEndpointParams{
		Name:    req.Name,
		URL:     req.URL,
		Events:  events,
		Enabled: enabled,
	})
	if err != nil {
		s.writeUnexpectedError(w, "failed to create webhook endpoint", err)
		return
	}

	writeJSON(w, http.StatusCreated, mapWebhookEndpointResponse(created))
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	var req updateWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if req.Name == nil && req.URL == nil && req.Events == nil && req.Enabled == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "at least one field must be provided"})
		return
	}

	params := storage.UpdateWebhookEndpointParams{Enabled: req.Enabled}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if err := requireField(name, "name"); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		if err := validateFieldLength(name, "name", maxWebhookNameLength); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		params.Name = &name
	}

	if req.URL != nil {
		value := strings.TrimSpace(*req.URL)
		if err := requireField(value, "url"); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		if err := validateFieldLength(value, "url", maxWebhookURLLength); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		if err := validateWebhookURL(value); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		params.URL = &value
	}

	if req.Events != nil {
		events, err := normalizeWebhookEvents(*req.Events)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		params.Events = &events
	}

	updated, err := s.service.UpdateWebhookEndpoint(r.Context(), id, params)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "webhook endpoint not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to update webhook endpoint", err)
		return
	}

	writeJSON(w, http.StatusOK, mapWebhookEndpointResponse(updated))
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	err = s.service.DeleteWebhookEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "webhook endpoint not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to delete webhook endpoint", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) requireProvisioningKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := extractAPIKey(r)
		if apiKey == "" {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing API key"})
			return
		}

		authorized, err := s.service.IsAuthorizedServerAPIKey(r.Context(), apiKey)
		if err != nil {
			s.writeUnexpectedError(w, "failed provisioning api key auth", err)
			return
		}

		if !authorized {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid API key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireManagementKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := extractAPIKey(r)
		if apiKey == "" {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing API key"})
			return
		}

		if !s.isAuthorizedManagementKey(apiKey) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid API key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAuthorizedManagementKey(candidate string) bool {
	if strings.TrimSpace(candidate) == "" {
		return false
	}

	for _, keyHash := range s.managementAPIKeyHashes {
		if err := bcrypt.CompareHashAndPassword(keyHash, []byte(candidate)); err == nil {
			return true
		}
	}

	return false
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		s.logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "panic", rec)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestTimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.rateLimiter.allow(r) {
			w.Header().Set("Retry-After", "1")
			writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "rate limit exceeded"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeUnexpectedError(w http.ResponseWriter, logMsg string, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		s.logger.Warn(logMsg, "error", err, "timeout", true)
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "request timed out"})
	case errors.Is(err, context.Canceled):
		s.logger.Warn(logMsg, "error", err, "canceled", true)
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "request canceled"})
	default:
		s.logger.Error(logMsg, "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
	}
}

func (s *Server) emitWebhookEvent(ctx context.Context, eventType string, payload map[string]any) {
	if err := s.service.EnqueueWebhookEvent(ctx, eventType, payload); err != nil {
		s.logger.Error("failed to enqueue webhook event", "event_type", eventType, "error", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`+"\n", http.StatusInternalServerError)
		return
	}

	writeRawJSON(w, status, body)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	if err := requireJSONContentType(r.Header.Get("Content-Type")); err != nil {
		return err
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}

	if len(body) == 0 {
		return fmt.Errorf("request body is required")
	}

	if len(body) > maxBodyBytes {
		return fmt.Errorf("request body exceeds %d bytes", maxBodyBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("request body is required")
		}
		return fmt.Errorf("invalid request body: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}

func requireJSONContentType(contentType string) error {
	if strings.TrimSpace(contentType) == "" {
		return fmt.Errorf("Content-Type must be application/json")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("invalid Content-Type header")
	}

	if !strings.EqualFold(mediaType, "application/json") {
		return fmt.Errorf("Content-Type must be application/json")
	}

	return nil
}

func hashGenerateRequest(req generateRequest) (string, error) {
	canonical := struct {
		Slug     string         `json:"slug"`
		Metadata map[string]any `json:"metadata"`
	}{
		Slug:     req.Slug,
		Metadata: req.Metadata,
	}

	b, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal canonical request: %w", err)
	}

	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func extractAPIKey(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization != "" {
		const bearer = "bearer "
		if len(authorization) > len(bearer) && strings.EqualFold(authorization[:len(bearer)], bearer) {
			return strings.TrimSpace(authorization[len(bearer):])
		}
	}

	xAPIKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if xAPIKey != "" {
		return xAPIKey
	}

	return ""
}

type requiredField struct {
	name  string
	value string
}

func requireField(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}

	return nil
}

func requireFields(fields ...requiredField) error {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	if len(missing) == 1 {
		return fmt.Errorf("%s is required", missing[0])
	}

	return fmt.Errorf("%s are required", strings.Join(missing, " and "))
}

func validateFieldLength(value, name string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s exceeds max length of %d", name, maxLen)
	}

	return nil
}

func validateOptionalFieldLength(value, name string, maxLen int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return validateFieldLength(trimmed, name, maxLen)
}

func validateSlugName(value string) error {
	_, err := slugdomain.ParseName(value)
	return err
}

func validateMetadata(metadata map[string]any) error {
	if metadata == nil {
		return nil
	}

	if len(metadata) > maxMetadataEntries {
		return fmt.Errorf("metadata exceeds max entries of %d", maxMetadataEntries)
	}

	nodes := 0
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return fmt.Errorf("metadata keys must be non-empty")
		}

		if len(trimmedKey) > maxSlugLength {
			return fmt.Errorf("metadata key %q exceeds max length of %d", trimmedKey, maxSlugLength)
		}

		if err := validateMetadataValue(value, 1, &nodes); err != nil {
			return fmt.Errorf("metadata value for key %q invalid: %w", trimmedKey, err)
		}
	}

	return nil
}

func validateMetadataValue(value any, depth int, nodes *int) error {
	if depth > maxMetadataDepth {
		return fmt.Errorf("max depth %d exceeded", maxMetadataDepth)
	}

	*nodes = *nodes + 1
	if *nodes > maxMetadataNodes {
		return fmt.Errorf("max node count %d exceeded", maxMetadataNodes)
	}

	switch typed := value.(type) {
	case nil, string, float64, bool, int, int32, int64, uint, uint32, uint64, json.Number:
		return nil
	case []any:
		if len(typed) > maxMetadataEntries {
			return fmt.Errorf("array length exceeds %d", maxMetadataEntries)
		}

		for _, item := range typed {
			if err := validateMetadataValue(item, depth+1, nodes); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		if len(typed) > maxMetadataEntries {
			return fmt.Errorf("object field count exceeds %d", maxMetadataEntries)
		}

		for k, v := range typed {
			trimmed := strings.TrimSpace(k)
			if trimmed == "" {
				return fmt.Errorf("object contains empty key")
			}
			if len(trimmed) > maxSlugLength {
				return fmt.Errorf("object key %q exceeds max length %d", trimmed, maxSlugLength)
			}
			if err := validateMetadataValue(v, depth+1, nodes); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported type %T", value)
	}
}

func mapAPIKeyResponse(record storage.APIKeyRecord) apiKeyResponse {
	return apiKeyResponse{
		ID:        record.ID,
		Name:      record.Name,
		Hint:      record.KeyHint,
		CreatedAt: record.CreatedAt,
		RevokedAt: record.RevokedAt,
	}
}

func mapWebhookEndpointResponse(endpoint storage.WebhookEndpoint) webhookEndpointResponse {
	return webhookEndpointResponse{
		ID:        endpoint.ID,
		Name:      endpoint.Name,
		URL:       endpoint.URL,
		Events:    endpoint.Events,
		Enabled:   endpoint.Enabled,
		CreatedAt: endpoint.CreatedAt,
		UpdatedAt: endpoint.UpdatedAt,
	}
}

func parsePathID(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("id is required")
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id must be a positive integer")
	}

	return id, nil
}

func normalizeWebhookEvents(events []string) ([]string, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("events must contain at least one event")
	}

	seen := make(map[string]struct{}, len(events))
	normalized := make([]string, 0, len(events))
	for _, raw := range events {
		event := strings.TrimSpace(raw)
		if event == "" {
			return nil, fmt.Errorf("events must not contain empty values")
		}

		if _, ok := supportedWebhookEventSet[event]; !ok {
			return nil, fmt.Errorf("unsupported webhook event %q", event)
		}

		if _, ok := seen[event]; ok {
			continue
		}
		seen[event] = struct{}{}
		normalized = append(normalized, event)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return supportedWebhookEventOrder[normalized[i]] < supportedWebhookEventOrder[normalized[j]]
	})

	return normalized, nil
}

func validateWebhookURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url is invalid")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("url must use http or https")
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("url must include host")
	}

	return nil
}

func hashManagementAPIKeys(keys map[string]struct{}) [][]byte {
	hashes := make([][]byte, 0, len(keys))
	for key := range keys {
		hashed, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost+2)
		if err != nil {
			panic(fmt.Sprintf("hash management api key: %v", err))
		}
		hashes = append(hashes, hashed)
	}

	return hashes
}
