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
	t.Setenv("CLIENT_IP_HEADER", "")
	t.Setenv("ALLOW_EXAMPLE_SECRETS", "true")
	t.Setenv("AUTH_FAILURE_MAX_ATTEMPTS", "")
	t.Setenv("AUTH_FAILURE_WINDOW", "")
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
	t.Setenv("CLIENT_IP_HEADER", "CF-Connecting-IP")
	t.Setenv("AUTH_FAILURE_MAX_ATTEMPTS", "5")
	t.Setenv("AUTH_FAILURE_WINDOW", "30m")
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
	if cfg.ClientIPHeader != "CF-Connecting-IP" {
		t.Fatalf("unexpected client ip header: %q", cfg.ClientIPHeader)
	}
	if cfg.AuthFailureMaxAttempts != 5 || cfg.AuthFailureWindow != 30*time.Minute {
		t.Fatalf("unexpected auth failure settings: %d %s", cfg.AuthFailureMaxAttempts, cfg.AuthFailureWindow)
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

func TestLoadDefaultsForProxyAndAbuseProtection(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", "management_key_dev_123456")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ClientIPHeader != "" {
		t.Fatalf("expected the TCP peer address by default, got header %q", cfg.ClientIPHeader)
	}
	if cfg.RateLimitGlobalRPS != 0 {
		t.Fatalf("expected the global limit to be off by default, got %v", cfg.RateLimitGlobalRPS)
	}
	if cfg.AuthFailureMaxAttempts != 10 || cfg.AuthFailureWindow != 15*time.Minute {
		t.Fatalf("unexpected auth failure defaults: %d %s", cfg.AuthFailureMaxAttempts, cfg.AuthFailureWindow)
	}
}

func TestLoadRejectsLegacyTrustProxyHeaders(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", "management_key_dev_123456")
	t.Setenv("TRUST_PROXY_HEADERS", "true")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CLIENT_IP_HEADER") {
		t.Fatalf("expected a pointer to CLIENT_IP_HEADER, got %v", err)
	}

	t.Setenv("TRUST_PROXY_HEADERS", "false")
	if _, err := Load(); err != nil {
		t.Fatalf("TRUST_PROXY_HEADERS=false should be ignored, got %v", err)
	}
}

func TestLoadRejectsInvalidClientIPHeader(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("MANAGEMENT_API_KEYS", "management_key_dev_123456")
	t.Setenv("CLIENT_IP_HEADER", "X-Forwarded-For: 1.2.3.4")

	if _, err := Load(); err == nil {
		t.Fatalf("expected an error for an invalid header name")
	}
}

func TestLoadRejectsPublishedSecrets(t *testing.T) {
	clearOptionalTimeoutEnv(t)
	t.Setenv("ALLOW_EXAMPLE_SECRETS", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	t.Setenv("MANAGEMENT_API_KEYS", "my_own_random_key_0123456789,management_key_dev_123456")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "MANAGEMENT_API_KEYS") {
		t.Fatalf("expected a published management key to be refused, got %v", err)
	}

	t.Setenv("MANAGEMENT_API_KEYS", "my_own_random_key_0123456789")
	t.Setenv("OFFLINE_SIGNING_ENCRYPTION_KEY", "change_me_to_at_least_32_chars_long")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "OFFLINE_SIGNING_ENCRYPTION_KEY") {
		t.Fatalf("expected a published signing secret to be refused, got %v", err)
	}

	t.Setenv("ALLOW_EXAMPLE_SECRETS", "true")
	if _, err := Load(); err != nil {
		t.Fatalf("ALLOW_EXAMPLE_SECRETS=true should permit example values, got %v", err)
	}

	t.Setenv("ALLOW_EXAMPLE_SECRETS", "")
	t.Setenv("OFFLINE_SIGNING_ENCRYPTION_KEY", "a_real_secret_that_nobody_published_0123")
	if _, err := Load(); err != nil {
		t.Fatalf("own secrets should load, got %v", err)
	}
}
