package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/storage"
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
	generateEndpointKey     = "/generate"
)

type Server struct {
	service            licenseService
	logger             *slog.Logger
	serverAPIKeyHashes [][32]byte
	requestTimeout     time.Duration
	rateLimiter        *ipRateLimiter
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
	GenerateLicenseIdempotent(ctx context.Context, endpoint, idemKey, requestHash, slugName string, metadata map[string]any) (storage.GeneratedLicense, json.RawMessage, bool, error)
	GenerateLicense(ctx context.Context, slugName string, metadata map[string]any) (storage.GeneratedLicense, error)
	RevokeLicense(ctx context.Context, licenseKey string) (storage.RevokeResult, error)
	ActivateLicense(ctx context.Context, licenseKey, fingerprint string, metadata map[string]any) (storage.ActivationResult, error)
	ValidateLicense(ctx context.Context, licenseKey, fingerprint string) (storage.ValidationResult, error)
	DeactivateLicense(ctx context.Context, licenseKey, fingerprint, reason string) (storage.DeactivationResult, error)
}

func NewServer(service licenseService, logger *slog.Logger, serverAPIKeys map[string]struct{}, requestTimeout time.Duration) *Server {
	opts := DefaultOptions()
	opts.RequestTimeout = requestTimeout
	return NewServerWithOptions(service, logger, serverAPIKeys, opts)
}

func NewServerWithOptions(service licenseService, logger *slog.Logger, serverAPIKeys map[string]struct{}, opts Options) *Server {
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
		service:            service,
		logger:             logger,
		serverAPIKeyHashes: hashAPIKeys(serverAPIKeys),
		requestTimeout:     opts.RequestTimeout,
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
	mux.Handle("POST /generate", s.requireServerKey(http.HandlerFunc(s.handleGenerate)))
	mux.Handle("POST /revoke", s.requireServerKey(http.HandlerFunc(s.handleRevoke)))
	mux.HandleFunc("POST /activate", s.handleActivate)
	mux.HandleFunc("POST /validate", s.handleValidate)
	mux.HandleFunc("POST /deactivate", s.handleDeactivate)

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

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := s.service.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "database unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
		_, responseBody, _, err := s.service.GenerateLicenseIdempotent(
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

	writeJSON(w, http.StatusOK, deactivateResponse{
		Valid:          result.Valid,
		Released:       result.Released,
		Status:         result.Status,
		ActiveSeats:    result.ActiveSeats,
		MaxActivations: result.MaxActivations,
		ExpiresAt:      result.ExpiresAt,
	})
}

func (s *Server) requireServerKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := extractAPIKey(r)
		if apiKey == "" {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing API key"})
			return
		}

		if !s.isAuthorizedServerKey(apiKey) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid API key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAuthorizedServerKey(candidate string) bool {
	if strings.TrimSpace(candidate) == "" {
		return false
	}

	candidateHash := sha256.Sum256([]byte(candidate))
	matched := 0
	for _, keyHash := range s.serverAPIKeyHashes {
		matched |= subtle.ConstantTimeCompare(candidateHash[:], keyHash[:])
	}

	return matched == 1
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

func hashAPIKeys(keys map[string]struct{}) [][32]byte {
	hashes := make([][32]byte, 0, len(keys))
	for key := range keys {
		hashes = append(hashes, sha256.Sum256([]byte(key)))
	}

	return hashes
}
