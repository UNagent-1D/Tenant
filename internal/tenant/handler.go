package tenant

import (
	"log"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/internal/auth"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func internalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":      "internal server error",
		"request_id": c.GetString("request_id"),
	})
}

func parsePagination(c *gin.Context) (page, limit, offset int) {
	page = 1
	limit = 20
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
	offset = (page - 1) * limit
	return
}

// ── Tenants ───────────────────────────────────────────────────────────────────

func GetTenants(c *gin.Context) {
	page, limit, offset := parsePagination(c)

	var total int
	db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM tenants`).Scan(&total)

	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		 FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	tenants := []Tenant{}
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
			&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt); err != nil {
			internalError(c)
			return
		}
		tenants = append(tenants, t)
	}
	c.JSON(http.StatusOK, gin.H{
		"data":       tenants,
		"pagination": Pagination{Page: page, Limit: limit, Total: total},
	})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, t)
}

var reservedSlugs = map[string]bool{
	"admin": true, "api": true, "www": true, "health": true,
	"internal": true, "public": true, "static": true,
}

// CreateTenant inserts the tenant then provisions its schema.
func CreateTenant(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTenantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errStr := err.Error()
			switch {
			case strings.Contains(errStr, "Slug"):
				c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
			case strings.Contains(errStr, "Name"):
				c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			case strings.Contains(errStr, "Plan"):
				c.JSON(http.StatusBadRequest, gin.H{"error": "plan must be one of: free, starter, pro, enterprise"})
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return
		}
		for _, ch := range req.Slug {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
				c.JSON(http.StatusBadRequest, gin.H{"error": "slug can only contain lowercase letters, numbers, and hyphens"})
				return
			}
		}
		if reservedSlugs[req.Slug] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slug is reserved and cannot be used"})
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
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("a tenant with slug '%s' already exists", req.Slug)})
				return
			}
			internalError(c)
			return
		}

		if err := provisionTenantSchema(c.Request.Context(), t.Slug, cfg); err != nil {
			db.Pool.Exec(c.Request.Context(), "DELETE FROM tenants WHERE id = $1", t.ID)
			if strings.Contains(err.Error(), "already exists") {
				c.JSON(http.StatusConflict, gin.H{"error": "tenant schema already exists"})
				return
			}
			log.Printf("provisionTenantSchema failed slug=%q: %v", t.Slug, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to provision tenant schema", "request_id": c.GetString("request_id")})
			return
		}

		// Auto-provision a default agent profile + active ACR config so the
		// brand-new tenant has a functional bot from the moment of creation.
		// Failure here is non-fatal: the schema is already there; the admin
		// can re-add the profile via the Profiles UI later. Log and continue.
		if err := autoProvisionDefaultAgent(c.Request.Context(), t.Slug); err != nil {
			log.Printf("autoProvisionDefaultAgent failed slug=%q: %v — tenant created without default agent", t.Slug, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, t)
}

// ── Channels ──────────────────────────────────────────────────────────────────

func GetChannels(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, tenant_id, channel_type, channel_key, is_active, created_at, updated_at
		             FROM %s.channels ORDER BY created_at DESC`, schema),
	)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	channels := []ChannelListItem{}
	for rows.Next() {
		var ch ChannelListItem
		if err := rows.Scan(&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey,
			&ch.IsActive, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			internalError(c)
			return
		}
		channels = append(channels, ch)
	}
	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func CreateChannel(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "ChannelType"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "channel_type must be one of: whatsapp, web_widget"})
		case strings.Contains(errStr, "ChannelKey"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "channel_key is required"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	var ch Channel
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.channels (tenant_id, channel_type, channel_key, webhook_secret_ref)
		             VALUES ($1, $2, $3, $4)
		             RETURNING id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active, created_at, updated_at`, schema),
		tenantID, req.ChannelType, req.ChannelKey, req.WebhookSecretRef,
	).Scan(&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey, &ch.WebhookSecretRef, &ch.IsActive, &ch.CreatedAt, &ch.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("a channel with key '%s' already exists for this tenant", req.ChannelKey)})
			return
		}
		internalError(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, c.Param("cid"))

	query := fmt.Sprintf(
		`UPDATE %s.channels SET %s WHERE id = $%d
		 RETURNING id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active, created_at, updated_at`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var ch Channel
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&ch.ID, &ch.TenantID, &ch.ChannelType, &ch.ChannelKey, &ch.WebhookSecretRef, &ch.IsActive, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}
		internalError(c)
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
		                    allowed_specialties, allowed_locations, agent_config_id, created_at, updated_at
		             FROM %s.agent_profiles ORDER BY created_at DESC`, schema),
	)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	profiles := []AgentProfile{}
	for rows.Next() {
		var p AgentProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules,
			&p.EscalationRules, &p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			internalError(c)
			return
		}
		profiles = append(profiles, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": profiles})
}

func CreateProfile(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if len(req.AllowedSpecialties) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "allowed_specialties must have at least one entry"})
		return
	}
	if len(req.AllowedLocations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "allowed_locations must have at least one entry"})
		return
	}

	var p AgentProfile
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.agent_profiles
		             (name, description, scheduling_flow_rules, escalation_rules, allowed_specialties, allowed_locations)
		             VALUES ($1, $2, $3, $4, $5, $6)
		             RETURNING id, name, description, scheduling_flow_rules, escalation_rules,
		                       allowed_specialties, allowed_locations, agent_config_id, created_at, updated_at`, schema),
		req.Name, req.Description, req.SchedulingFlowRules, req.EscalationRules,
		req.AllowedSpecialties, req.AllowedLocations,
	).Scan(&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules,
		&p.EscalationRules, &p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID,
		&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		internalError(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, c.Param("pid"))

	query := fmt.Sprintf(
		`UPDATE %s.agent_profiles SET %s WHERE id = $%d
		 RETURNING id, name, description, scheduling_flow_rules, escalation_rules,
		           allowed_specialties, allowed_locations, agent_config_id, created_at, updated_at`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var p AgentProfile
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules,
		&p.EscalationRules, &p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, p)
}

// ── Agent Configs (ACR) ───────────────────────────────────────────────────────

func GetConfigs(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var exists bool
	if err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.agent_profiles WHERE id = $1)`, schema),
		c.Param("pid"),
	).Scan(&exists); err != nil {
		internalError(c)
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                    tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at
		             FROM %s.agent_configs WHERE agent_profile_id = $1 ORDER BY version DESC`, schema),
		c.Param("pid"),
	)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	configs := []AgentConfig{}
	for rows.Next() {
		var cfg AgentConfig
		if err := rows.Scan(&cfg.ID, &cfg.AgentProfileID, &cfg.Version, &cfg.Status,
			&cfg.ConversationPolicy, &cfg.EscalationRules, &cfg.ToolPermissions, &cfg.LLMParams,
			&cfg.ChannelFormatRules, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.ActivatedAt); err != nil {
			internalError(c)
			return
		}
		configs = append(configs, cfg)
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "no active config found for this profile"})
			return
		}
		internalError(c)
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
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "ConversationPolicy"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_policy is required"})
		case strings.Contains(errStr, "EscalationRules"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "escalation_rules is required"})
		case strings.Contains(errStr, "ToolPermissions"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "tool_permissions must have at least one entry"})
		case strings.Contains(errStr, "LLMParams"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "llm_params is required"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
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
	if len(req.ToolPermissions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool_permissions must have at least one entry"})
		return
	}
	if err := validateToolPermissions(c, req.ToolPermissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var profileExists bool
	if err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.agent_profiles WHERE id = $1)`, schema),
		c.Param("pid"),
	).Scan(&profileExists); err != nil {
		internalError(c)
		return
	}
	if !profileExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
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
		internalError(c)
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

	var configStatus string
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT status FROM %s.agent_configs WHERE id = $1`, schema),
		c.Param("cid"),
	).Scan(&configStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		internalError(c)
		return
	}
	if configStatus != "draft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only draft configs can be updated"})
		return
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	args = append(args, c.Param("cid"))

	query := fmt.Sprintf(
		`UPDATE %s.agent_configs SET %s WHERE id = $%d
		 RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		           tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var agentCfg AgentConfig
	if err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&agentCfg.ID, &agentCfg.AgentProfileID, &agentCfg.Version, &agentCfg.Status,
		&agentCfg.ConversationPolicy, &agentCfg.EscalationRules, &agentCfg.ToolPermissions, &agentCfg.LLMParams,
		&agentCfg.ChannelFormatRules, &agentCfg.CreatedBy, &agentCfg.CreatedAt, &agentCfg.ActivatedAt,
	); err != nil {
		internalError(c)
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
		internalError(c)
		return
	}
	defer tx.Rollback(c.Request.Context())

	var configStatus string
	err = tx.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT status FROM %s.agent_configs WHERE id = $1`, schema),
		configID,
	).Scan(&configStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		internalError(c)
		return
	}
	if configStatus != "draft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only a draft config can be activated"})
		return
	}

	tx.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_configs SET status = 'archived' WHERE agent_profile_id = $1 AND status = 'active'`, schema),
		profileID,
	)

	var agentCfg AgentConfig
	err = tx.QueryRow(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_configs SET status = 'active', activated_at = NOW()
		             WHERE id = $1
		             RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                       tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`, schema),
		configID,
	).Scan(&agentCfg.ID, &agentCfg.AgentProfileID, &agentCfg.Version, &agentCfg.Status,
		&agentCfg.ConversationPolicy, &agentCfg.EscalationRules, &agentCfg.ToolPermissions, &agentCfg.LLMParams,
		&agentCfg.ChannelFormatRules, &agentCfg.CreatedBy, &agentCfg.CreatedAt, &agentCfg.ActivatedAt)

	if err != nil {
		internalError(c)
		return
	}

	tx.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_profiles SET agent_config_id = $1 WHERE id = $2`, schema),
		configID, profileID,
	)

	if err = tx.Commit(c.Request.Context()); err != nil {
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, agentCfg)
}

func validateToolPermissions(c *gin.Context, raw json.RawMessage) error {
	var entries []struct {
		ToolName string `json:"tool_name"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("tool_permissions must be a JSON array")
	}
	for _, e := range entries {
		var isActive bool
		err := db.Pool.QueryRow(c.Request.Context(),
			`SELECT is_active FROM tool_registry WHERE name = $1`, e.ToolName,
		).Scan(&isActive)
		if err != nil {
			return fmt.Errorf("tool '%s' is not registered in the tool registry", e.ToolName)
		}
		if !isActive {
			return fmt.Errorf("tool '%s' is currently deactivated globally", e.ToolName)
		}
	}
	return nil
}

func validateLLMParams(raw json.RawMessage) error {
	var p struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		MaxTokens   int     `json:"max_tokens"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("llm_params is invalid JSON")
	}
	allowed := map[string]bool{"gpt-4o": true, "gpt-4o-mini": true}
	if !allowed[p.Model] {
		return fmt.Errorf("model '%s' is not in the allowed model list", p.Model)
	}
	if p.Temperature < 0 || p.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0")
	}
	if p.MaxTokens < 1 || p.MaxTokens > 4096 {
		return fmt.Errorf("max_tokens must be between 1 and 4096")
	}
	return nil
}

func validateEscalationRules(raw json.RawMessage) error {
	var rules struct {
		Triggers []any `json:"triggers"`
	}
	if err := json.Unmarshal(raw, &rules); err != nil {
		return fmt.Errorf("escalation_rules is invalid JSON")
	}
	if len(rules.Triggers) == 0 {
		return fmt.Errorf("escalation_rules must have at least one trigger")
	}
	return nil
}

// ── Data Sources ──────────────────────────────────────────────────────────────

func GetDataSources(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := db.Pool.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, name, source_type, base_url, credential_ref, route_configs, is_active, created_at, updated_at
		             FROM %s.data_sources ORDER BY created_at DESC`, schema),
	)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	sources := []DataSource{}
	for rows.Next() {
		var ds DataSource
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL,
			&ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			internalError(c)
			return
		}
		sources = append(sources, ds)
	}
	c.JSON(http.StatusOK, gin.H{"data": sources})
}

func CreateDataSource(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "Name"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		case strings.Contains(errStr, "SourceType"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_type must be one of: scheduling, patient_registry"})
		case strings.Contains(errStr, "BaseURL"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "base_url must be a valid URL"})
		case strings.Contains(errStr, "RouteConfigs"):
			c.JSON(http.StatusBadRequest, gin.H{"error": "route_configs must have at least one operation"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	if u, err := url.ParseRequestURI(req.BaseURL); err != nil || u.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url must be a valid URL"})
		return
	}
	var routeMap map[string]any
	if err := json.Unmarshal(req.RouteConfigs, &routeMap); err != nil || len(routeMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route_configs must have at least one operation"})
		return
	}

	var ds DataSource
	err := db.Pool.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.data_sources (name, source_type, base_url, credential_ref, route_configs)
		             VALUES ($1, $2, $3, $4, $5)
		             RETURNING id, name, source_type, base_url, credential_ref, route_configs, is_active, created_at, updated_at`, schema),
		req.Name, req.SourceType, req.BaseURL, req.CredentialRef, req.RouteConfigs,
	).Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive, &ds.CreatedAt, &ds.UpdatedAt)

	if err != nil {
		internalError(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, c.Param("did"))

	query := fmt.Sprintf(
		`UPDATE %s.data_sources SET %s WHERE id = $%d
		 RETURNING id, name, source_type, base_url, credential_ref, route_configs, is_active, created_at, updated_at`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var ds DataSource
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "data source not found"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, ds)
}

// ── Tool Registry ─────────────────────────────────────────────────────────────

func GetTools(c *gin.Context) {
	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT id, name, description, openai_function_def, is_active, created_at, updated_at
		 FROM tool_registry ORDER BY name`,
	)
	if err != nil {
		internalError(c)
		return
	}
	defer rows.Close()

	tools := []Tool{}
	for rows.Next() {
		var t Tool
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			internalError(c)
			return
		}
		tools = append(tools, t)
	}
	c.JSON(http.StatusOK, gin.H{"data": tools})
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
		 RETURNING id, name, description, openai_function_def, is_active, created_at, updated_at`,
		req.Name, req.Description, req.OpenAIFunctionDef,
	).Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("a tool with name '%s' already exists", req.Name)})
			return
		}
		internalError(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, c.Param("tid"))

	query := fmt.Sprintf(
		"UPDATE tool_registry SET %s WHERE id = $%d RETURNING id, name, description, openai_function_def, is_active, created_at, updated_at",
		strings.Join(setClauses, ", "), i,
	)

	var t Tool
	err := db.Pool.QueryRow(c.Request.Context(), query, args...).Scan(
		&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, t)
}
