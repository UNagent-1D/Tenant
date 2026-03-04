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

var jwtSecret = []byte(getEnv("JWT_SECRET", "super-secreto-cambiar-en-produccion"))

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// AuthMiddleware extrae y valida el JWT desde la cabecera Authorization
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token de autorización requerido en los headers"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "El formato del token debe ser 'Bearer <token>'"})
			return
		}

		tokenString := parts[1]
		claims := &models.Claims{}

		// Parsear el token e inferir su validez basado en la clave secreta y el algoritmo
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("algoritmo de firma no soportado: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			return
		}

		// Injectar claims a la sesión para que los siguientes handlers puedan acceder al usuario
		c.Set("user_claims", claims)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// RoleMiddleware revisa si el rol en los JWT claims (del middleware previo) está en los roles permitidos por el endpoint
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sesión sin roles identificados"})
			return
		}

		userRole := roleVal.(string)
		hasPermission := false

		// Validamos el array de roles que permiten acceso al recurso
		for _, role := range allowedRoles {
			if userRole == role {
				hasPermission = true
				break
			}
		}

		// Por norma de negocio, app_admin siempre sobre-escribe cualquier permiso particular de middleware y es omnipresente.
		if userRole == "app_admin" {
			hasPermission = true
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acceso denegado: permisos insuficientes para la acción solicitada"})
			return
		}

		c.Next()
	}
}
