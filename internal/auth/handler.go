package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// scanUser runs the login query with one transparent retry on stale-connection
// errors. pgxpool retries internally when SafeToRetry is set, but Supabase's
// Session Pooler can surface bad conns that slip past that check, so we add
// an explicit second attempt after a fresh Ping.
func scanUser(ctx context.Context, email string, u *User) error {
	const q = `SELECT id, email, password_hash, role, tenant_id, is_active
	            FROM users WHERE email = $1 AND is_active = true`
	scan := func() error {
		return db.Pool.QueryRow(ctx, q, email).Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TenantID, &u.IsActive,
		)
	}
	err := scan()
	if err == nil || err == pgx.ErrNoRows {
		return err
	}
	if !pgconn.SafeToRetry(err) {
		return err
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if pingErr := db.Pool.Ping(pingCtx); pingErr != nil {
		return pingErr
	}
	return scan()
}

func LoginHandler(cfg *config.Config) gin.HandlerFunc {
	secret := []byte(cfg.JWTSecret)
	expiry := cfg.JWTExpiry
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
			return
		}

		var u User
		err := scanUser(c.Request.Context(), req.Email, &u)
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

		claims := &Claims{
			UserID:   u.ID,
			Email:    u.Email,
			TenantID: u.TenantID,
			Role:     u.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar el token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

func GetUsers(c *gin.Context) {
	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT id, email, role, tenant_id, is_active, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar usuarios"})
		return
	}
	defer rows.Close()

	users := []UserResponse{}
	for rows.Next() {
		var u UserResponse
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer usuarios"})
			return
		}
		users = append(users, u)
	}
	c.JSON(http.StatusOK, users)
}

func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	caller := c.MustGet("user_claims").(*Claims)

	switch caller.Role {
	case "app_admin":
		if (req.Role == "tenant_admin" || req.Role == "tenant_operator") && req.TenantID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id requerido para este rol"})
			return
		}
		if req.Role == "app_admin" {
			req.TenantID = nil
		}
	case "tenant_admin":
		if req.Role != "tenant_operator" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant_admin solo puede crear tenant_operator"})
			return
		}
		req.TenantID = caller.TenantID
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Rol no autorizado"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al encriptar contraseña"})
		return
	}

	var newID string
	err = db.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO users (email, password_hash, role, tenant_id)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		req.Email, string(hash), req.Role, req.TenantID,
	).Scan(&newID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "El email ya está registrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear usuario"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user_id": newID})
}

func UpdateUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	caller := c.MustGet("user_claims").(*Claims)
	userID := c.Param("uid")

	if caller.Role == "tenant_admin" {
		var targetTenantID *string
		err := db.Pool.QueryRow(c.Request.Context(),
			"SELECT tenant_id FROM users WHERE id = $1", userID,
		).Scan(&targetTenantID)
		if err != nil || targetTenantID == nil || *targetTenantID != *caller.TenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acceso denegado"})
			return
		}
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.Role != nil {
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", i))
		args = append(args, *req.Role)
		i++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *req.IsActive)
		i++
	}
	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sin campos para actualizar"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $%d RETURNING id, email, role, tenant_id, is_active, created_at",
		strings.Join(setClauses, ", "), i,
	)

	var u UserResponse
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar usuario"})
		return
	}
	c.JSON(http.StatusOK, u)
}
