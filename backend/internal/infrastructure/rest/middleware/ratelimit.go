package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/infrastructure/rest/apiutil"
)

type ipBucket struct {
	tokens   int
	lastFill time.Time
}

// IPRateLimiter enforces a fixed-window limit of maxRequests per window per client IP.
// It is safe for concurrent use and self-cleans stale entries every 10 minutes.
type IPRateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*ipBucket
	maxRequests int
	window      time.Duration
}

func NewIPRateLimiter(maxRequests int, window time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		clients:     make(map[string]*ipBucket),
		maxRequests: maxRequests,
		window:      window,
	}
	go rl.cleanup()
	return rl
}

func (rl *IPRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.clients[ip]
	if !exists || now.Sub(b.lastFill) >= rl.window {
		rl.clients[ip] = &ipBucket{tokens: rl.maxRequests - 1, lastFill: now}
		return true
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func (rl *IPRateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window * 2)
		for ip, b := range rl.clients {
			if b.lastFill.Before(cutoff) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a Gin middleware that rejects requests exceeding the limit.
func (rl *IPRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, apiutil.ErrorResponse{Error: apiutil.ErrorBody{
				Code:    "rate_limit_exceeded",
				Message: "Too many requests. Please try again later.",
			}})
			c.Abort()
			return
		}
		c.Next()
	}
}
