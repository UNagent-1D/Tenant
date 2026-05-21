package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
func internalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":      "internal server error",
		"request_id": c.GetString("request_id"),
	})
}

func scanUser(ctx context.Context, email string, u *User) error {
	const q = `SELECT id, email, password_hash, role, tenant_id, is_active, created_at
	            FROM users WHERE email = $1`
	scan := func() error {
		return db.Pool.QueryRow(ctx, q, email).Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt,
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
			errStr := err.Error()
			switch {
			case strings.Contains(errStr, "Email"):
				c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			case strings.Contains(errStr, "Password"):
				c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
			}
			return
		}

		var u User
		err := scanUser(c.Request.Context(), req.Email, &u)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			internalError(c)
			return
		}

		if !u.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "account is deactivated"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		expiresAt := time.Now().Add(expiry)
		claims := &Claims{
			UserID:   u.ID,
			Email:    u.Email,
			TenantID: u.TenantID,
			Role:     u.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expiresAt),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
		if err != nil {
			internalError(c)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      tokenString,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
			"user": gin.H{
				"id":         u.ID,
				"email":      u.Email,
				"role":       u.Role,
				"tenant_id":  u.TenantID,
				"is_active":  u.IsActive,
				"created_at": u.CreatedAt.UTC().Format(time.RFC3339),
			},
		})
	}
}

func GetMe(c *gin.Context) {
	claims := c.MustGet("user_claims").(*Claims)

	var u UserResponse
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT id, email, role, tenant_id, is_active, created_at FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, u)
}

func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		for _, msg := range strings.Split(err.Error(), "\n") {
			if strings.Contains(msg, "CurrentPassword") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "current_password is required"})
				return
			}
			if strings.Contains(msg, "NewPassword") && strings.Contains(msg, "required") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "new_password is required"})
				return
			}
			if strings.Contains(msg, "NewPassword") && strings.Contains(msg, "min") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "new_password must be at least 8 characters"})
				return
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims := c.MustGet("user_claims").(*Claims)

	var hash string
	if err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT password_hash FROM users WHERE id = $1`, claims.UserID,
	).Scan(&hash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization token"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		internalError(c)
		return
	}

	if _, err := db.Pool.Exec(c.Request.Context(),
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		string(newHash), claims.UserID,
	); err != nil {
		internalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

func GetUsers(c *gin.Context) {
	caller := c.MustGet("user_claims").(*Claims)

	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	offset := (page - 1) * limit

	var (
		total    int
		countQ   string
		listQ    string
		listArgs []any
	)

	if caller.Role == "tenant_admin" {
		countQ = `SELECT COUNT(*) FROM users WHERE tenant_id = $1`
		listQ = `SELECT id, email, role, tenant_id, is_active, created_at FROM users
		          WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		db.Pool.QueryRow(c.Request.Context(), countQ, caller.TenantID).Scan(&total)
		listArgs = []any{caller.TenantID, limit, offset}
	} else {
		countQ = `SELECT COUNT(*) FROM users`
		listQ = `SELECT id, email, role, tenant_id, is_active, created_at FROM users
		          ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		db.Pool.QueryRow(c.Request.Context(), countQ).Scan(&total)
		listArgs = []any{limit, offset}
	}

	rows, err := db.Pool.Query(c.Request.Context(), listQ, listArgs...)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	users := []UserResponse{}
	for rows.Next() {
		var u UserResponse
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt); err != nil {
			internalError(c)
			return
		}
		users = append(users, u)
	}
	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func GetUser(c *gin.Context) {
	caller := c.MustGet("user_claims").(*Claims)
	userID := c.Param("uid")

	var u UserResponse
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT id, email, role, tenant_id, is_active, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
			return
		}
		internalError(c)
		return
	}

	if caller.Role == "tenant_admin" {
		if u.TenantID == nil || caller.TenantID == nil || *u.TenantID != *caller.TenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied to this user"})
			return
		}
	}

	c.JSON(http.StatusOK, u)
}

func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "Email"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		case strings.Contains(errStr, "Password") && strings.Contains(errStr, "min"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		case strings.Contains(errStr, "Password"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		case strings.Contains(errStr, "Role"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of: app_admin, tenant_admin, tenant_operator"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	caller := c.MustGet("user_claims").(*Claims)

	switch caller.Role {
	case "app_admin":
		if (req.Role == "tenant_admin" || req.Role == "tenant_operator") && req.TenantID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required for tenant_admin and tenant_operator roles"})
			return
		}
		if req.Role == "app_admin" {
			req.TenantID = nil
		}
	case "tenant_admin":
		if req.Role == "app_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant_admin cannot create app_admin users"})
			return
		}
		if req.Role != "tenant_operator" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant_admin can only create tenant_operator users"})
			return
		}
		req.TenantID = caller.TenantID
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		internalError(c)
		return
	}

	// Empty-string names are normalised to NULL so the column stays clean
	// for rows where the caller didn't supply a name.
	var firstName, lastName *string
	if req.FirstName != "" {
		firstName = &req.FirstName
	}
	if req.LastName != "" {
		lastName = &req.LastName
	}

	var u UserResponse
	err = db.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO users (email, password_hash, role, tenant_id, first_name, last_name)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, email, role, tenant_id, is_active, created_at, first_name, last_name`,
		req.Email, string(hash), req.Role, req.TenantID, firstName, lastName,
	).Scan(&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt, &u.FirstName, &u.LastName)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				c.JSON(http.StatusConflict, gin.H{"error": "a user with this email already exists"})
				return
			case "23503":
				c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id references a tenant that does not exist"})
				return
			}
		}
		internalError(c)
		return
	}

	c.JSON(http.StatusCreated, u)
}

func UpdateUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of: app_admin, tenant_admin, tenant_operator"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied to this user"})
			return
		}
		if req.Role != nil && *req.Role == "app_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant_admin cannot assign app_admin role"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, u)
}

func DeleteUser(c *gin.Context) {
	caller := c.MustGet("user_claims").(*Claims)
	userID := c.Param("uid")

	if caller.UserID == userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete yourself"})
		return
	}

	result, err := db.Pool.Exec(c.Request.Context(),
		`DELETE FROM users WHERE id = $1 AND is_active = false`, userID,
	)
	if err != nil {
		internalError(c)
		return
	}
	if result.RowsAffected() == 0 {
		var exists bool
		db.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID,
		).Scan(&exists)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "user must be inactive before deletion"})
		return
	}

	c.Status(http.StatusNoContent)
}
