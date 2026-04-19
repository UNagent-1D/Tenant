package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetChannels(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := config.DB.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active
		             FROM %s.channels`, schema),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar canales"})
		return
	}
	defer rows.Close()

	channels := []models.Channel{}
	for rows.Next() {
		var ch models.Channel
		if err := rows.Scan(&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey, &ch.WebhookSecretRef, &ch.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer canales"})
			return
		}
		channels = append(channels, ch)
	}
	c.JSON(http.StatusOK, channels)
}

func CreateChannel(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var req models.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ch models.Channel
	err := config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.channels (tenant_id, channel_type, channel_key, webhook_secret_ref)
		             VALUES ($1, $2, $3, $4)
		             RETURNING id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active`, schema),
		tenantID, req.ChannelType, req.ChannelKey, req.WebhookSecretRef,
	).Scan(&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey, &ch.WebhookSecretRef, &ch.IsActive)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear canal"})
		return
	}
	c.JSON(http.StatusCreated, ch)
}

func UpdateChannel(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req models.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.ChannelKey != nil {
		setClauses = append(setClauses, fmt.Sprintf("channel_key = $%d", i))
		args = append(args, *req.ChannelKey)
		i++
	}
	if req.WebhookSecretRef != nil {
		setClauses = append(setClauses, fmt.Sprintf("webhook_secret_ref = $%d", i))
		args = append(args, *req.WebhookSecretRef)
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
	args = append(args, c.Param("cid"))

	query := fmt.Sprintf(
		`UPDATE %s.channels SET %s WHERE id = $%d
		 RETURNING id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var ch models.Channel
	err := config.DB.QueryRow(c.Request.Context(), query, args...).Scan(
		&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey, &ch.WebhookSecretRef, &ch.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Canal no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar canal"})
		return
	}
	c.JSON(http.StatusOK, ch)
}
