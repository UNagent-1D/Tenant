package middlewares

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/UNagent-1D/Tenant/models"
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
func StructuredLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		userID, tenantID := "", ""
		if v, ok := c.Get("user_claims"); ok {
			if cl, ok := v.(*models.Claims); ok {
				userID = cl.UserID
				if cl.TenantID != nil {
					tenantID = *cl.TenantID
				}
			}
		}
		rid, _ := c.Get("request_id")

		log.Printf(
			`{"request_id":%q,"method":%q,"path":%q,"status":%d,"latency_ms":%d,"user_id":%q,"tenant_id":%q}`,
			rid, c.Request.Method, c.Request.URL.Path,
			c.Writer.Status(), latency.Milliseconds(),
			userID, tenantID,
		)
	}
}
