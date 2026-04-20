package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateLimiterConfig struct {
	enabled           bool
	globalRPS         float64
	globalBurst       int
	perIPRPS          float64
	perIPBurst        int
	ipTTL             time.Duration
	maxIPEntries      int
	trustProxyHeaders bool
}

type ipRateLimiter struct {
	enabled           bool
	global            *rate.Limiter
	perIPRPS          rate.Limit
	perIPBurst        int
	ipTTL             time.Duration
	maxIPEntries      int
	trustProxyHeaders bool

	mu          sync.Mutex
	entries     map[string]*ipLimiterEntry
	lastCleanup time.Time
}

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(cfg rateLimiterConfig) *ipRateLimiter {
	rl := &ipRateLimiter{
		enabled:           cfg.enabled,
		ipTTL:             cfg.ipTTL,
		maxIPEntries:      cfg.maxIPEntries,
		trustProxyHeaders: cfg.trustProxyHeaders,
		entries:           make(map[string]*ipLimiterEntry),
	}

	if !cfg.enabled {
		return rl
	}

	rl.global = rate.NewLimiter(rate.Limit(cfg.globalRPS), cfg.globalBurst)
	rl.perIPRPS = rate.Limit(cfg.perIPRPS)
	rl.perIPBurst = cfg.perIPBurst

	if rl.ipTTL <= 0 {
		rl.ipTTL = 10 * time.Minute
	}

	if rl.maxIPEntries <= 0 {
		rl.maxIPEntries = 10000
	}

	return rl
}

func (r *ipRateLimiter) allow(req *http.Request) bool {
	if !r.enabled {
		return true
	}

	if r.global != nil && !r.global.Allow() {
		return false
	}

	now := time.Now().UTC()
	ip := clientIP(req, r.trustProxyHeaders)
	if ip == "" {
		ip = "unknown"
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.maybeCleanup(now)

	entry, ok := r.entries[ip]
	if !ok {
		if len(r.entries) >= r.maxIPEntries {
			return false
		}

		entry = &ipLimiterEntry{
			limiter:  rate.NewLimiter(r.perIPRPS, r.perIPBurst),
			lastSeen: now,
		}
		r.entries[ip] = entry
	}

	entry.lastSeen = now
	return entry.limiter.Allow()
}

func (r *ipRateLimiter) maybeCleanup(now time.Time) {
	if r.lastCleanup.IsZero() {
		r.lastCleanup = now
		return
	}

	cleanupInterval := r.ipTTL / 2
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}

	if now.Sub(r.lastCleanup) < cleanupInterval {
		return
	}

	cutoff := now.Add(-r.ipTTL)
	for ip, entry := range r.entries {
		if entry.lastSeen.Before(cutoff) {
			delete(r.entries, ip)
		}
	}

	r.lastCleanup = now
}

func clientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				if ip := strings.TrimSpace(parts[0]); ip != "" {
					return ip
				}
			}
		}

		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}
