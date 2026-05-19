package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates the Bearer JWT and injects claims into context.
// Sets "user_claims", "user_role", "user_id", and "tenant_id" (string keys).
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	secret := []byte(cfg.JWTSecret)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unsupported signing algorithm: %v", t.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
			return
		}

		c.Set("user_claims", claims)
		c.Set("user_role", claims.Role)
		c.Set("user_id", claims.UserID)
		if claims.TenantID != nil {
			c.Set("tenant_id", *claims.TenantID)
		}
		c.Next()
	}
}

// InternalKeyMiddleware validates the X-Internal-Key header for service-to-service endpoints.
func InternalKeyMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Internal-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-Internal-Key header"})
			return
		}
		if key != cfg.InternalAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid internal API key"})
			return
		}
		c.Next()
	}
}

// OnlyAdminMiddleware enforces app_admin-only access and returns denyMsg on 403.
func OnlyAdminMiddleware(denyMsg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
			return
		}
		if roleVal.(string) == "app_admin" {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": denyMsg})
	}
}

// RoleMiddleware checks that the caller's role is in allowedRoles.
// app_admin always passes regardless of the allowed list.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
			return
		}
		userRole := roleVal.(string)

		if userRole == "app_admin" {
			c.Next()
			return
		}
		for _, r := range allowedRoles {
			if userRole == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
