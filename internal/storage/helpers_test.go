package storage

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestGenerateLicenseKeyFormat(t *testing.T) {
	key, err := generateLicenseKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	parts := strings.Split(key, "-")
	if len(parts) != 4 {
		t.Fatalf("expected 4 groups, got %d key=%q", len(parts), key)
	}

	for _, part := range parts {
		if len(part) != 4 {
			t.Fatalf("expected 4 chars per group, got %d for %q", len(part), part)
		}
	}
}

func TestResolveExpirationDuration(t *testing.T) {
	expiresAt, err := resolveExpiration("duration", sql.NullInt32{Int32: 30, Valid: true}, sql.NullTime{})
	if err != nil {
		t.Fatalf("resolve expiration: %v", err)
	}

	if expiresAt == nil {
		t.Fatalf("expected non-nil expiration for duration policy")
	}

	if time.Until(*expiresAt) < (29 * 24 * time.Hour) {
		t.Fatalf("expected expiration roughly 30 days in future, got %s", expiresAt)
	}
}

func TestResolveExpirationFixedDate(t *testing.T) {
	fixed := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	expiresAt, err := resolveExpiration("fixed_date", sql.NullInt32{}, sql.NullTime{Time: fixed, Valid: true})
	if err != nil {
		t.Fatalf("resolve fixed date expiration: %v", err)
	}

	if expiresAt == nil || !expiresAt.Equal(fixed) {
		t.Fatalf("expected fixed expiration %s, got %+v", fixed, expiresAt)
	}
}
