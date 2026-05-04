package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"simple-license-server/internal/offlinejwt"
	"simple-license-server/internal/storage"
)

func (s *Server) issueOfflineToken(ctx context.Context, licenseID, slugName, fingerprint string, licenseExpiresAt *time.Time, lifetimeSeconds int) (string, error) {
	licenseID = strings.TrimSpace(licenseID)
	slugName = strings.TrimSpace(slugName)
	fingerprint = strings.TrimSpace(fingerprint)
	if licenseID == "" || slugName == "" || fingerprint == "" {
		return "", nil
	}
	if lifetimeSeconds <= 0 {
		return "", fmt.Errorf("offline token lifetime must be greater than 0")
	}

	activeKey, err := s.service.GetActiveSigningKey(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	if activeKey.Algorithm != offlinejwt.AlgorithmEd25519 {
		return "", fmt.Errorf("unsupported active signing key algorithm %q", activeKey.Algorithm)
	}
	if strings.TrimSpace(s.offlineSigningKey) == "" {
		return "", fmt.Errorf("OFFLINE_SIGNING_ENCRYPTION_KEY is not configured")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(lifetimeSeconds) * time.Second)
	if licenseExpiresAt != nil && licenseExpiresAt.UTC().Before(expiresAt) {
		expiresAt = licenseExpiresAt.UTC()
	}
	if !expiresAt.After(now) {
		return "", nil
	}

	return offlinejwt.SignEd25519JWT(activeKey.PrivateKeyEncrypted, s.offlineSigningKey, activeKey.Kid, offlinejwt.TokenParams{
		Issuer:      s.offlineTokenIssuer,
		Audience:    s.offlineTokenAudience,
		Subject:     licenseID,
		Slug:        slugName,
		Fingerprint: fingerprint,
		ExpiresAt:   expiresAt,
		IssuedAt:    now,
	})
}
