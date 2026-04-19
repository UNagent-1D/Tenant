package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func GetTools(c *gin.Context) {
	rows, err := config.DB.Query(c.Request.Context(),
		`SELECT id, name, description, openai_function_def, is_active FROM tool_registry ORDER BY name`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar herramientas"})
		return
	}
	defer rows.Close()

	tools := []models.Tool{}
	for rows.Next() {
		var t models.Tool
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer herramientas"})
			return
		}
		tools = append(tools, t)
	}
	c.JSON(http.StatusOK, tools)
}

func CreateTool(c *gin.Context) {
	var req models.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var t models.Tool
	err := config.DB.QueryRow(c.Request.Context(),
		`INSERT INTO tool_registry (name, description, openai_function_def)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, description, openai_function_def, is_active`,
		req.Name, req.Description, req.OpenAIFunctionDef,
	).Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe una herramienta con ese nombre"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear herramienta"})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func UpdateTool(c *gin.Context) {
	var req models.UpdateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", i))
		args = append(args, *req.Description)
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
	args = append(args, c.Param("tid"))

	query := fmt.Sprintf(
		"UPDATE tool_registry SET %s WHERE id = $%d RETURNING id, name, description, openai_function_def, is_active",
		strings.Join(setClauses, ", "), i,
	)

	var t models.Tool
	err := config.DB.QueryRow(c.Request.Context(), query, args...).Scan(
		&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Herramienta no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar herramienta"})
		return
	}
	c.JSON(http.StatusOK, t)
}
