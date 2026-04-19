package middlewares

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// jwtKey reads the secret lazily so godotenv.Load() runs before the first use.
func jwtKey() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "super-secreto-cambiar-en-produccion"
	}
	return []byte(s)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// AuthMiddleware validates the Bearer JWT in the Authorization header.
func AuthMiddleware() gin.HandlerFunc {
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

		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("algoritmo no soportado: %v", t.Header["alg"])
			}
			return jwtKey(), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			return
		}

		c.Set("user_claims", claims)
		c.Set("user_role", claims.Role)
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
