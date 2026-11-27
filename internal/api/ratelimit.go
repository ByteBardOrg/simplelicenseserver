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
	enabled bool
	// globalRPS caps all traffic together. 0 disables it: a cap shared by every client lets a handful of
	// abusive IPs lock everyone out, and an edge proxy is better placed to absorb floods.
	globalRPS    float64
	globalBurst  int
	perIPRPS     float64
	perIPBurst   int
	ipTTL        time.Duration
	maxIPEntries int
}

// ipRateLimiter is a token bucket per key, where a key is a traffic class plus a client IP.
type ipRateLimiter struct {
	enabled      bool
	global       *rate.Limiter
	perIPRPS     rate.Limit
	perIPBurst   int
	ipTTL        time.Duration
	maxIPEntries int

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
		enabled:      cfg.enabled,
		ipTTL:        cfg.ipTTL,
		maxIPEntries: cfg.maxIPEntries,
		entries:      make(map[string]*ipLimiterEntry),
	}

	if !cfg.enabled {
		return rl
	}

	if cfg.globalRPS > 0 {
		burst := cfg.globalBurst
		if burst <= 0 {
			burst = int(cfg.globalRPS)
		}
		rl.global = rate.NewLimiter(rate.Limit(cfg.globalRPS), burst)
	}
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

func (r *ipRateLimiter) allow(key string) bool {
	if !r.enabled {
		return true
	}

	if r.global != nil && !r.global.Allow() {
		return false
	}

	now := time.Now().UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.maybeCleanup(now)

	entry, ok := r.entries[key]
	if !ok {
		if len(r.entries) >= r.maxIPEntries {
			// Make room rather than refusing every new client: with a full table, refusing would let
			// anyone who can present many addresses lock out all new visitors.
			r.evictExpiredOrOldest(now)
		}

		entry = &ipLimiterEntry{
			limiter:  rate.NewLimiter(r.perIPRPS, r.perIPBurst),
			lastSeen: now,
		}
		r.entries[key] = entry
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

	r.removeExpired(now)
	r.lastCleanup = now
}

func (r *ipRateLimiter) removeExpired(now time.Time) {
	cutoff := now.Add(-r.ipTTL)
	for key, entry := range r.entries {
		if entry.lastSeen.Before(cutoff) {
			delete(r.entries, key)
		}
	}
}

func (r *ipRateLimiter) evictExpiredOrOldest(now time.Time) {
	r.removeExpired(now)
	if len(r.entries) < r.maxIPEntries {
		return
	}

	oldestKey := ""
	var oldest time.Time
	for key, entry := range r.entries {
		if oldestKey == "" || entry.lastSeen.Before(oldest) {
			oldestKey, oldest = key, entry.lastSeen
		}
	}
	delete(r.entries, oldestKey)
}

// Traffic classes get separate per-IP budgets, so heavy license checks from an office can't lock
// its admin out of the management UI, and the other way round.
const (
	rateClassPublic = "public"
	rateClassAdmin  = "admin"
)

func rateLimitClass(path string) (class string, limited bool) {
	switch {
	case path == "/ui" || strings.HasPrefix(path, "/ui/"):
		// The bundled UI's static files: cheap to serve, and a single page load fetches many of them.
		return "", false
	case path == "/api/v1/activate", path == "/api/v1/validate", path == "/api/v1/deactivate":
		return rateClassPublic, true
	default:
		return rateClassAdmin, true
	}
}

// clientIPFromRequest returns the address to rate limit and log. By default that is the TCP peer.
// When header is set (CLIENT_IP_HEADER), the IP is read from that header instead:
//   - "X-Forwarded-For": the rightmost entry, which is the one the proxy in front of us appended.
//     The leftmost entries are whatever the client sent and must never be trusted.
//   - any other header, such as "CF-Connecting-IP" or "X-Real-IP": its value, which the proxy sets.
//
// Values that are not valid IPs are ignored. Only set a header when every request reaches the
// server through a proxy that sets it, otherwise clients can choose their own address.
func clientIPFromRequest(r *http.Request, header string) string {
	if header != "" {
		if ip := ipFromHeader(r, header); ip != "" {
			return ip
		}
	}

	return remoteIP(r)
}

func ipFromHeader(r *http.Request, header string) string {
	values := r.Header.Values(header)
	if len(values) == 0 {
		return ""
	}

	candidate := ""
	if strings.EqualFold(header, "X-Forwarded-For") {
		parts := strings.Split(strings.Join(values, ","), ",")
		for i := len(parts) - 1; i >= 0; i-- {
			if part := strings.TrimSpace(parts[i]); part != "" {
				candidate = part
				break
			}
		}
	} else {
		candidate = strings.TrimSpace(strings.Split(values[0], ",")[0])
	}

	if ip := net.ParseIP(candidate); ip != nil {
		return ip.String()
	}
	return ""
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}

// authFailureLimiter locks an IP out of key-protected routes after repeated wrong keys.
// Requests with no key at all are not counted; only presented-and-wrong keys are.
type authFailureLimiter struct {
	maxAttempts int
	window      time.Duration
	maxEntries  int

	mu      sync.Mutex
	entries map[string]*authFailureEntry
}

type authFailureEntry struct {
	count       int
	windowStart time.Time
}

func newAuthFailureLimiter(maxAttempts int, window time.Duration, maxEntries int) *authFailureLimiter {
	if maxAttempts <= 0 {
		return nil
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if maxEntries <= 0 {
		maxEntries = 10000
	}
	return &authFailureLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		maxEntries:  maxEntries,
		entries:     make(map[string]*authFailureEntry),
	}
}

// blockedFor reports how much longer ip is locked out, or 0 if it is not.
func (a *authFailureLimiter) blockedFor(ip string, now time.Time) time.Duration {
	if a == nil {
		return 0
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.entries[ip]
	if !ok {
		return 0
	}
	ends := entry.windowStart.Add(a.window)
	if !now.Before(ends) {
		delete(a.entries, ip)
		return 0
	}
	if entry.count < a.maxAttempts {
		return 0
	}
	return ends.Sub(now)
}

func (a *authFailureLimiter) recordFailure(ip string, now time.Time) {
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.entries[ip]
	if ok && !now.Before(entry.windowStart.Add(a.window)) {
		ok = false
	}
	if !ok {
		if len(a.entries) >= a.maxEntries {
			a.evict(now)
		}
		entry = &authFailureEntry{windowStart: now}
		a.entries[ip] = entry
	}
	entry.count++
}

func (a *authFailureLimiter) evict(now time.Time) {
	oldestKey := ""
	var oldest time.Time
	for key, entry := range a.entries {
		if !now.Before(entry.windowStart.Add(a.window)) {
			delete(a.entries, key)
			continue
		}
		if oldestKey == "" || entry.windowStart.Before(oldest) {
			oldestKey, oldest = key, entry.windowStart
		}
	}
	if len(a.entries) >= a.maxEntries && oldestKey != "" {
		delete(a.entries, oldestKey)
	}
}
