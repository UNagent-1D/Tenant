package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func newRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestIDMiddleware injects X-Request-ID into request and response.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = newRequestID()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

// StructuredLoggerMiddleware emits one JSON log line per request.
// Reads "user_id" and "tenant_id" string keys set by AuthMiddleware.
func StructuredLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		rid, _ := c.Get("request_id")
		log.Printf(
			`{"request_id":%q,"method":%q,"path":%q,"status":%d,"latency_ms":%d,"user_id":%q,"tenant_id":%q}`,
			rid, c.Request.Method, c.Request.URL.Path,
			c.Writer.Status(), latency.Milliseconds(),
			c.GetString("user_id"), c.GetString("tenant_id"),
		)
	}
}
