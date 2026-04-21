package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/storage"
)

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
