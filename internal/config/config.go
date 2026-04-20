package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	ServerAPIKeys         map[string]struct{}
	RequestTimeout        time.Duration
	ShutdownTimeout       time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	RateLimitEnabled      bool
	RateLimitGlobalRPS    float64
	RateLimitGlobalBurst  int
	RateLimitPerIPRPS     float64
	RateLimitPerIPBurst   int
	RateLimitIPTTL        time.Duration
	RateLimitMaxIPEntries int
	TrustProxyHeaders     bool
}

func Load() (Config, error) {
	cfg := Config{
		Port:                  strings.TrimSpace(getEnv("PORT", "8080")),
		DatabaseURL:           strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RequestTimeout:        15 * time.Second,
		ShutdownTimeout:       10 * time.Second,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
		RateLimitEnabled:      true,
		RateLimitGlobalRPS:    100,
		RateLimitGlobalBurst:  200,
		RateLimitPerIPRPS:     20,
		RateLimitPerIPBurst:   40,
		RateLimitIPTTL:        10 * time.Minute,
		RateLimitMaxIPEntries: 10000,
		TrustProxyHeaders:     false,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	keys := strings.TrimSpace(os.Getenv("SERVER_API_KEYS"))
	if keys == "" {
		if single := strings.TrimSpace(os.Getenv("SERVER_API_KEY")); single != "" {
			keys = single
		}
	}

	if keys == "" {
		return Config{}, fmt.Errorf("SERVER_API_KEYS or SERVER_API_KEY is required")
	}

	cfg.ServerAPIKeys = make(map[string]struct{})
	for _, raw := range strings.Split(keys, ",") {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}

		if len(key) < 16 {
			return Config{}, fmt.Errorf("server API keys must be at least 16 characters")
		}

		cfg.ServerAPIKeys[key] = struct{}{}
	}

	if len(cfg.ServerAPIKeys) == 0 {
		return Config{}, fmt.Errorf("no valid API keys found in SERVER_API_KEYS")
	}

	var err error
	cfg.RequestTimeout, err = parseDurationEnv("REQUEST_TIMEOUT", cfg.RequestTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = parseDurationEnv("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.ReadTimeout, err = parseDurationEnv("HTTP_READ_TIMEOUT", cfg.ReadTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.WriteTimeout, err = parseDurationEnv("HTTP_WRITE_TIMEOUT", cfg.WriteTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.IdleTimeout, err = parseDurationEnv("HTTP_IDLE_TIMEOUT", cfg.IdleTimeout)
	if err != nil {
		return Config{}, err
	}

	if cfg.RequestTimeout <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT must be greater than 0")
	}

	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be greater than 0")
	}

	if cfg.ReadTimeout <= 0 || cfg.WriteTimeout <= 0 || cfg.IdleTimeout <= 0 {
		return Config{}, fmt.Errorf("HTTP_*_TIMEOUT values must be greater than 0")
	}

	cfg.RateLimitEnabled, err = parseBoolEnv("RATE_LIMIT_ENABLED", cfg.RateLimitEnabled)
	if err != nil {
		return Config{}, err
	}

	cfg.RateLimitGlobalRPS, err = parseFloat64Env("RATE_LIMIT_GLOBAL_RPS", cfg.RateLimitGlobalRPS)
	if err != nil {
		return Config{}, err
	}

	cfg.RateLimitGlobalBurst, err = parseIntEnv("RATE_LIMIT_GLOBAL_BURST", cfg.RateLimitGlobalBurst)
	if err != nil {
		return Config{}, err
	}

	cfg.RateLimitPerIPRPS, err = parseFloat64Env("RATE_LIMIT_PER_IP_RPS", cfg.RateLimitPerIPRPS)
	if err != nil {
		return Config{}, err
	}

	cfg.RateLimitPerIPBurst, err = parseIntEnv("RATE_LIMIT_PER_IP_BURST", cfg.RateLimitPerIPBurst)
	if err != nil {
		return Config{}, err
	}

	cfg.RateLimitIPTTL, err = parseDurationEnv("RATE_LIMIT_IP_TTL", cfg.RateLimitIPTTL)
	if err != nil {
		return Config{}, err
	}

	cfg.RateLimitMaxIPEntries, err = parseIntEnv("RATE_LIMIT_MAX_IP_ENTRIES", cfg.RateLimitMaxIPEntries)
	if err != nil {
		return Config{}, err
	}

	cfg.TrustProxyHeaders, err = parseBoolEnv("TRUST_PROXY_HEADERS", cfg.TrustProxyHeaders)
	if err != nil {
		return Config{}, err
	}

	if cfg.RateLimitEnabled {
		if cfg.RateLimitGlobalRPS <= 0 || cfg.RateLimitPerIPRPS <= 0 {
			return Config{}, fmt.Errorf("RATE_LIMIT_*_RPS values must be greater than 0")
		}

		if cfg.RateLimitGlobalBurst <= 0 || cfg.RateLimitPerIPBurst <= 0 {
			return Config{}, fmt.Errorf("RATE_LIMIT_*_BURST values must be greater than 0")
		}

		if cfg.RateLimitIPTTL <= 0 {
			return Config{}, fmt.Errorf("RATE_LIMIT_IP_TTL must be greater than 0")
		}

		if cfg.RateLimitMaxIPEntries <= 0 {
			return Config{}, fmt.Errorf("RATE_LIMIT_MAX_IP_ENTRIES must be greater than 0")
		}
	}

	return cfg, nil
}

func getEnv(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

func parseDurationEnv(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid duration %q: %w", name, raw, err)
	}

	return d, nil
}

func parseBoolEnv(name string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}

	v := strings.ToLower(raw)
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s has invalid boolean %q", name, raw)
	}
}

func parseFloat64Env(name string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid float %q: %w", name, raw, err)
	}

	return v, nil
}

func parseIntEnv(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid integer %q: %w", name, raw, err)
	}

	return v, nil
}
