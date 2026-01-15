package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests  map[string]*clientBucket
	mu        sync.RWMutex
	rate      int           // requests per window
	window    time.Duration // time window
	cleanupCh chan struct{}
}

type clientBucket struct {
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a rate limiter that allows 'rate' requests per 'window' duration.
// Reads from env vars RATE_LIMIT_REQUESTS (default 100) and RATE_LIMIT_WINDOW_SECONDS (default 60).
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
		requests:  make(map[string]*clientBucket),
		rate:      rate,
		window:    time.Duration(windowSeconds) * time.Second,
		cleanupCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// RateLimit is a middleware that limits requests per client IP.
func (rl *rateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.allowRequest(clientIP) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":      "rate limit exceeded",
				"retryAfter": int(rl.window.Seconds()),
			})
			return
		}

		c.Next()
	}
}

func (rl *rateLimiter) allowRequest(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.requests[clientIP]

	if !exists {
		rl.requests[clientIP] = &clientBucket{
			tokens:     rl.rate - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed >= rl.window {
		bucket.tokens = rl.rate
		bucket.lastRefill = now
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
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
				if now.Sub(bucket.lastRefill) > rl.window*2 {
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
