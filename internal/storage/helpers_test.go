package storage

import (
	"database/sql"
	"encoding/hex"
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
	if len(parts) != 5 {
		t.Fatalf("expected 5 groups, got %d key=%q", len(parts), key)
	}

	for _, part := range parts {
		if len(part) != 6 {
			t.Fatalf("expected 6 chars per group, got %d for %q", len(part), part)
		}

		if _, err := hex.DecodeString(strings.ToLower(part)); err != nil {
			t.Fatalf("expected hex characters only, got %q", part)
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

func TestGenerateAPIKeyValueFormat(t *testing.T) {
	key, err := generateAPIKeyValue()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}

	if len(key) != 64 {
		t.Fatalf("expected 64-char key, got %d (%q)", len(key), key)
	}

	if _, err := hex.DecodeString(key); err != nil {
		t.Fatalf("expected hex key format: %v", err)
	}

	if len(key) < 20 {
		t.Fatalf("expected generated api key to be long, got %q", key)
	}
}

func TestHashAPIKeyIsDeterministicHex(t *testing.T) {
	h1 := hashAPIKey("server_key_example_123")
	h2 := hashAPIKey("server_key_example_123")

	if h1 != h2 {
		t.Fatalf("expected deterministic hash, got %q and %q", h1, h2)
	}

	if len(h1) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(h1))
	}

	if _, err := hex.DecodeString(h1); err != nil {
		t.Fatalf("expected valid hex hash: %v", err)
	}
}

func TestKeyHint(t *testing.T) {
	if got := keyHint("ab"); got != "ab" {
		t.Fatalf("expected short value unchanged, got %q", got)
	}

	if got := keyHint("short"); got != "shor" {
		t.Fatalf("expected truncated 4-char hint, got %q", got)
	}

	if got := keyHint("0123456789abcdef"); got != "0123" {
		t.Fatalf("expected 4-char hint, got %q", got)
	}
}
