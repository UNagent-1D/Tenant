package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// UpsertFromOTPRequest is the payload User-Auth sends after a successful
// OTP verification. Tenant is the source of truth for the session JWT — it
// looks the user up by email, creates one if missing (default tenant_operator
// with a random password), and mints the canonical token.
type UpsertFromOTPRequest struct {
	Email      string  `json:"email"       binding:"required,email"`
	Document   string  `json:"document"    binding:"required"`
	TenantID   *string `json:"tenant_id"`
	TenantSlug string  `json:"tenant_slug"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
}

// UpsertFromOTPHandler exchanges a verified OTP identity for a Tenant
// session JWT. Gated by InternalKeyMiddleware so only User-Auth (or other
// trusted internal callers) can hit it.
func UpsertFromOTPHandler(cfg *config.Config) gin.HandlerFunc {
	secret := []byte(cfg.JWTSecret)
	expiry := cfg.JWTExpiry
	return func(c *gin.Context) {
		var req UpsertFromOTPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
			return
		}

		var u User
		err := scanUser(c.Request.Context(), req.Email, &u)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			internalError(c)
			return
		}

		// First-time login via OTP: create a tenant_operator with a random
		// password. The user authenticates via OTP going forward; the
		// password is just a placeholder satisfying the NOT NULL constraint.
		if errors.Is(err, pgx.ErrNoRows) {
			randomPwd, perr := randomPassword(32)
			if perr != nil {
				internalError(c)
				return
			}
			hash, perr := bcrypt.GenerateFromPassword([]byte(randomPwd), bcrypt.DefaultCost)
			if perr != nil {
				internalError(c)
				return
			}

			role := "tenant_operator"
			if req.TenantID == nil {
				// No tenant scope → app_admin. Conservative default: refuse
				// so we don't accidentally elevate via OTP.
				c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required for new users"})
				return
			}

			const insertQ = `INSERT INTO users (email, password_hash, role, tenant_id, is_active)
			                 VALUES ($1, $2, $3, $4, true)
			                 RETURNING id, email, role, tenant_id, is_active, created_at`
			if err := db.Pool.QueryRow(c.Request.Context(), insertQ,
				req.Email, string(hash), role, *req.TenantID,
			).Scan(&u.ID, &u.Email, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt); err != nil {
				internalError(c)
				return
			}
		} else if !u.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "account is deactivated"})
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

func randomPassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
