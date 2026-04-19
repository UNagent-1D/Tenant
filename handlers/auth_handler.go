package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func jwtKey() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "super-secreto-cambiar-en-produccion"
	}
	return []byte(s)
}

func LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	var u models.User
	err := config.DB.QueryRow(c.Request.Context(),
		`SELECT id, email, password_hash, role, tenant_id, is_active
		 FROM users WHERE email = $1 AND is_active = true`,
		req.Email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TenantID, &u.IsActive)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
		return
	}

	claims := &models.Claims{
		UserID:   u.ID,
		Email:    u.Email,
		TenantID: u.TenantID,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtKey())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar el token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
