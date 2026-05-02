package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(getEnv("JWT_SECRET", "super-secreto-cambiar-en-produccion"))

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// scanLoginRow runs the login query with one transparent retry on driver-bad-conn
// errors. lib/pq does not always surface a stale Supabase pooler connection as
// driver.ErrBadConn on the first call, so we Ping + retry once before giving up.
func scanLoginRow(query, email string, user *models.User, uTenant *models.UserTenant) error {
	scan := func() error {
		return config.DB.QueryRow(query, email).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Active,
			&uTenant.TenantID, &uTenant.Role,
		)
	}

	err := scan()
	if err == nil || err == sql.ErrNoRows {
		return err
	}
	if !errors.Is(err, driver.ErrBadConn) {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if pingErr := config.DB.PingContext(ctx); pingErr != nil {
		return pingErr
	}
	return scan()
}

// LoginHandler controla la autenticación, verifica clave y emite JWT en base al rol de BD
func LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload de solicitud inválido"})
		return
	}

	var user models.User
	var uTenant models.UserTenant

	// Buscamos al usuario en la base de datos, uniéndolo con la tabla de user_tenants para obtener su rol y tenant
	// En un diseño donde el usuario pertenece a un solo tenant, esto trae 1 registro (o app_admin que trae null)
	query := `
		SELECT u.id, u.email, u.password_hash, u.is_active, ut.tenant_id, ut.role
		FROM users u
		LEFT JOIN user_tenants ut ON u.id = ut.user_id
		WHERE u.email = $1 AND u.is_active = true
		LIMIT 1;
	`

	err := scanLoginRow(query, req.Email, &user, &uTenant)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno del servidor", "detalle": err.Error()})
		return
	}

	// Comparamos el hash de la clave almacenado contra la contraseña en texto plano
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
		return
	}

	// Preparamos los claims del JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.Claims{
		UserID:   user.ID,
		Email:    user.Email,
		TenantID: uTenant.TenantID,
		Role:     uTenant.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Firmamos el JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar el token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   tokenString,
		"message": "Bienvenido, autenticación exitosa",
	})
}
