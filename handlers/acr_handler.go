package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetConfigs(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := config.DB.Query(c.Request.Context(),
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

	configs := []models.AgentConfig{}
	for rows.Next() {
		var cfg models.AgentConfig
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

	var cfg models.AgentConfig
	err := config.DB.QueryRow(c.Request.Context(),
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
	caller := c.MustGet("user_claims").(*models.Claims)

	var req models.CreateAgentConfigRequest
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
	config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT COALESCE(MAX(version), 0) + 1 FROM %s.agent_configs WHERE agent_profile_id = $1`, schema),
		c.Param("pid"),
	).Scan(&nextVersion)

	var cfg models.AgentConfig
	err := config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.agent_configs
		             (agent_profile_id, version, conversation_policy, escalation_rules, tool_permissions, llm_params, channel_format_rules, created_by)
		             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		             RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                       tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`, schema),
		c.Param("pid"), nextVersion, req.ConversationPolicy, req.EscalationRules,
		req.ToolPermissions, req.LLMParams, req.ChannelFormatRules, caller.UserID,
	).Scan(&cfg.ID, &cfg.AgentProfileID, &cfg.Version, &cfg.Status,
		&cfg.ConversationPolicy, &cfg.EscalationRules, &cfg.ToolPermissions, &cfg.LLMParams,
		&cfg.ChannelFormatRules, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.ActivatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear config"})
		return
	}
	c.JSON(http.StatusCreated, cfg)
}

func UpdateConfig(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req models.UpdateAgentConfigRequest
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

	var cfg models.AgentConfig
	err := config.DB.QueryRow(c.Request.Context(), query, args...).Scan(
		&cfg.ID, &cfg.AgentProfileID, &cfg.Version, &cfg.Status,
		&cfg.ConversationPolicy, &cfg.EscalationRules, &cfg.ToolPermissions, &cfg.LLMParams,
		&cfg.ChannelFormatRules, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.ActivatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Config no encontrada o no está en estado draft"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar config"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func ActivateConfig(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	configID := c.Param("cid")
	profileID := c.Param("pid")

	tx, err := config.DB.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Archive current active config (if any)
	tx.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_configs SET status = 'archived' WHERE agent_profile_id = $1 AND status = 'active'`, schema),
		profileID,
	)

	// Activate target config
	var cfg models.AgentConfig
	err = tx.QueryRow(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_configs SET status = 'active', activated_at = NOW()
		             WHERE id = $1 AND status = 'draft'
		             RETURNING id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		                       tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at`, schema),
		configID,
	).Scan(&cfg.ID, &cfg.AgentProfileID, &cfg.Version, &cfg.Status,
		&cfg.ConversationPolicy, &cfg.EscalationRules, &cfg.ToolPermissions, &cfg.LLMParams,
		&cfg.ChannelFormatRules, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.ActivatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Config no encontrada o no está en estado draft"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al activar config"})
		return
	}

	// Update agent_profiles.agent_config_id to point to the active config
	tx.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.agent_profiles SET agent_config_id = $1 WHERE id = $2`, schema),
		configID, profileID,
	)

	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar transacción"})
		return
	}
	c.JSON(http.StatusOK, cfg)
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
