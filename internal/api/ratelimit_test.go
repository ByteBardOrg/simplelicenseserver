package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientIPFromRequest(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		headers map[string][]string
		want    string
	}{
		{"peer address by default", "", map[string][]string{"X-Forwarded-For": {"1.1.1.1"}, "Cf-Connecting-Ip": {"2.2.2.2"}}, "10.0.0.9"},
		{"cloudflare header", "CF-Connecting-IP", map[string][]string{"Cf-Connecting-Ip": {"203.0.113.7"}}, "203.0.113.7"},
		{"cloudflare header absent falls back to peer", "CF-Connecting-IP", nil, "10.0.0.9"},
		{"invalid header value falls back to peer", "CF-Connecting-IP", map[string][]string{"Cf-Connecting-Ip": {"not-an-ip"}}, "10.0.0.9"},
		{"xff uses the rightmost entry", "X-Forwarded-For", map[string][]string{"X-Forwarded-For": {"6.6.6.6, 198.51.100.4"}}, "198.51.100.4"},
		{"xff across header lines", "X-Forwarded-For", map[string][]string{"X-Forwarded-For": {"6.6.6.6", "198.51.100.5"}}, "198.51.100.5"},
		{"ipv6", "X-Real-IP", map[string][]string{"X-Real-Ip": {"2001:db8::1"}}, "2001:db8::1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.RemoteAddr = "10.0.0.9:5555"
			for key, values := range tc.headers {
				for _, v := range values {
					req.Header.Add(key, v)
				}
			}
			if got := clientIPFromRequest(req, tc.header); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func limitedTestServer(opts Options) http.Handler {
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = 15 * time.Second
	}
	return NewServerWithOptions(stubService{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		map[string]struct{}{"management_key_dev_123456": {}}, opts).Routes()
}

func serve(handler http.Handler, method, path, remote string, headers map[string]string, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.RemoteAddr = remote
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestRateLimitKeepsPublicAndAdminBudgetsSeparate(t *testing.T) {
	handler := limitedTestServer(Options{RateLimitEnabled: true, RateLimitPerIPRPS: 0.001, RateLimitPerIPBurst: 1})
	body := `{"license_key":"K","fingerprint":"fp"}`
	ip := "10.0.0.1:1"

	if rr := serve(handler, http.MethodPost, "/api/v1/validate", ip, nil, body); rr.Code != http.StatusOK {
		t.Fatalf("first validate: expected 200, got %d", rr.Code)
	}
	if rr := serve(handler, http.MethodPost, "/api/v1/validate", ip, nil, body); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second validate: expected 429, got %d", rr.Code)
	}
	if rr := serve(handler, http.MethodGet, "/healthz", ip, nil, ""); rr.Code != http.StatusOK {
		t.Fatalf("admin route from the same IP must have its own budget, got %d", rr.Code)
	}
}

func TestRateLimitSkipsUIAssets(t *testing.T) {
	handler := limitedTestServer(Options{RateLimitEnabled: true, RateLimitPerIPRPS: 0.001, RateLimitPerIPBurst: 1})
	for i := 0; i < 5; i++ {
		if rr := serve(handler, http.MethodGet, "/ui/assets/app.js", "10.0.0.2:1", nil, ""); rr.Code == http.StatusTooManyRequests {
			t.Fatalf("UI asset request %d was rate limited", i)
		}
	}
}

func TestRateLimitUsesConfiguredClientIPHeader(t *testing.T) {
	handler := limitedTestServer(Options{
		RateLimitEnabled:    true,
		RateLimitPerIPRPS:   0.001,
		RateLimitPerIPBurst: 1,
		ClientIPHeader:      "CF-Connecting-IP",
	})
	proxy := "172.18.0.5:443" // every request arrives from the same proxy

	if rr := serve(handler, http.MethodGet, "/healthz", proxy, map[string]string{"CF-Connecting-IP": "203.0.113.1"}, ""); rr.Code != http.StatusOK {
		t.Fatalf("client 1: expected 200, got %d", rr.Code)
	}
	if rr := serve(handler, http.MethodGet, "/healthz", proxy, map[string]string{"CF-Connecting-IP": "203.0.113.2"}, ""); rr.Code != http.StatusOK {
		t.Fatalf("client 2 must not share client 1's budget, got %d", rr.Code)
	}
	if rr := serve(handler, http.MethodGet, "/healthz", proxy, map[string]string{"CF-Connecting-IP": "203.0.113.1"}, ""); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("client 1 again: expected 429, got %d", rr.Code)
	}
}

func TestRateLimiterEvictsOldestWhenFull(t *testing.T) {
	rl := newIPRateLimiter(rateLimiterConfig{enabled: true, perIPRPS: 1, perIPBurst: 1, ipTTL: time.Hour, maxIPEntries: 2})
	for _, key := range []string{"a", "b", "c"} {
		if !rl.allow(key) {
			t.Fatalf("new key %q refused; a full table must evict instead", key)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(rl.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(rl.entries))
	}
	if _, ok := rl.entries["a"]; ok {
		t.Fatalf("expected the least recently seen key to be evicted")
	}
}

func TestGlobalLimitOffByDefault(t *testing.T) {
	defaults := DefaultOptions()
	if defaults.RateLimitGlobalRPS != 0 {
		t.Fatalf("expected global limit off by default, got %v", defaults.RateLimitGlobalRPS)
	}
	rl := newIPRateLimiter(rateLimiterConfig{enabled: true, perIPRPS: 1, perIPBurst: 1})
	if rl.global != nil {
		t.Fatalf("expected no global limiter when globalRPS is 0")
	}
}

func TestAuthFailureLockout(t *testing.T) {
	handler := limitedTestServer(Options{AuthFailureMaxAttempts: 3, AuthFailureWindow: time.Minute})
	bad := map[string]string{"Authorization": "Bearer wrong_key_123456789"}
	good := map[string]string{"Authorization": "Bearer management_key_dev_123456"}
	attacker := "10.0.0.66:1"

	for i := 0; i < 3; i++ {
		if rr := serve(handler, http.MethodGet, "/management/slugs", attacker, bad, ""); rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rr.Code)
		}
	}

	rr := serve(handler, http.MethodGet, "/management/slugs", attacker, good, "")
	if rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("expected 429 with Retry-After even for a correct key during lockout, got %d", rr.Code)
	}

	// Provisioning routes share the lockout.
	if rr := serve(handler, http.MethodPost, "/api/v1/generate", attacker, good, `{"slug":"default"}`); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected provisioning route to be locked too, got %d", rr.Code)
	}

	// Other clients are unaffected.
	if rr := serve(handler, http.MethodGet, "/management/slugs", "10.0.0.7:1", good, ""); rr.Code != http.StatusOK {
		t.Fatalf("other IP: expected 200, got %d", rr.Code)
	}
}

func TestAuthFailureLockoutIgnoresMissingKeys(t *testing.T) {
	handler := limitedTestServer(Options{AuthFailureMaxAttempts: 2, AuthFailureWindow: time.Minute})
	ip := "10.0.0.8:1"
	for i := 0; i < 5; i++ {
		if rr := serve(handler, http.MethodGet, "/management/slugs", ip, nil, ""); rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for a missing key, got %d", rr.Code)
		}
	}
	if rr := serve(handler, http.MethodGet, "/management/slugs", ip, map[string]string{"Authorization": "Bearer management_key_dev_123456"}, ""); rr.Code != http.StatusOK {
		t.Fatalf("requests without a key must not count toward the lockout, got %d", rr.Code)
	}
}

func TestAuthFailureWindowExpires(t *testing.T) {
	limiter := newAuthFailureLimiter(1, time.Minute, 10)
	start := time.Now()
	limiter.recordFailure("ip", start)
	if limiter.blockedFor("ip", start.Add(30*time.Second)) <= 0 {
		t.Fatalf("expected lockout inside the window")
	}
	if limiter.blockedFor("ip", start.Add(61*time.Second)) != 0 {
		t.Fatalf("expected lockout to end with the window")
	}
}
