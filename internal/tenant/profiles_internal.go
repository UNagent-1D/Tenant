package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// ActiveAgentResponse is what agent-runtime (and chat-orch in Phase 3)
// consume to know which LLM model, system prompt, tools, and escalation
// rules apply to a tenant's bot. Mirrors agent-runtime's ACRConfig shape.
//
// DataSource* fields carry the active data source bound to this config,
// so agent-runtime's tenant-stub can return per-tenant route_configs
// without a second round-trip.
type ActiveAgentResponse struct {
	TenantID              string          `json:"tenant_id"`
	TenantSlug            string          `json:"tenant_slug"`
	ProfileID             string          `json:"profile_id"`
	ProfileName           string          `json:"profile_name"`
	ConfigID              string          `json:"config_id"`
	Version               int             `json:"version"`
	DataSourceID          string          `json:"data_source_id,omitempty"`
	DataSourceName        string          `json:"data_source_name,omitempty"`
	DataSourceBaseURL     string          `json:"data_source_base_url,omitempty"`
	DataSourceRouteConfig json.RawMessage `json:"data_source_route_configs,omitempty"`
	ConversationPolicy    json.RawMessage `json:"conversation_policy"`
	EscalationRules       json.RawMessage `json:"escalation_rules"`
	ToolPermissions       json.RawMessage `json:"tool_permissions"`
	LLMParams             json.RawMessage `json:"llm_params"`
	ChannelFormatRules    json.RawMessage `json:"channel_format_rules"`
	AllowedSpecialties    []string        `json:"allowed_specialties"`
	AllowedLocations      []string        `json:"allowed_locations"`
}

// GetActiveAgentByTenantID resolves the active profile+config for a tenant
// by tenant ID (or slug if the param looks like a slug). Gated by
// InternalKeyMiddleware in router.go.
func GetActiveAgentByTenantID(c *gin.Context) {
	tenantParam := c.Param("id")
	if tenantParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant id is required"})
		return
	}

	slug, err := resolveSlug(c.Request.Context(), tenantParam)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		internalError(c)
		return
	}

	schema := fmt.Sprintf("%q", "tenant_"+slug)
	q := fmt.Sprintf(`
		SELECT
			p.id, p.name,
			c.id, c.version,
			COALESCE(c.data_source_id::text, ''),
			COALESCE(ds.name, ''),
			COALESCE(ds.base_url, ''),
			COALESCE(ds.route_configs, '{}'::jsonb),
			c.conversation_policy,
			c.escalation_rules,
			c.tool_permissions,
			c.llm_params,
			COALESCE(c.channel_format_rules, '{}'::jsonb),
			COALESCE(p.allowed_specialties, '{}'::text[]),
			COALESCE(p.allowed_locations, '{}'::text[])
		FROM %s.agent_profiles p
		JOIN %s.agent_configs c
		  ON c.agent_profile_id = p.id AND c.status = 'active'
		LEFT JOIN %s.data_sources ds ON ds.id = c.data_source_id AND ds.is_active = true
		ORDER BY p.created_at ASC
		LIMIT 1
	`, schema, schema, schema)

	var resp ActiveAgentResponse
	resp.TenantSlug = slug
	if err := db.Pool.QueryRow(c.Request.Context(), q).Scan(
		&resp.ProfileID,
		&resp.ProfileName,
		&resp.ConfigID,
		&resp.Version,
		&resp.DataSourceID,
		&resp.DataSourceName,
		&resp.DataSourceBaseURL,
		&resp.DataSourceRouteConfig,
		&resp.ConversationPolicy,
		&resp.EscalationRules,
		&resp.ToolPermissions,
		&resp.LLMParams,
		&resp.ChannelFormatRules,
		&resp.AllowedSpecialties,
		&resp.AllowedLocations,
	); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active agent config for tenant"})
			return
		}
		internalError(c)
		return
	}

	// Look up tenant_id (in case the caller passed a slug).
	if err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT id FROM tenants WHERE slug = $1`, slug,
	).Scan(&resp.TenantID); err != nil {
		// Non-fatal: leave TenantID empty.
		resp.TenantID = ""
	}

	c.JSON(http.StatusOK, resp)
}

// resolveSlug accepts either a UUID (id) or a slug and returns the slug.
// We need the slug to compose the schema name (tenant_<slug>).
func resolveSlug(ctx context.Context, param string) (string, error) {
	// UUID heuristic: 36 chars with hyphens at the canonical positions.
	if len(param) == 36 && param[8] == '-' && param[13] == '-' && param[18] == '-' && param[23] == '-' {
		var slug string
		err := db.Pool.QueryRow(ctx, `SELECT slug FROM tenants WHERE id = $1`, param).Scan(&slug)
		return slug, err
	}
	// Otherwise assume it's already a slug; verify it exists.
	var slug string
	err := db.Pool.QueryRow(ctx, `SELECT slug FROM tenants WHERE slug = $1`, param).Scan(&slug)
	return slug, err
}
