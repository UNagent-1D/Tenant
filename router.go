package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/handlers"
	"github.com/UNagent-1D/Tenant/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type tenantRow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domain    *string   `json:"domain"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createTenantRequest struct {
	Name   string `json:"name" binding:"required"`
	Domain string `json:"domain"`
}

func createTenantHandler(c *gin.Context) {
	var req createTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido", "detail": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Domain = strings.TrimSpace(req.Domain)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre es obligatorio"})
		return
	}

	var domain any
	if req.Domain == "" {
		domain = nil
	} else {
		domain = req.Domain
	}

	var t tenantRow
	err := config.DB.QueryRow(
		`INSERT INTO tenants (name, domain) VALUES ($1, $2)
		 RETURNING id, name, domain, is_active, created_at, updated_at`,
		req.Name, domain,
	).Scan(&t.ID, &t.Name, &t.Domain, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un tenant con ese dominio"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

type createUserRequest struct {
	Email     string  `json:"email" binding:"required,email"`
	Password  string  `json:"password" binding:"required,min=6"`
	FirstName string  `json:"first_name" binding:"required"`
	LastName  string  `json:"last_name" binding:"required"`
	Role      string  `json:"role" binding:"required"`
	TenantID  *string `json:"tenant_id"`
}

type createUserResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

func createUserHandler(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido", "detail": err.Error()})
		return
	}

	switch req.Role {
	case "app_admin":
		req.TenantID = nil
	case "tenant_admin", "tenant_operator":
		if req.TenantID == nil || strings.TrimSpace(*req.TenantID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id es obligatorio para este rol"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rol inválido"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cifrar la contraseña"})
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRow(
		`INSERT INTO users (email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		strings.ToLower(strings.TrimSpace(req.Email)), string(hash), req.FirstName, req.LastName,
	).Scan(&userID)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un usuario con ese email"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID != nil {
		var exists bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)`, *req.TenantID).Scan(&exists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id no existe"})
			return
		}
	}

	if _, err := tx.Exec(
		`INSERT INTO user_tenants (user_id, tenant_id, role) VALUES ($1, $2, $3)`,
		userID, req.TenantID, req.Role,
	); err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "La combinación rol/tenant viola la política del sistema"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createUserResponse{UserID: userID, Message: "Usuario creado"})
}

func listTenantsHandler(c *gin.Context) {
	rows, err := config.DB.Query(`SELECT id, name, domain, is_active, created_at, updated_at FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []tenantRow{}
	for rows.Next() {
		var t tenantRow
		if err := rows.Scan(&t.ID, &t.Name, &t.Domain, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, t)
	}
	c.JSON(http.StatusOK, out)
}

// corsMiddleware answers preflight OPTIONS and attaches the CORS headers
// every response needs so the browser-hosted frontend can call us.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SetupRouter inicializa y configura todas las rutas HTTP de la aplicación
func SetupRouter() *gin.Engine {
	// 2. Levanta el enrutador de Gin
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		if err := config.DB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "db": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "tenant"})
	})

	// 3. Rutas Públicas (No requieren Token)
	authGroup := router.Group("/auth")
	{
		// Recibe { "email": "", "password": "" } y emite un JWT firmado verificando contra SQL.
		// Rate-limited por IP para mitigar fuerza bruta / credential stuffing contra bcrypt.
		authGroup.POST("/login", middlewares.LoginRateLimiter(), handlers.LoginHandler)
	}

	// 4. API protegida general (El AuthMiddleware obliga la presencia de JWT válido)
	apiGroup := router.Group("/api")
	apiGroup.Use(middlewares.AuthMiddleware())
	{
		// 4A. Rutas del App Admin (solo "app_admin", que no tiene Tenant ID asignado, los administra todos)
		adminGroup := apiGroup.Group("/admin")
		adminGroup.Use(middlewares.RoleMiddleware("app_admin"))
		{
			adminGroup.GET("/tenants", listTenantsHandler)
			adminGroup.POST("/tenants", createTenantHandler)
			adminGroup.POST("/users", createUserHandler)
		}

		// 4B. Rutas protegidas a nivel del tenant
		tenantGroup := apiGroup.Group("/tenant")
		{
			// Estadísticas: Requiere ser Tenant Admin
			tenantGroup.GET("/stats", middlewares.RoleMiddleware("tenant_admin"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Estadísticas del tenant principal"})
			})

			// Operadores: Los tenant admins u operators pueden consultar el board de operaciones
			tenantGroup.GET("/operations", middlewares.RoleMiddleware("tenant_admin", "tenant_operator"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Panel de Operaciones de este Tenant"})
			})
		}
	}

	return router
}
