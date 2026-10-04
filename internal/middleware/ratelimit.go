package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// defaultMaxBuckets bounds the memory a flood of distinct peers can make this middleware hold.
// Each entry is small, so the ceiling only matters under a distributed source; reaching it makes
// a new peer wait for the next window instead of allocating.
const defaultMaxBuckets = 100_000

// rateLimiter is a fixed-window counter per client. It is deliberately not a token bucket: the
// allowance refills in one step at each window boundary and unused requests do not carry over.
type rateLimiter struct {
	requests   map[string]*clientBucket
	mu         sync.RWMutex
	rate       int           // requests per window
	window     time.Duration // time window
	maxBuckets int           // ceiling on tracked clients
	cleanupCh  chan struct{}
}

type clientBucket struct {
	remaining   int
	windowStart time.Time
}

// NewRateLimiter creates a rate limiter that allows 'rate' requests per 'window' duration.
// Reads from env vars RATE_LIMIT_REQUESTS (default 100) and RATE_LIMIT_WINDOW_SECONDS (default 60).
//
// The counter is keyed on the client address as reported by ClientIP, so the router must be
// configured with ConfigureTrustedProxies before this is useful: with Gin's default trust set
// every caller could choose its own key.
func NewRateLimiter() *rateLimiter {
	rate := 100
	windowSeconds := 60

	if envRate := os.Getenv("RATE_LIMIT_REQUESTS"); envRate != "" {
		if parsed, err := strconv.Atoi(envRate); err == nil && parsed > 0 {
			rate = parsed
		}
	}

	if envWindow := os.Getenv("RATE_LIMIT_WINDOW_SECONDS"); envWindow != "" {
		if parsed, err := strconv.Atoi(envWindow); err == nil && parsed > 0 {
			windowSeconds = parsed
		}
	}

	rl := &rateLimiter{
		requests:   make(map[string]*clientBucket),
		rate:       rate,
		window:     time.Duration(windowSeconds) * time.Second,
		maxBuckets: defaultMaxBuckets,
		cleanupCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// RateLimit is a middleware that limits requests per client IP.
func (rl *rateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := rl.allowRequest(c.ClientIP())
		if !allowed {
			seconds := retryAfterSeconds(retryAfter)
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":      "rate limit exceeded",
				"retryAfter": seconds,
			})
			return
		}

		c.Next()
	}
}

// allowRequest reports whether the client may proceed, and how long to wait when it may not.
func (rl *rateLimiter) allowRequest(clientIP string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.requests[clientIP]

	if !exists {
		if len(rl.requests) >= rl.maxBuckets {
			// ponytail: refuse the new bucket instead of evicting an old one; the map only fills
			// under a flood of distinct peers, and eviction would let the flood evict itself back
			// in. An LRU is the upgrade path if a real workload ever hits this.
			return false, rl.window
		}
		rl.requests[clientIP] = &clientBucket{remaining: rl.rate - 1, windowStart: now}
		return true, 0
	}

	// The allowance is granted in one step when the window rolls over.
	if elapsed := now.Sub(bucket.windowStart); elapsed >= rl.window {
		bucket.remaining = rl.rate
		bucket.windowStart = now
	}

	if bucket.remaining > 0 {
		bucket.remaining--
		return true, 0
	}

	return false, rl.window - now.Sub(bucket.windowStart)
}

// retryAfterSeconds converts the wait into the whole seconds Retry-After requires.
func retryAfterSeconds(wait time.Duration) int {
	return max(int(wait.Seconds()), 1)
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, bucket := range rl.requests {
				if now.Sub(bucket.windowStart) > rl.window*2 {
					delete(rl.requests, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.cleanupCh:
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (rl *rateLimiter) Stop() {
	close(rl.cleanupCh)
}
