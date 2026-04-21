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
