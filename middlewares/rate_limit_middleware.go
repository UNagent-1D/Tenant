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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
// Every rejection is mirrored to Compliance /v1/event as a fire-and-forget
// HTTP call so the platform keeps a durable audit trail of throttle events.
// Failures of that call are logged and dropped; they never affect the
// 429 response.
//
// Limits are read from env (with safe defaults):
//
//	AUTH_RATE_LIMIT_BURST    default 5         max attempts in a burst per IP
//	AUTH_RATE_LIMIT_PER_SEC  default 0.0333    refill rate (one per 30s)
//	COMPLIANCE_URL           default empty     when set, 429s are audited
func LoginRateLimiter() gin.HandlerFunc {
	burst := envFloat("AUTH_RATE_LIMIT_BURST", 5)
	perSec := envFloat("AUTH_RATE_LIMIT_PER_SEC", 1.0/30.0)
	limiter := NewRateLimiter(burst, perSec)
	complianceURL := os.Getenv("COMPLIANCE_URL")

	return func(c *gin.Context) {
		key := clientIP(c)
		ok, retry := limiter.Allow(key)
		if !ok {
			retrySecs := int(retry.Seconds())
			if retrySecs < 1 {
				retrySecs = 1
			}
			c.Header("Retry-After", strconv.Itoa(retrySecs))
			emitRateLimitAudit(complianceURL, key, retrySecs)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":            fmt.Sprintf("Demasiados intentos de inicio de sesión. Espera %d segundos e intenta de nuevo.", retrySecs),
				"retry_after_secs": retrySecs,
			})
			return
		}
		c.Next()
	}
}

// emitRateLimitAudit posts a single audit row to Compliance /v1/event in
// a detached goroutine. The tenant_id is intentionally "unknown" because
// the throttle fires before the login request has been authenticated.
func emitRateLimitAudit(complianceURL, clientIP string, retrySecs int) {
	if complianceURL == "" {
		return
	}
	body := map[string]any{
		"level":     "WARN",
		"tenant_id": "unknown",
		"component": "tenant",
		"action":    "RATE_LIMITED",
		"metadata": map[string]any{
			"scope":            "login",
			"client_ip":        clientIP,
			"retry_after_secs": retrySecs,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			complianceURL+"/v1/event", bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}()
}

// clientIP returns the best-effort real client IP.
//
// Proxy headers (CF-Connecting-IP, X-Forwarded-For) are only trusted when
// TRUST_PROXY_HEADERS=true is set in the environment.  In that mode the
// service is assumed to be behind a trusted edge (e.g. Cloudflare) that
// strips or overwrites these headers before forwarding.  When the flag is
// absent or false (default, local dev), only the socket peer address is
// used so that clients cannot bypass rate limiting by spoofing headers.
func clientIP(c *gin.Context) string {
	if os.Getenv("TRUST_PROXY_HEADERS") == "true" {
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
