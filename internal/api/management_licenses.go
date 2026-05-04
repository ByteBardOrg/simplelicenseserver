package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/storage"
)

type licenseManagementResponse struct {
	ID                          string         `json:"id"`
	LicenseKey                  string         `json:"license_key"`
	Slug                        string         `json:"slug"`
	Status                      string         `json:"status"`
	Metadata                    map[string]any `json:"metadata"`
	MaxActivations              int            `json:"max_activations"`
	ActiveSeats                 int            `json:"active_seats"`
	OfflineEnabled              bool           `json:"offline_enabled"`
	OfflineTokenLifetimeHours   int            `json:"offline_token_lifetime_hours"`
	ExpiresAt                   *time.Time     `json:"expires_at"`
	CreatedAt                   time.Time      `json:"created_at"`
	ActivatedAt                 *time.Time     `json:"activated_at"`
	LastValidatedAt             *time.Time     `json:"last_validated_at"`
	RevokedAt                   *time.Time     `json:"revoked_at"`
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
		ID:                          record.ID,
		LicenseKey:                  record.Key,
		Slug:                        record.SlugName,
		Status:                      displayLicenseStatus(record),
		Metadata:                    record.Metadata,
		MaxActivations:              record.MaxActivations,
		ActiveSeats:                 record.ActiveSeats,
		OfflineEnabled:              record.OfflineEnabled,
		OfflineTokenLifetimeHours:   offlineTokenLifetimeHoursFromSeconds(record.OfflineTokenLifetimeSeconds),
		ExpiresAt:                   record.ExpiresAt,
		CreatedAt:                   record.CreatedAt,
		ActivatedAt:                 record.ActivatedAt,
		LastValidatedAt:             record.LastValidatedAt,
		RevokedAt:                   record.RevokedAt,
	}
}

func displayLicenseStatus(record storage.LicenseRow) string {
	if record.Status != "revoked" && record.ExpiresAt != nil && !time.Now().UTC().Before(record.ExpiresAt.UTC()) {
		return "expired"
	}

	return record.Status
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
