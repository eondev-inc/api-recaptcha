package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newRateLimitedRouter(t *testing.T) (*gin.Engine, *rateLimiter) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter()
	t.Cleanup(rl.Stop)

	r := gin.New()
	// The limiter keys on ClientIP, so the router must not trust a forwarded header that no
	// deployment has declared. This is the same call main.go makes from TRUSTED_PROXIES.
	if err := ConfigureTrustedProxies(r, ""); err != nil {
		t.Fatalf("ConfigureTrustedProxies: %v", err)
	}
	r.Use(rl.RateLimit())
	r.POST("/verify", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	return r, rl
}

func doLimitedRequest(t *testing.T, r http.Handler, remoteAddr, forwardedFor string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/verify", nil)
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	return rec
}

// A rejected caller must be told when to retry: a 429 without Retry-After forces clients into a
// blind retry loop.
func TestRateLimiter_RejectsOverQuotaWithRetryAfter(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS", "2")
	t.Setenv("RATE_LIMIT_WINDOW_SECONDS", "60")

	r, _ := newRateLimitedRouter(t)

	for i := 0; i < 2; i++ {
		if rec := doLimitedRequest(t, r, "", ""); rec.Code != http.StatusOK {
			t.Fatalf("request %d should be allowed, got %d", i+1, rec.Code)
		}
	}

	rec := doLimitedRequest(t, r, "", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected the third request to be rejected with 429, got %d", rec.Code)
	}

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("a 429 must carry Retry-After")
	}
	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("Retry-After must be an integer number of seconds, got %q", retryAfter)
	}
	if seconds <= 0 || seconds > 60 {
		t.Fatalf("Retry-After must be within the window, got %d", seconds)
	}
}

// The bucket key comes from ClientIP. Gin trusts every proxy by default, so without an explicit
// trusted-proxy configuration a client could mint a fresh bucket per request just by rotating
// X-Forwarded-For, which turns the rate limit into a no-op.
func TestRateLimiter_ForgedForwardedForCannotResetTheBucket(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS", "1")
	t.Setenv("RATE_LIMIT_WINDOW_SECONDS", "60")

	r, _ := newRateLimitedRouter(t)

	if rec := doLimitedRequest(t, r, "", "198.51.100.7"); rec.Code != http.StatusOK {
		t.Fatalf("the first request from the real peer should be allowed, got %d", rec.Code)
	}

	rec := doLimitedRequest(t, r, "", "203.0.113.99")
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("a forged X-Forwarded-For must not unlock the same peer's exhausted bucket, got %d", rec.Code)
	}
}

func TestRateLimiter_WindowAllowsAgainAfterExpiry(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS", "1")
	t.Setenv("RATE_LIMIT_WINDOW_SECONDS", "1")

	r, _ := newRateLimitedRouter(t)

	if rec := doLimitedRequest(t, r, "", ""); rec.Code != http.StatusOK {
		t.Fatalf("the first request should be allowed, got %d", rec.Code)
	}
	if rec := doLimitedRequest(t, r, "", ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("the second request should be rejected, got %d", rec.Code)
	}

	time.Sleep(1200 * time.Millisecond)

	if rec := doLimitedRequest(t, r, "", ""); rec.Code != http.StatusOK {
		t.Errorf("the bucket should refill after the window, got %d", rec.Code)
	}
}

// Distinct peers are the only way the bucket map grows, so the map needs a ceiling: otherwise a
// flood of distinct addresses is unbounded memory growth inside the process that is trying to
// absorb the flood.
func TestRateLimiter_CapsTheBucketMap(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS", "10")
	t.Setenv("RATE_LIMIT_WINDOW_SECONDS", "60")

	r, rl := newRateLimitedRouter(t)
	rl.maxBuckets = 2

	if rec := doLimitedRequest(t, r, "10.0.0.1:1111", ""); rec.Code != http.StatusOK {
		t.Fatalf("first peer should be allowed, got %d", rec.Code)
	}
	if rec := doLimitedRequest(t, r, "10.0.0.2:1111", ""); rec.Code != http.StatusOK {
		t.Fatalf("second peer should be allowed, got %d", rec.Code)
	}

	rec := doLimitedRequest(t, r, "10.0.0.3:1111", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("a peer beyond the bucket cap must be rejected, got %d", rec.Code)
	}

	rl.mu.RLock()
	tracked := len(rl.requests)
	rl.mu.RUnlock()
	if tracked > 2 {
		t.Errorf("the bucket map exceeded its cap: %d entries", tracked)
	}
}
