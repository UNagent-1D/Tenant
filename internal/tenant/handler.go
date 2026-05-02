package tenant

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/internal/auth"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ── Tenants ───────────────────────────────────────────────────────────────────

func GetTenants(c *gin.Context) {
	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		 FROM tenants ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar tenants"})
		return
	}
	defer rows.Close()

	tenants := []Tenant{}
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
			&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer tenants"})
			return
		}
		tenants = append(tenants, t)
	}
	c.JSON(http.StatusOK, tenants)
}

func GetTenant(c *gin.Context) {
	var t Tenant
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		 FROM tenants WHERE id = $1`,
		c.Param("id"),
	).Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// CreateTenant inserts the tenant then provisions its schema.
// cfg is needed by provisionTenantSchema to run golang-migrate.
func CreateTenant(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTenantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var t Tenant
		err := db.Pool.QueryRow(c.Request.Context(),
			`INSERT INTO tenants (slug, name, plan, branding_logo_url, branding_primary_color)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at`,
			req.Slug, req.Name, req.Plan, req.BrandingLogoURL, req.BrandingPrimaryColor,
		).Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
			&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": "El slug ya existe"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear tenant"})
			return
		}

		if err := provisionTenantSchema(c.Request.Context(), t.Slug, cfg); err != nil {
			// Best-effort rollback of the tenant row
			db.Pool.Exec(c.Request.Context(), "DELETE FROM tenants WHERE id = $1", t.ID)
			if strings.Contains(err.Error(), "already exists") {
				c.JSON(http.StatusConflict, gin.H{"error": "El schema del tenant ya existe"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al provisionar schema del tenant"})
			return
		}

		c.JSON(http.StatusCreated, t)
	}
}

func UpdateTenant(c *gin.Context) {
	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	caller := c.MustGet("user_claims").(*auth.Claims)
	if caller.Role == "tenant_admin" {
		if req.Plan != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "only app_admin can change plan"})
			return
		}
		if req.Status != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "only app_admin can change status"})
			return
		}
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", i))
		args = append(args, *req.Name)
		i++
	}
	if req.Plan != nil {
		setClauses = append(setClauses, fmt.Sprintf("plan = $%d", i))
		args = append(args, *req.Plan)
		i++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", i))
		args = append(args, *req.Status)
		i++
	}
	if req.BrandingLogoURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("branding_logo_url = $%d", i))
		args = append(args, *req.BrandingLogoURL)
		i++
	}
	if req.BrandingPrimaryColor != nil {
		setClauses = append(setClauses, fmt.Sprintf("branding_primary_color = $%d", i))
		args = append(args, *req.BrandingPrimaryColor)
		i++
	}
	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sin campos para actualizar"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, c.Param("id"))

	query := fmt.Sprintf(
		`UPDATE tenants SET %s WHERE id = $%d
		 RETURNING id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at`,
		strings.Join(setClauses, ", "), i,
	)

	var t Tenant
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar tenant"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// ── Channels ──────────────────────────────────────────────────────────────────

func GetChannels(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active
		             FROM %s.channels`, schema),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar canales"})
		return
	}
	defer rows.Close()

	channels := []Channel{}
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey,
			&ch.WebhookSecretRef, &ch.IsActive); err != nil {
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

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ch Channel
	err := db.Pool.QueryRow(c.Request.Context(),
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

	var req UpdateChannelRequest
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

	var ch Channel
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
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

// ── Agent Profiles ────────────────────────────────────────────────────────────

func GetProfiles(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, name, description, scheduling_flow_rules, escalation_rules,
		                    allowed_specialties, allowed_locations, agent_config_id
		             FROM %s.agent_profiles`, schema),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar perfiles"})
		return
	}
	defer rows.Close()

	profiles := []AgentProfile{}
	for rows.Next() {
		var p AgentProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules,
			&p.EscalationRules, &p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer perfiles"})
			return
		}
		profiles = append(profiles, p)
	}
	c.JSON(http.StatusOK, profiles)
}

func CreateProfile(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var p AgentProfile
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.agent_profiles
		             (name, description, scheduling_flow_rules, escalation_rules, allowed_specialties, allowed_locations)
		             VALUES ($1, $2, $3, $4, $5, $6)
		             RETURNING id, name, description, scheduling_flow_rules, escalation_rules,
		                       allowed_specialties, allowed_locations, agent_config_id`, schema),
		req.Name, req.Description, req.SchedulingFlowRules, req.EscalationRules,
		req.AllowedSpecialties, req.AllowedLocations,
	).Scan(&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules,
		&p.EscalationRules, &p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear perfil"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func UpdateProfile(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", i))
		args = append(args, *req.Name)
		i++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", i))
		args = append(args, *req.Description)
		i++
	}
	if req.SchedulingFlowRules != nil {
		setClauses = append(setClauses, fmt.Sprintf("scheduling_flow_rules = $%d", i))
		args = append(args, req.SchedulingFlowRules)
		i++
	}
	if req.EscalationRules != nil {
		setClauses = append(setClauses, fmt.Sprintf("escalation_rules = $%d", i))
		args = append(args, req.EscalationRules)
		i++
	}
	if req.AllowedSpecialties != nil {
		setClauses = append(setClauses, fmt.Sprintf("allowed_specialties = $%d", i))
		args = append(args, req.AllowedSpecialties)
		i++
	}
	if req.AllowedLocations != nil {
		setClauses = append(setClauses, fmt.Sprintf("allowed_locations = $%d", i))
		args = append(args, req.AllowedLocations)
		i++
	}
	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sin campos para actualizar"})
		return
	}
	args = append(args, c.Param("pid"))

	query := fmt.Sprintf(
		`UPDATE %s.agent_profiles SET %s WHERE id = $%d
		 RETURNING id, name, description, scheduling_flow_rules, escalation_rules,
		           allowed_specialties, allowed_locations, agent_config_id`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var p AgentProfile
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules,
		&p.EscalationRules, &p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Perfil no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar perfil"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// ── Agent Configs (ACR) ───────────────────────────────────────────────────────

func GetConfigs(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                    tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at
		             FROM %s.agent_configs WHERE agent_profile_id = $1 ORDER BY version DESC`, schema),
		c.Param("pid"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar configs"})
		return
	}
	defer rows.Close()

	configs := []AgentConfig{}
	for rows.Next() {
		var cfg AgentConfig
		if err := rows.Scan(&cfg.ID, &cfg.AgentProfileID, &cfg.Version, &cfg.Status,
			&cfg.ConversationPolicy, &cfg.EscalationRules, &cfg.ToolPermissions, &cfg.LLMParams,
			&cfg.ChannelFormatRules, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.ActivatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer configs"})
			return
		}
		configs = append(configs, cfg)
	}
	c.JSON(http.StatusOK, configs)
}

func GetActiveConfig(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var cfg AgentConfig
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                    tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at
		             FROM %s.agent_configs WHERE agent_profile_id = $1 AND status = 'active'`, schema),
		c.Param("pid"),
	).Scan(&cfg.ID, &cfg.AgentProfileID, &cfg.Version, &cfg.Status,
		&cfg.ConversationPolicy, &cfg.EscalationRules, &cfg.ToolPermissions, &cfg.LLMParams,
		&cfg.ChannelFormatRules, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.ActivatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "No hay config activa para este perfil"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func CreateConfig(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	caller := c.MustGet("user_claims").(*auth.Claims)

	var req CreateAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateLLMParams(req.LLMParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateEscalationRules(req.EscalationRules); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var nextVersion int
	db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT COALESCE(MAX(version), 0) + 1 FROM %s.agent_configs WHERE agent_profile_id = $1`, schema),
		c.Param("pid"),
	).Scan(&nextVersion)

	var agentCfg AgentConfig
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.agent_configs
		             (agent_profile_id, version, conversation_policy, escalation_rules, tool_permissions, llm_params, channel_format_rules, created_by)
		             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		             RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                       tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`, schema),
		c.Param("pid"), nextVersion, req.ConversationPolicy, req.EscalationRules,
		req.ToolPermissions, req.LLMParams, req.ChannelFormatRules, caller.UserID,
	).Scan(&agentCfg.ID, &agentCfg.AgentProfileID, &agentCfg.Version, &agentCfg.Status,
		&agentCfg.ConversationPolicy, &agentCfg.EscalationRules, &agentCfg.ToolPermissions, &agentCfg.LLMParams,
		&agentCfg.ChannelFormatRules, &agentCfg.CreatedBy, &agentCfg.CreatedAt, &agentCfg.ActivatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear config"})
		return
	}
	c.JSON(http.StatusCreated, agentCfg)
}

func UpdateConfig(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req UpdateAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.LLMParams != nil {
		if err := validateLLMParams(req.LLMParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.EscalationRules != nil {
		if err := validateEscalationRules(req.EscalationRules); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.ConversationPolicy != nil {
		setClauses = append(setClauses, fmt.Sprintf("conversation_policy = $%d", i))
		args = append(args, req.ConversationPolicy)
		i++
	}
	if req.EscalationRules != nil {
		setClauses = append(setClauses, fmt.Sprintf("escalation_rules = $%d", i))
		args = append(args, req.EscalationRules)
		i++
	}
	if req.ToolPermissions != nil {
		setClauses = append(setClauses, fmt.Sprintf("tool_permissions = $%d", i))
		args = append(args, req.ToolPermissions)
		i++
	}
	if req.LLMParams != nil {
		setClauses = append(setClauses, fmt.Sprintf("llm_params = $%d", i))
		args = append(args, req.LLMParams)
		i++
	}
	if req.ChannelFormatRules != nil {
		setClauses = append(setClauses, fmt.Sprintf("channel_format_rules = $%d", i))
		args = append(args, req.ChannelFormatRules)
		i++
	}
	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sin campos para actualizar"})
		return
	}
	args = append(args, c.Param("cid"))

	query := fmt.Sprintf(
		`UPDATE %s.agent_configs SET %s WHERE id = $%d AND status = 'draft'
		 RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		           tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var agentCfg AgentConfig
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&agentCfg.ID, &agentCfg.AgentProfileID, &agentCfg.Version, &agentCfg.Status,
		&agentCfg.ConversationPolicy, &agentCfg.EscalationRules, &agentCfg.ToolPermissions, &agentCfg.LLMParams,
		&agentCfg.ChannelFormatRules, &agentCfg.CreatedBy, &agentCfg.CreatedAt, &agentCfg.ActivatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Config no encontrada o no está en estado draft"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar config"})
		return
	}
	c.JSON(http.StatusOK, agentCfg)
}

func ActivateConfig(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	configID := c.Param("cid")
	profileID := c.Param("pid")

	tx, err := db.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	tx.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_configs SET status = 'archived' WHERE agent_profile_id = $1 AND status = 'active'`, schema),
		profileID,
	)

	var agentCfg AgentConfig
	err = tx.QueryRow(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_configs SET status = 'active', activated_at = NOW()
		             WHERE id = $1 AND status = 'draft'
		             RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                       tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`, schema),
		configID,
	).Scan(&agentCfg.ID, &agentCfg.AgentProfileID, &agentCfg.Version, &agentCfg.Status,
		&agentCfg.ConversationPolicy, &agentCfg.EscalationRules, &agentCfg.ToolPermissions, &agentCfg.LLMParams,
		&agentCfg.ChannelFormatRules, &agentCfg.CreatedBy, &agentCfg.CreatedAt, &agentCfg.ActivatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Config no encontrada o no está en estado draft"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al activar config"})
		return
	}

	tx.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_profiles SET agent_config_id = $1 WHERE id = $2`, schema),
		configID, profileID,
	)

	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar transacción"})
		return
	}
	c.JSON(http.StatusOK, agentCfg)
}

func validateLLMParams(raw json.RawMessage) error {
	var p struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		MaxTokens   int     `json:"max_tokens"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("llm_params inválido")
	}
	allowed := map[string]bool{"gpt-4o": true, "gpt-4o-mini": true}
	if !allowed[p.Model] {
		return fmt.Errorf("modelo no permitido: %s", p.Model)
	}
	if p.Temperature < 0 || p.Temperature > 2 {
		return fmt.Errorf("temperature debe estar entre 0.0 y 2.0")
	}
	if p.MaxTokens < 1 || p.MaxTokens > 4096 {
		return fmt.Errorf("max_tokens debe estar entre 1 y 4096")
	}
	return nil
}

func validateEscalationRules(raw json.RawMessage) error {
	var rules []any
	if err := json.Unmarshal(raw, &rules); err != nil {
		return fmt.Errorf("escalation_rules debe ser un array JSON")
	}
	if len(rules) == 0 {
		return fmt.Errorf("debe haber al menos una regla de escalación")
	}
	return nil
}

// ── Data Sources ──────────────────────────────────────────────────────────────

func GetDataSources(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, name, source_type, base_url, credential_ref, route_configs, is_active
		             FROM %s.data_sources`, schema),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar data sources"})
		return
	}
	defer rows.Close()

	sources := []DataSource{}
	for rows.Next() {
		var ds DataSource
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL,
			&ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer data sources"})
			return
		}
		sources = append(sources, ds)
	}
	c.JSON(http.StatusOK, sources)
}

func CreateDataSource(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ds DataSource
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.data_sources (name, source_type, base_url, credential_ref, route_configs)
		             VALUES ($1, $2, $3, $4, $5)
		             RETURNING id, name, source_type, base_url, credential_ref, route_configs, is_active`, schema),
		req.Name, req.SourceType, req.BaseURL, req.CredentialRef, req.RouteConfigs,
	).Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear data source"})
		return
	}
	c.JSON(http.StatusCreated, ds)
}

func UpdateDataSource(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.RouteConfigs != nil {
		setClauses = append(setClauses, fmt.Sprintf("route_configs = $%d", i))
		args = append(args, req.RouteConfigs)
		i++
	}
	if req.CredentialRef != nil {
		setClauses = append(setClauses, fmt.Sprintf("credential_ref = $%d", i))
		args = append(args, *req.CredentialRef)
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
	args = append(args, c.Param("did"))

	query := fmt.Sprintf(
		`UPDATE %s.data_sources SET %s WHERE id = $%d
		 RETURNING id, name, source_type, base_url, credential_ref, route_configs, is_active`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var ds DataSource
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data source no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar data source"})
		return
	}
	c.JSON(http.StatusOK, ds)
}

// ── Tool Registry ─────────────────────────────────────────────────────────────

func GetTools(c *gin.Context) {
	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT id, name, description, openai_function_def, is_active FROM tool_registry ORDER BY name`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar herramientas"})
		return
	}
	defer rows.Close()

	tools := []Tool{}
	for rows.Next() {
		var t Tool
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer herramientas"})
			return
		}
		tools = append(tools, t)
	}
	c.JSON(http.StatusOK, tools)
}

func CreateTool(c *gin.Context) {
	var req CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var t Tool
	err := db.Pool.QueryRow(c.Request.Context(),
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
	var req UpdateToolRequest
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

	var t Tool
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
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

// ── End Users ─────────────────────────────────────────────────────────────────

func CreateEndUser(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var req CreateEndUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var eu EndUser
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.end_users (tenant_id, full_name, national_id, cellphone, email, date_of_birth, external_ref)
		             VALUES ($1, $2, $3, $4, $5, $6, $7)
		             RETURNING id, tenant_id, full_name, national_id, cellphone, email, is_active, external_ref, created_at`, schema),
		tenantID, req.FullName, req.NationalID, req.Cellphone, req.Email, req.DateOfBirth, req.ExternalRef,
	).Scan(&eu.ID, &eu.TenantID, &eu.FullName, &eu.NationalID, &eu.Cellphone,
		&eu.Email, &eu.IsActive, &eu.ExternalRef, &eu.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "national_id o cellphone ya registrado para este tenant"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar usuario final"})
		return
	}
	c.JSON(http.StatusCreated, eu)
}

// LookupByPhone always returns 200 to prevent enumeration attacks.
func LookupByPhone(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var id string
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT id FROM %s.end_users WHERE tenant_id = $1 AND cellphone = $2 AND is_active = true`, schema),
		tenantID, c.Param("number"),
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusOK, LookupResponse{Exists: false})
		return
	}
	c.JSON(http.StatusOK, LookupResponse{Exists: true, UserID: &id})
}

// LookupByNationalID always returns 200 to prevent enumeration attacks.
func LookupByNationalID(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var id string
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT id FROM %s.end_users WHERE tenant_id = $1 AND national_id = $2 AND is_active = true`, schema),
		tenantID, c.Param("nid"),
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusOK, LookupResponse{Exists: false})
		return
	}
	c.JSON(http.StatusOK, LookupResponse{Exists: true, UserID: &id})
}
