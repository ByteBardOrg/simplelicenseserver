package config

import (
	"strings"
	"testing"
	"time"
)

func clearOptionalTimeoutEnv(t *testing.T) {
	t.Helper()
	t.Setenv("REQUEST_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")
	t.Setenv("RATE_LIMIT_ENABLED", "")
	t.Setenv("RATE_LIMIT_GLOBAL_RPS", "")
	t.Setenv("RATE_LIMIT_GLOBAL_BURST", "")
	t.Setenv("RATE_LIMIT_PER_IP_RPS", "")
	t.Setenv("RATE_LIMIT_PER_IP_BURST", "")
	t.Setenv("RATE_LIMIT_IP_TTL", "")
	t.Setenv("RATE_LIMIT_MAX_IP_ENTRIES", "")
	t.Setenv("TRUST_PROXY_HEADERS", "")
	t.Setenv("UI_ENABLED", "")
	t.Setenv("OFFLINE_SIGNING_ENCRYPTION_KEY", "")
	t.Setenv("OFFLINE_TOKEN_ISSUER", "")
	t.Setenv("OFFLINE_TOKEN_AUDIENCE", "")
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MANAGEMENT_API_KEYS", "management_key_dev_123456")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error when DATABASE_URL is missing")
	}

	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRequiresServerKeys(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", "")
	t.Setenv("MANAGEMENT_API_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error when no server API keys are configured")
	}

	if !strings.Contains(err.Error(), "MANAGEMENT_API_KEYS or MANAGEMENT_API_KEY is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadParsesServerKeysAndTimeouts(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", " management_key_alpha_123, management_key_beta_456 ,management_key_alpha_123 ")
	t.Setenv("REQUEST_TIMEOUT", "20s")
	t.Setenv("SHUTDOWN_TIMEOUT", "12s")
	t.Setenv("HTTP_READ_TIMEOUT", "13s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "31s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "61s")
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_GLOBAL_RPS", "120")
	t.Setenv("RATE_LIMIT_GLOBAL_BURST", "220")
	t.Setenv("RATE_LIMIT_PER_IP_RPS", "21")
	t.Setenv("RATE_LIMIT_PER_IP_BURST", "41")
	t.Setenv("RATE_LIMIT_IP_TTL", "11m")
	t.Setenv("RATE_LIMIT_MAX_IP_ENTRIES", "15000")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("UI_ENABLED", "true")
	t.Setenv("OFFLINE_SIGNING_ENCRYPTION_KEY", "")
	t.Setenv("OFFLINE_TOKEN_ISSUER", "")
	t.Setenv("OFFLINE_TOKEN_AUDIENCE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if _, ok := cfg.ManagementAPIKeys["management_key_alpha_123"]; !ok {
		t.Fatalf("expected key management_key_alpha_123 in parsed key set")
	}
	if _, ok := cfg.ManagementAPIKeys["management_key_beta_456"]; !ok {
		t.Fatalf("expected key management_key_beta_456 in parsed key set")
	}
	if len(cfg.ManagementAPIKeys) != 2 {
		t.Fatalf("expected deduplicated key set of size 2, got %d", len(cfg.ManagementAPIKeys))
	}

	if cfg.RequestTimeout != 20*time.Second {
		t.Fatalf("unexpected request timeout: %s", cfg.RequestTimeout)
	}
	if cfg.ShutdownTimeout != 12*time.Second {
		t.Fatalf("unexpected shutdown timeout: %s", cfg.ShutdownTimeout)
	}
	if cfg.ReadTimeout != 13*time.Second {
		t.Fatalf("unexpected read timeout: %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 31*time.Second {
		t.Fatalf("unexpected write timeout: %s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 61*time.Second {
		t.Fatalf("unexpected idle timeout: %s", cfg.IdleTimeout)
	}

	if !cfg.RateLimitEnabled {
		t.Fatalf("expected rate limit enabled")
	}
	if cfg.RateLimitGlobalRPS != 120 {
		t.Fatalf("unexpected global rps: %v", cfg.RateLimitGlobalRPS)
	}
	if cfg.RateLimitGlobalBurst != 220 {
		t.Fatalf("unexpected global burst: %d", cfg.RateLimitGlobalBurst)
	}
	if cfg.RateLimitPerIPRPS != 21 {
		t.Fatalf("unexpected per-ip rps: %v", cfg.RateLimitPerIPRPS)
	}
	if cfg.RateLimitPerIPBurst != 41 {
		t.Fatalf("unexpected per-ip burst: %d", cfg.RateLimitPerIPBurst)
	}
	if cfg.RateLimitIPTTL != 11*time.Minute {
		t.Fatalf("unexpected per-ip ttl: %s", cfg.RateLimitIPTTL)
	}
	if cfg.RateLimitMaxIPEntries != 15000 {
		t.Fatalf("unexpected max ip entries: %d", cfg.RateLimitMaxIPEntries)
	}
	if !cfg.TrustProxyHeaders {
		t.Fatalf("expected trust proxy headers enabled")
	}
	if !cfg.UIEnabled {
		t.Fatalf("expected ui enabled")
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", "management_key_dev_123456")
	t.Setenv("REQUEST_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected duration parse error")
	}

	if !strings.Contains(err.Error(), "REQUEST_TIMEOUT") {
		t.Fatalf("expected timeout error to reference REQUEST_TIMEOUT, got: %v", err)
	}
}

func TestLoadRejectsShortAPIKey(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", "short_key")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected API key length validation error")
	}

	if !strings.Contains(err.Error(), "at least 16 characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}
