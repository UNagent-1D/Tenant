package auth

import (
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token de autorización requerido"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Formato inválido: Bearer <token>"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("algoritmo no soportado: %v", t.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
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
		if c.GetHeader("X-Internal-Key") != cfg.InternalAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid internal API key"})
			return
		}
		c.Next()
	}
}

// RoleMiddleware checks that the caller's role is in allowedRoles.
// app_admin always passes regardless of the allowed list.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sesión sin rol identificado"})
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
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Permisos insuficientes"})
	}
}
