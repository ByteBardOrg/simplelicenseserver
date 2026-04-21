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

type errorResponse struct {
	Error string `json:"error"`
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
