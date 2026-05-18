package middlewares

// IP-keyed token-bucket throttle for the public auth surface.
//
// The login endpoint sits behind bcrypt (cost 10), so each verification
// costs ~50ms of CPU. Without throttling, a small fleet of clients can
// keep every Tenant worker bcrypt'ing in parallel and turn a leaked
// credential list into an offline-time attack.
//
// This middleware burns one token per request from a per-IP bucket; when
// the bucket is empty the response is HTTP 429 with `Retry-After`. The
// bucket is process-local and reset on restart, which is acceptable for
// a single-replica deployment. If multiple replicas are added later, the
// same call site can swap the in-memory map for a Redis-backed bucket
// without touching the handler.

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter implements a per-key token bucket. Safe for concurrent use.
type RateLimiter struct {
	mu           sync.Mutex
	buckets      map[string]*tokenBucket
	capacity     float64
	refillPerSec float64
}

// NewRateLimiter constructs a limiter with the given capacity (max burst)
// and refill rate. Both values are clamped to a sane minimum.
func NewRateLimiter(capacity, refillPerSec float64) *RateLimiter {
	if capacity < 1 {
		capacity = 1
	}
	if refillPerSec < 0.0001 {
		refillPerSec = 0.0001
	}
	return &RateLimiter{
		buckets:      make(map[string]*tokenBucket),
		capacity:     capacity,
		refillPerSec: refillPerSec,
	}
}

// Allow returns (true, 0) if a token was consumed, or (false, wait) if
// the caller should retry after `wait`.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	b, ok := r.buckets[key]
	if !ok {
		b = &tokenBucket{tokens: r.capacity, lastRefill: now}
		r.buckets[key] = b
	}
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * r.refillPerSec
	if b.tokens > r.capacity {
		b.tokens = r.capacity
	}
	b.lastRefill = now
	if b.tokens >= 1 {
		b.tokens -= 1
		return true, 0
	}
	missing := 1 - b.tokens
	wait := time.Duration(missing/r.refillPerSec*float64(time.Second)) + time.Millisecond
	return false, wait
}

// LoginRateLimiter returns a middleware that throttles requests by client
// IP. It is intended for `/auth/login` to blunt brute-force / credential
// stuffing attacks against bcrypt-protected accounts.
//
// Limits are read from env (with safe defaults) so they can be tuned per
// environment without rebuilding the image:
//
//   AUTH_RATE_LIMIT_BURST    default 5  — max attempts in a burst per IP
//   AUTH_RATE_LIMIT_PER_SEC  default 0.0333 (1 per 30s) — refill rate
func LoginRateLimiter() gin.HandlerFunc {
	burst := envFloat("AUTH_RATE_LIMIT_BURST", 5)
	perSec := envFloat("AUTH_RATE_LIMIT_PER_SEC", 1.0/30.0)
	limiter := NewRateLimiter(burst, perSec)

	return func(c *gin.Context) {
		key := clientIP(c)
		ok, retry := limiter.Allow(key)
		if !ok {
			retrySecs := int(retry.Seconds())
			if retrySecs < 1 {
				retrySecs = 1
			}
			c.Header("Retry-After", strconv.Itoa(retrySecs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":             "Demasiados intentos de inicio de sesión. Por favor, espere antes de reintentar.",
				"retry_after_secs":  retrySecs,
			})
			return
		}
		c.Next()
	}
}

// clientIP prefers Cloudflare's CF-Connecting-IP header (set when the
// service is exposed via Cloudflare Tunnel) and falls back to
// X-Forwarded-For and finally the socket peer. Behind the reverse proxy,
// the socket peer is always the proxy itself — never use it as the sole
// throttle key in production.
func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may be a comma list; take the first element.
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return trimSpace(xff[:i])
			}
		}
		return trimSpace(xff)
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil || host == "" {
		return c.Request.RemoteAddr
	}
	return host
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func envFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
