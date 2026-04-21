package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/storage"
)

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
