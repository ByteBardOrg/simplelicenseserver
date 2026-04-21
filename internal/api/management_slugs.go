package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	slugdomain "simple-license-server/internal/domain/slug"
	"simple-license-server/internal/storage"
)

type slugResponse struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	MaxActivations int        `json:"max_activations"`
	ExpirationType string     `json:"expiration_type"`
	ExpirationDays *int       `json:"expiration_days"`
	FixedExpiresAt *time.Time `json:"fixed_expires_at"`
	IsDefault      bool       `json:"is_default"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type listSlugsResponse struct {
	Slugs []slugResponse `json:"slugs"`
}

type createSlugRequest struct {
	Name           string  `json:"name"`
	MaxActivations int     `json:"max_activations"`
	ExpirationType string  `json:"expiration_type"`
	ExpirationDays *int    `json:"expiration_days"`
	FixedExpiresAt *string `json:"fixed_expires_at"`
}

type updateSlugRequest struct {
	Name           *string `json:"name"`
	MaxActivations *int    `json:"max_activations"`
	ExpirationType *string `json:"expiration_type"`
	ExpirationDays *int    `json:"expiration_days"`
	FixedExpiresAt *string `json:"fixed_expires_at"`
}

func (s *Server) handleListSlugs(w http.ResponseWriter, r *http.Request) {
	items, err := s.service.ListSlugs(r.Context())
	if err != nil {
		s.writeUnexpectedError(w, "failed to list slugs", err)
		return
	}

	response := make([]slugResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapSlugResponse(item))
	}

	writeJSON(w, http.StatusOK, listSlugsResponse{Slugs: response})
}

func (s *Server) handleGetSlug(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if err := validateSlugPathName(name); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	record, err := s.service.GetSlugByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "slug not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to get slug", err)
		return
	}

	writeJSON(w, http.StatusOK, mapSlugResponse(record))
}

func (s *Server) handleCreateSlug(w http.ResponseWriter, r *http.Request) {
	var req createSlugRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.ExpirationType = strings.TrimSpace(req.ExpirationType)

	if err := validateSlugPathName(req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	policy, err := resolveSlugPolicy(req.MaxActivations, req.ExpirationType, req.ExpirationDays, req.FixedExpiresAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	record, err := s.service.CreateSlug(r.Context(), storage.CreateSlugParams{
		Name:           req.Name,
		MaxActivations: policy.MaxActivations(),
		ExpirationType: policy.ExpirationType(),
		ExpirationDays: policy.ExpirationDays(),
		FixedExpiresAt: policy.FixedExpiresAt(),
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "slug already exists"})
			return
		}
		s.writeUnexpectedError(w, "failed to create slug", err)
		return
	}

	writeJSON(w, http.StatusCreated, mapSlugResponse(record))
}

func (s *Server) handleUpdateSlug(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if err := validateSlugPathName(name); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	current, err := s.service.GetSlugByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "slug not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to load slug", err)
		return
	}

	var req updateSlugRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if req.Name == nil && req.MaxActivations == nil && req.ExpirationType == nil && req.ExpirationDays == nil && req.FixedExpiresAt == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "at least one field must be provided"})
		return
	}

	resolvedName := current.Name
	if req.Name != nil {
		resolvedName = strings.TrimSpace(*req.Name)
		if err := validateSlugPathName(resolvedName); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
	}

	resolvedMaxActivations := current.MaxActivations
	if req.MaxActivations != nil {
		resolvedMaxActivations = *req.MaxActivations
	}

	resolvedExpirationType := current.ExpirationType
	if req.ExpirationType != nil {
		resolvedExpirationType = strings.TrimSpace(*req.ExpirationType)
	}

	resolvedExpirationDays := current.ExpirationDays
	if req.ExpirationDays != nil {
		v := *req.ExpirationDays
		resolvedExpirationDays = &v
	}

	resolvedFixedExpiresAt := current.FixedExpiresAt
	if req.FixedExpiresAt != nil {
		parsed, err := parseOptionalRFC3339(*req.FixedExpiresAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		resolvedFixedExpiresAt = parsed
	}

	policy, err := resolveSlugPolicy(
		resolvedMaxActivations,
		resolvedExpirationType,
		resolvedExpirationDays,
		rfc3339Ptr(resolvedFixedExpiresAt),
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	nameParam := resolvedName
	maxParam := policy.MaxActivations()
	expTypeParam := policy.ExpirationType()
	expDaysParam := policy.ExpirationDays()
	fixedParam := policy.FixedExpiresAt()

	record, err := s.service.UpdateSlugByName(r.Context(), name, storage.UpdateSlugParams{
		Name:           &nameParam,
		MaxActivations: &maxParam,
		ExpirationType: &expTypeParam,
		ExpirationDays: expDaysParam,
		FixedExpiresAt: &fixedParam,
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "slug not found"})
			return
		}
		if errors.Is(err, storage.ErrConflict) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "slug already exists"})
			return
		}
		s.writeUnexpectedError(w, "failed to update slug", err)
		return
	}

	writeJSON(w, http.StatusOK, mapSlugResponse(record))
}

func (s *Server) handleDeleteSlug(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if err := validateSlugPathName(name); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	err := s.service.DeleteSlugByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "slug not found"})
			return
		}
		if errors.Is(err, storage.ErrConflict) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "slug cannot be deleted"})
			return
		}
		s.writeUnexpectedError(w, "failed to delete slug", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func mapSlugResponse(record storage.SlugRecord) slugResponse {
	return slugResponse{
		ID:             record.ID,
		Name:           record.Name,
		MaxActivations: record.MaxActivations,
		ExpirationType: record.ExpirationType,
		ExpirationDays: record.ExpirationDays,
		FixedExpiresAt: record.FixedExpiresAt,
		IsDefault:      record.IsDefault,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func validateSlugPathName(name string) error {
	if err := requireField(name, "slug name"); err != nil {
		return err
	}
	if err := validateFieldLength(name, "slug name", maxSlugLength); err != nil {
		return err
	}
	if err := validateSlugName(name); err != nil {
		return err
	}

	return nil
}

func resolveSlugPolicy(maxActivations int, expirationType string, expirationDays *int, fixedExpiresAtRaw *string) (slugdomain.Policy, error) {
	var fixedExpiresAt *time.Time
	if fixedExpiresAtRaw != nil {
		parsed, err := parseOptionalRFC3339(*fixedExpiresAtRaw)
		if err != nil {
			return slugdomain.Policy{}, err
		}
		fixedExpiresAt = parsed
	}

	return slugdomain.NewPolicy(maxActivations, expirationType, expirationDays, fixedExpiresAt)
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, fmt.Errorf("timestamp must be RFC3339")
	}

	v := parsed.UTC()
	return &v, nil
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.UTC().Format(time.RFC3339)
	return &v
}
