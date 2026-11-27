package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/storage"
)

type licenseManagementResponse struct {
	ID                        string         `json:"id"`
	LicenseKey                string         `json:"license_key"`
	Slug                      string         `json:"slug"`
	Product                   *string        `json:"product"`
	Status                    string         `json:"status"`
	Metadata                  map[string]any `json:"metadata"`
	Attributes                map[string]any `json:"attributes"`
	MaxActivations            int            `json:"max_activations"`
	ActiveSeats               int            `json:"active_seats"`
	OfflineEnabled            bool           `json:"offline_enabled"`
	OfflineTokenLifetimeHours int            `json:"offline_token_lifetime_hours"`
	ExpiresAt                 *time.Time     `json:"expires_at"`
	CreatedAt                 time.Time      `json:"created_at"`
	ActivatedAt               *time.Time     `json:"activated_at"`
	LastValidatedAt           *time.Time     `json:"last_validated_at"`
	RevokedAt                 *time.Time     `json:"revoked_at"`
}

type licensePaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type licenseCountsResponse struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Revoked  int `json:"revoked"`
	Expired  int `json:"expired"`
}

type listLicensesResponse struct {
	Licenses   []licenseManagementResponse `json:"licenses"`
	Pagination licensePaginationResponse   `json:"pagination"`
	Counts     licenseCountsResponse       `json:"counts"`
}

func (s *Server) handleListLicenses(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := parsePositiveIntQuery(query, "page", 1, 0)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	pageSize, err := parsePositiveIntQuery(query, "page_size", 10, maxListPageSize)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	search := strings.TrimSpace(query.Get("q"))
	if err := validateOptionalFieldLength(search, "search", maxSearchQueryLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	status := strings.TrimSpace(query.Get("status"))
	if err := validateLicenseListStatus(status); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.service.ListLicenses(r.Context(), storage.LicenseListParams{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
		Status:   status,
	})
	if err != nil {
		s.writeUnexpectedError(w, "failed to list licenses", err)
		return
	}

	licenses := make([]licenseManagementResponse, 0, len(result.Licenses))
	for _, license := range result.Licenses {
		licenses = append(licenses, mapLicenseManagementResponse(license))
	}

	writeJSON(w, http.StatusOK, listLicensesResponse{
		Licenses: licenses,
		Pagination: licensePaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      result.Total,
			TotalPages: totalPages(result.Total, pageSize),
		},
		Counts: licenseCountsResponse{
			Total:    result.Counts.Total,
			Active:   result.Counts.Active,
			Inactive: result.Counts.Inactive,
			Revoked:  result.Counts.Revoked,
			Expired:  result.Counts.Expired,
		},
	})
}

func mapLicenseManagementResponse(record storage.LicenseRow) licenseManagementResponse {
	return licenseManagementResponse{
		ID:                        record.ID,
		LicenseKey:                record.Key,
		Slug:                      record.SlugName,
		Product:                   optionalString(record.Product),
		Status:                    displayLicenseStatus(record),
		Metadata:                  record.Metadata,
		Attributes:                record.Attributes,
		MaxActivations:            record.MaxActivations,
		ActiveSeats:               record.ActiveSeats,
		OfflineEnabled:            record.OfflineEnabled,
		OfflineTokenLifetimeHours: offlineTokenLifetimeHoursFromSeconds(record.OfflineTokenLifetimeSeconds),
		ExpiresAt:                 record.ExpiresAt,
		CreatedAt:                 record.CreatedAt,
		ActivatedAt:               record.ActivatedAt,
		LastValidatedAt:           record.LastValidatedAt,
		RevokedAt:                 record.RevokedAt,
	}
}

func displayLicenseStatus(record storage.LicenseRow) string {
	return displayStatus(record.Status, record.ExpiresAt)
}

// displayStatus adds the derived "expired" status; the database only stores inactive, active and revoked.
func displayStatus(status string, expiresAt *time.Time) string {
	if status != "revoked" && expiresAt != nil && !time.Now().UTC().Before(expiresAt.UTC()) {
		return "expired"
	}

	return status
}

func validateLicenseListStatus(status string) error {
	switch status {
	case "", "inactive", "active", "revoked", "expired":
		return nil
	default:
		return fmt.Errorf("status must be one of inactive, active, revoked, or expired")
	}
}

func totalPages(total, pageSize int) int {
	if total == 0 || pageSize <= 0 {
		return 0
	}

	return (total + pageSize - 1) / pageSize
}

type activationManagementResponse struct {
	ID              int64          `json:"id"`
	Fingerprint     string         `json:"fingerprint"`
	Metadata        map[string]any `json:"metadata"`
	ActivatedAt     time.Time      `json:"activated_at"`
	LastValidatedAt *time.Time     `json:"last_validated_at"`
}

type licenseDetailResponse struct {
	licenseManagementResponse
	Activations []activationManagementResponse `json:"activations"`
}

// updateLicenseRequest uses a raw message for expires_at so that `null` (never expires)
// can be told apart from the field being left out (no change).
type updateLicenseRequest struct {
	MaxActivations *int            `json:"max_activations"`
	ExpiresAt      json.RawMessage `json:"expires_at"`
	Revoked        *bool           `json:"revoked"`
}

func (s *Server) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	licenseKey, ok := licenseKeyFromPath(w, r)
	if !ok {
		return
	}

	detail, err := s.service.GetLicense(r.Context(), licenseKey)
	if err != nil {
		s.writeLicenseLookupError(w, err, "failed to get license")
		return
	}

	writeJSON(w, http.StatusOK, mapLicenseDetailResponse(detail))
}

func (s *Server) handleUpdateLicense(w http.ResponseWriter, r *http.Request) {
	licenseKey, ok := licenseKeyFromPath(w, r)
	if !ok {
		return
	}

	var req updateLicenseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	params := storage.UpdateLicenseParams{MaxActivations: req.MaxActivations, Revoked: req.Revoked}
	if len(req.ExpiresAt) > 0 {
		params.SetExpiresAt = true
		if string(req.ExpiresAt) != "null" {
			var raw string
			if err := json.Unmarshal(req.ExpiresAt, &raw); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "expires_at must be an RFC3339 timestamp or null"})
				return
			}
			parsed, err := parseOptionalRFC3339(raw)
			if err != nil || parsed == nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "expires_at must be an RFC3339 timestamp or null"})
				return
			}
			params.ExpiresAt = parsed
		}
	}

	if params.MaxActivations == nil && !params.SetExpiresAt && params.Revoked == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "at least one of max_activations, expires_at or revoked is required"})
		return
	}
	if params.MaxActivations != nil && *params.MaxActivations <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "max_activations must be greater than 0"})
		return
	}

	updated, err := s.service.UpdateLicense(r.Context(), licenseKey, params)
	if err != nil {
		s.writeLicenseLookupError(w, err, "failed to update license")
		return
	}

	license := updated.Detail.License
	payload := map[string]any{
		"license_key":     license.Key,
		"status":          displayLicenseStatus(license),
		"max_activations": license.MaxActivations,
		"expires_at":      license.ExpiresAt,
		"revoked_at":      license.RevokedAt,
	}
	if updated.NewlyRevoked {
		s.emitWebhookEvent(r.Context(), webhookEventLicenseRevoked, payload)
	} else {
		s.emitWebhookEvent(r.Context(), webhookEventLicenseUpdated, payload)
	}

	writeJSON(w, http.StatusOK, mapLicenseDetailResponse(updated.Detail))
}

func (s *Server) handleRemoveActivation(w http.ResponseWriter, r *http.Request) {
	licenseKey, ok := licenseKeyFromPath(w, r)
	if !ok {
		return
	}

	activationID, err := parsePathID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	removed, err := s.service.RemoveActivation(r.Context(), licenseKey, activationID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "license or activation not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to remove activation", err)
		return
	}

	license := removed.Detail.License
	s.emitWebhookEvent(r.Context(), webhookEventLicenseDeactivated, map[string]any{
		"license_key":     license.Key,
		"fingerprint":     removed.Fingerprint,
		"reason":          storage.AdminRemovedReason,
		"released":        true,
		"status":          displayLicenseStatus(license),
		"active_seats":    license.ActiveSeats,
		"max_activations": license.MaxActivations,
		"expires_at":      license.ExpiresAt,
	})

	writeJSON(w, http.StatusOK, mapLicenseDetailResponse(removed.Detail))
}

func licenseKeyFromPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	licenseKey := strings.TrimSpace(r.PathValue("key"))
	if err := requireField(licenseKey, "license key"); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return "", false
	}
	if err := validateFieldLength(licenseKey, "license key", maxLicenseKeyLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return "", false
	}
	return licenseKey, true
}

func (s *Server) writeLicenseLookupError(w http.ResponseWriter, err error, message string) {
	if errors.Is(err, storage.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "license not found"})
		return
	}
	s.writeUnexpectedError(w, message, err)
}

func mapLicenseDetailResponse(detail storage.LicenseDetail) licenseDetailResponse {
	activations := make([]activationManagementResponse, 0, len(detail.Activations))
	for _, activation := range detail.Activations {
		activations = append(activations, activationManagementResponse{
			ID:              activation.ID,
			Fingerprint:     activation.Fingerprint,
			Metadata:        activation.Metadata,
			ActivatedAt:     activation.CreatedAt,
			LastValidatedAt: activation.LastValidatedAt,
		})
	}

	return licenseDetailResponse{
		licenseManagementResponse: mapLicenseManagementResponse(detail.License),
		Activations:               activations,
	}
}
