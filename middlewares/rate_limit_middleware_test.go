package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newTestRouter(burst, perSec float64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/login", LoginRateLimiter(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimiterAllowsUpToBurst(t *testing.T) {
	t.Setenv("AUTH_RATE_LIMIT_BURST", "3")
	t.Setenv("AUTH_RATE_LIMIT_PER_SEC", "0.0001")
	r := newTestRouter(3, 0.0001)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i+1, w.Code)
		}
	}
}

func TestRateLimiterRejectsAfterBurst(t *testing.T) {
	t.Setenv("AUTH_RATE_LIMIT_BURST", "3")
	t.Setenv("AUTH_RATE_LIMIT_PER_SEC", "0.0001")
	r := newTestRouter(3, 0.0001)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("setup request %d: got %d, want 200", i+1, w.Code)
		}
	}

	// Burst exhausted — next request must be 429.
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after burst, got %d", w.Code)
	}
}

func TestRateLimiter429CarriesRetryAfterHeader(t *testing.T) {
	t.Setenv("AUTH_RATE_LIMIT_BURST", "1")
	t.Setenv("AUTH_RATE_LIMIT_PER_SEC", "0.0333") // 1 per 30s default
	r := newTestRouter(1, 0.0333)

	ip := "10.0.0.3:12345"
	// Exhaust
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = ip
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Trigger 429
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header on 429, got none")
	}
}

func TestRateLimiterIPsAreIsolated(t *testing.T) {
	t.Setenv("AUTH_RATE_LIMIT_BURST", "1")
	t.Setenv("AUTH_RATE_LIMIT_PER_SEC", "0.0001")
	r := newTestRouter(1, 0.0001)

	// Exhaust IP-A
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.1.0.1:1"
	r.ServeHTTP(httptest.NewRecorder(), req)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.1.0.1:1"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("IP-A: expected 429, got %d", w.Code)
	}

	// IP-B is unaffected
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.1.0.2:1"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("IP-B: expected 200, got %d", w.Code)
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	// Very fast refill: 1_000 tokens/s = effectively immediate.
	t.Setenv("AUTH_RATE_LIMIT_BURST", "1")
	t.Setenv("AUTH_RATE_LIMIT_PER_SEC", "1000")
	r := newTestRouter(1, 1000)

	ip := "10.2.0.1:1"
	// Exhaust
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = ip
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Sleep 2ms — enough for a 1000 tokens/s rate to deliver 2 tokens.
	time.Sleep(2 * time.Millisecond)

	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after refill, got %d", w.Code)
	}
}

func TestRateLimiterTokenBucketAllowExact(t *testing.T) {
	rl := NewRateLimiter(5, 0.0001)
	for i := 0; i < 5; i++ {
		ok, _ := rl.Allow("key")
		if !ok {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	ok, wait := rl.Allow("key")
	if ok {
		t.Fatal("6th attempt should be denied")
	}
	if wait <= 0 {
		t.Fatalf("expected positive wait duration, got %v", wait)
	}
}

func TestRateLimiterCFConnectingIPHeaderTakesPriority(t *testing.T) {
	t.Setenv("AUTH_RATE_LIMIT_BURST", "1")
	t.Setenv("AUTH_RATE_LIMIT_PER_SEC", "0.0001")
	r := newTestRouter(1, 0.0001)

	cfIP := "203.0.113.10"

	// Exhaust the CF-Connecting-IP bucket
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.Header.Set("CF-Connecting-IP", cfIP)
	req.RemoteAddr = "127.0.0.1:1" // Proxy IP, different from CF header
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Second request with same CF-Connecting-IP → 429 (same bucket)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.Header.Set("CF-Connecting-IP", cfIP)
	req.RemoteAddr = "127.0.0.1:2"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for same CF-Connecting-IP, got %d", w.Code)
	}

	// A different CF IP is unaffected
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.Header.Set("CF-Connecting-IP", "203.0.113.11")
	req.RemoteAddr = "127.0.0.1:3"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for different CF-Connecting-IP, got %d", w.Code)
	}
}
