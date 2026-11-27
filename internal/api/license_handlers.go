package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/storage"
	"simple-license-server/internal/version"
)

type generateRequest struct {
	Slug     string         `json:"slug"`
	Metadata map[string]any `json:"metadata"`
}

type generateResponse struct {
	LicenseKey string         `json:"license_key"`
	Slug       string         `json:"slug"`
	Status     string         `json:"status"`
	Metadata   map[string]any `json:"metadata"`
	Attributes map[string]any `json:"attributes"`
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

type validateRequest struct {
	LicenseKey  string `json:"license_key"`
	Fingerprint string `json:"fingerprint"`
}

type deactivateRequest struct {
	LicenseKey  string `json:"license_key"`
	Fingerprint string `json:"fingerprint"`
	Reason      string `json:"reason"`
}

// publicLicenseResponse is the single response shape of /api/v1/activate, /validate and /deactivate.
// Anyone holding a license key can call these endpoints, so it only carries what an app needs to decide
// whether to unlock itself: never license metadata, slug names, internal IDs, or other devices.
type publicLicenseResponse struct {
	Valid   bool           `json:"valid"`
	Reason  *string        `json:"reason"`
	License *publicLicense `json:"license"`
	Token   string         `json:"token,omitempty"`
}

type publicLicense struct {
	// Product lets an app confirm the key is for it (compare with a value hard-coded in the app).
	Product    *string        `json:"product"`
	Status     string         `json:"status"`
	ExpiresAt  *time.Time     `json:"expires_at"`
	Seats      publicSeats    `json:"seats"`
	Attributes map[string]any `json:"attributes"`
}

type publicSeats struct {
	Used int `json:"used"`
	Max  int `json:"max"`
}

const reasonNotFound = "not_found"

func newPublicLicenseResponse(valid bool, reason, product, status string, expiresAt *time.Time, usedSeats, maxSeats int, attributes map[string]any) publicLicenseResponse {
	if attributes == nil {
		attributes = map[string]any{}
	}
	resp := publicLicenseResponse{
		Valid: valid,
		License: &publicLicense{
			Product:    optionalString(product),
			Status:     displayStatus(status, expiresAt),
			ExpiresAt:  expiresAt,
			Seats:      publicSeats{Used: usedSeats, Max: maxSeats},
			Attributes: attributes,
		},
	}
	if !valid && reason != "" {
		resp.Reason = &reason
	}
	return resp
}

// optionalString maps the empty string the database stores for "not set" to JSON null.
func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func notFoundLicenseResponse() publicLicenseResponse {
	reason := reasonNotFound
	return publicLicenseResponse{Valid: false, Reason: &reason}
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
				"attributes":  generated.Attributes,
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
		Attributes: generated.Attributes,
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
		"attributes":  generated.Attributes,
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
			writeJSON(w, http.StatusOK, notFoundLicenseResponse())
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

	token := ""
	if result.Valid && result.OfflineEnabled {
		var err error
		token, err = s.issueOfflineToken(r.Context(), result.LicenseID, result.Slug, result.Product, result.Fingerprint, result.Attributes, result.ExpiresAt, result.OfflineTokenLifetimeSeconds)
		if err != nil {
			s.writeUnexpectedError(w, "failed to issue offline token", err)
			return
		}
	}

	resp := newPublicLicenseResponse(result.Valid, result.Reason, result.Product, result.Status, result.ExpiresAt, result.ActiveSeats, result.MaxActivations, result.Attributes)
	resp.Token = token
	writeJSON(w, http.StatusOK, resp)
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
			writeJSON(w, http.StatusOK, notFoundLicenseResponse())
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

	token := ""
	if result.Valid && result.OfflineEnabled {
		var err error
		token, err = s.issueOfflineToken(r.Context(), result.LicenseID, result.Slug, result.Product, req.Fingerprint, result.Attributes, result.ExpiresAt, result.OfflineTokenLifetimeSeconds)
		if err != nil {
			s.writeUnexpectedError(w, "failed to refresh offline token", err)
			return
		}
	}

	resp := newPublicLicenseResponse(result.Valid, result.Reason, result.Product, result.Status, result.ExpiresAt, result.ActiveSeats, result.MaxActivations, result.Attributes)
	resp.Token = token
	writeJSON(w, http.StatusOK, resp)
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
			writeJSON(w, http.StatusOK, notFoundLicenseResponse())
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

	writeJSON(w, http.StatusOK, newPublicLicenseResponse(result.Valid, result.Reason, result.Product, result.Status, result.ExpiresAt, result.ActiveSeats, result.MaxActivations, result.Attributes))
}
