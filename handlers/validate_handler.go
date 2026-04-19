package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ValidateToken validates a JWT and returns the claims in the format expected by microservices.
// Used by conversation-chat (and any other service) as its AUTH_SERVICE_URL/validate endpoint.
func ValidateToken(c *gin.Context) {
	var tokenString string

	// Prefer Authorization header: "Bearer <token>"
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "bearer ") {
		tokenString = authHeader[7:]
	}

	// Fallback: read from request body { "token": "..." }
	if tokenString == "" {
		var body struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&body); err == nil && body.Token != "" {
			tokenString = body.Token
		}
	}

	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token no proporcionado"})
		return
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo de firma no soportado: %v", token.Header["alg"])
		}
		return jwtSecret, nil // jwtSecret is defined in auth_handler.go (same package)
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     claims.UserID,
		"email":       claims.Email,
		"role":        claims.Role,
		"tenant_id":   claims.TenantID,
		"tenant_slug": nil,
	})
}
