package handlers

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser maneja la creación y asignación de roles a usuarios
func CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload de solicitud inválido", "detalle": err.Error()})
		return
	}

	// Obtener los claims del usuario que hace la solicitud
	claimsVal, exists := c.Get("user_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	requesterClaims := claimsVal.(*models.Claims)
	requesterRole := requesterClaims.Role

	// ----- 1. Lógica de validación RBAC y Tenant -----
	if requesterRole == "app_admin" {
		// app_admin puede hacer lo que quiera, pero si crea tenant_admin o tenant_operator, DEBE proveer un TenantID
		if (req.Role == "tenant_admin" || req.Role == "tenant_operator") && req.TenantID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Debe especificar un tenant_id para este rol"})
			return
		}
		// Si crea un app_admin, forzamos que el tenant_id sea nulo
		if req.Role == "app_admin" {
			req.TenantID = nil
		}
	} else if requesterRole == "tenant_admin" {
		// tenant_admin SOLO puede crear tenant_operator
		if req.Role != "tenant_operator" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Un tenant_admin solo puede crear usuarios con rol tenant_operator"})
			return
		}
		// Forzamos el TenantID al del requester
		req.TenantID = requesterClaims.TenantID
	} else {
		// Nunca debería llegar aquí por el Middleware de Roles, pero por seguridad:
		c.JSON(http.StatusForbidden, gin.H{"error": "Rol no autorizado para crear usuarios"})
		return
	}

	// ----- 2. Hashing de contraseña -----
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al encriptar la contraseña"})
		return
	}
	passwordHash := string(hashedBytes)

	// ----- 3. Transacción SQL -----
	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar la transacción"})
		return
	}
	// Nos aseguramos de hacer rollback en caso de pánico o retorno temprano (si Commit ya se hizo, Rollback no hace nada)
	defer tx.Rollback()

	// 3a. Insertar el usuario
	var newUserID string
	insertUserQuery := `
		INSERT INTO users (email, password_hash, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`
	err = tx.QueryRow(insertUserQuery, req.Email, passwordHash, req.FirstName, req.LastName).Scan(&newUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el usuario", "detalle": err.Error()})
		return
	}

	// 3b. Insertar la asignación en user_tenants
	insertRoleQuery := `
		INSERT INTO user_tenants (user_id, tenant_id, role)
		VALUES ($1, $2, $3);
	`
	_, err = tx.Exec(insertRoleQuery, newUserID, req.TenantID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al asignar el rol al usuario", "detalle": err.Error()})
		return
	}

	// Confirmar la transacción
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al finalizar la transacción"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Usuario creado y rol asignado exitosamente",
		"user_id": newUserID,
	})
}

// DeleteUser maneja la eliminación de usuarios
func DeleteUser(c *gin.Context) {
	targetUserID := c.Param("id")

	// Obtener los claims del usuario que hace la solicitud
	claimsVal, exists := c.Get("user_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	requesterClaims := claimsVal.(*models.Claims)
	requesterRole := requesterClaims.Role

	var targetRole string
	var targetTenantID *string
	
	err := config.DB.QueryRow("SELECT role, tenant_id FROM user_tenants WHERE user_id = $1", targetUserID).Scan(&targetRole, &targetTenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	if requesterRole == "app_admin" {
		if targetRole == "app_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "app_admin no puede eliminar a otro app_admin"})
			return
		}
	} else if requesterRole == "tenant_admin" {
		if targetRole != "tenant_operator" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Un tenant_admin solo puede eliminar tenant_operators"})
			return
		}
		if targetTenantID == nil || requesterClaims.TenantID == nil || *targetTenantID != *requesterClaims.TenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "No puedes eliminar un usuario que no pertenece a tu tenant"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Rol no autorizado"})
		return
	}

	_, err = config.DB.Exec("DELETE FROM users WHERE id = $1", targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el usuario", "detalle": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuario eliminado exitosamente"})
}
