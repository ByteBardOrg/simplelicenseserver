package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simple-license-server/internal/offlinejwt"
	"simple-license-server/internal/storage"
)

type signingKeyResponse struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Kid          string     `json:"kid"`
	Algorithm    string     `json:"algorithm"`
	Status       string     `json:"status"`
	PublicKeyPEM string     `json:"public_key_pem,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ActivatedAt  *time.Time `json:"activated_at"`
	RetiredAt    *time.Time `json:"retired_at"`
}

type listSigningKeysResponse struct {
	SigningKeys []signingKeyResponse `json:"signing_keys"`
}

type listPublicSigningKeysResponse struct {
	SigningKeys []signingKeyResponse `json:"signing_keys"`
}

type createSigningKeyRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleListSigningKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.service.ListSigningKeys(r.Context())
	if err != nil {
		s.writeUnexpectedError(w, "failed to list signing keys", err)
		return
	}

	response := make([]signingKeyResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, mapSigningKeyResponse(key, false))
	}

	writeJSON(w, http.StatusOK, listSigningKeysResponse{SigningKeys: response})
}

func (s *Server) handleListPublicSigningKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.service.ListPublicSigningKeys(r.Context())
	if err != nil {
		s.writeUnexpectedError(w, "failed to list public signing keys", err)
		return
	}

	response := make([]signingKeyResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, mapSigningKeyResponse(key, true))
	}

	writeJSON(w, http.StatusOK, listPublicSigningKeysResponse{SigningKeys: response})
}

func (s *Server) handleCreateSigningKey(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(s.offlineSigningKey) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "OFFLINE_SIGNING_ENCRYPTION_KEY is not configured"})
		return
	}

	var req createSigningKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if err := validateOptionalFieldLength(name, "name", maxSigningKeyNameLength); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	keyPair, err := offlinejwt.GenerateEd25519KeyPair()
	if err != nil {
		s.writeUnexpectedError(w, "failed to generate signing key", err)
		return
	}

	encryptedPrivateKey, err := offlinejwt.EncryptPrivateKey(keyPair.PrivatePEM, s.offlineSigningKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	created, err := s.service.CreateSigningKey(r.Context(), storage.CreateSigningKeyParams{
		Name:                name,
		Kid:                 keyPair.Kid,
		Algorithm:           keyPair.Algorithm,
		PrivateKeyEncrypted: encryptedPrivateKey,
		PublicKeyPEM:        keyPair.PublicPEM,
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "signing key already exists"})
			return
		}
		s.writeUnexpectedError(w, "failed to create signing key", err)
		return
	}

	writeJSON(w, http.StatusCreated, mapSigningKeyResponse(created, true))
}

func (s *Server) handleActivateSigningKey(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	key, err := s.service.ActivateSigningKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "signing key not found"})
			return
		}
		if errors.Is(err, storage.ErrConflict) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "retired signing key cannot be activated"})
			return
		}
		s.writeUnexpectedError(w, "failed to activate signing key", err)
		return
	}

	writeJSON(w, http.StatusOK, mapSigningKeyResponse(key, true))
}

func (s *Server) handleRetireSigningKey(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	key, err := s.service.RetireSigningKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "signing key not found"})
			return
		}
		s.writeUnexpectedError(w, "failed to retire signing key", err)
		return
	}

	writeJSON(w, http.StatusOK, mapSigningKeyResponse(key, true))
}

func mapSigningKeyResponse(record storage.SigningKeyRecord, includePublicKey bool) signingKeyResponse {
	resp := signingKeyResponse{
		ID:          record.ID,
		Name:        record.Name,
		Kid:         record.Kid,
		Algorithm:   record.Algorithm,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		ActivatedAt: record.ActivatedAt,
		RetiredAt:   record.RetiredAt,
	}
	if includePublicKey {
		resp.PublicKeyPEM = record.PublicKeyPEM
	}

	return resp
}
