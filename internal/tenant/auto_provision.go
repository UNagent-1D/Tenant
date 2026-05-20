package tenant

import (
	"context"
	"fmt"

	"github.com/UNagent-1D/Tenant/pkg/db"
)

// autoProvisionDefaultAgent inserts a "hospital base" agent profile + an
// active ACR config into the freshly-created tenant_{slug} schema. Called
// from CreateTenant after provisionTenantSchema runs the per-tenant
// migrations.
//
// The defaults mirror the hardcoded values in
// agent-runtime/src/agents/hospital.ts so brand-new tenants get a
// functional bot the moment they're created. Names/values can be tweaked
// later through the Profiles UI.
func autoProvisionDefaultAgent(ctx context.Context, slug string) error {
	schema := fmt.Sprintf("%q", "tenant_"+slug)

	// Insert agent_profile.
	var profileID string
	q1 := fmt.Sprintf(`
		INSERT INTO %s.agent_profiles
			(name, description, scheduling_flow_rules, escalation_rules,
			 allowed_specialties, allowed_locations)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6)
		RETURNING id
	`, schema)

	const schedulingFlow = `{
		"steps": ["list_doctors", "get_doctor_schedule", "book_appointment"],
		"defaults": {"channel": "telegram"}
	}`
	const escalation = `{
		"triggers": ["operator", "human", "humano"],
		"operator_ttl_seconds": 900,
		"ttl_fallback": "queue"
	}`

	specialties := []string{"general", "pediatria", "cardiologia"}
	locations := []string{"bogota", "medellin"}

	if err := db.Pool.QueryRow(ctx, q1,
		"Hospital base",
		"Perfil por defecto para clínicas. Edítalo desde el panel de Perfiles.",
		schedulingFlow,
		escalation,
		specialties,
		locations,
	).Scan(&profileID); err != nil {
		return fmt.Errorf("insert agent_profile: %w", err)
	}

	// Insert agent_config (active).
	var configID string
	q2 := fmt.Sprintf(`
		INSERT INTO %s.agent_configs
			(agent_profile_id, version, status, conversation_policy,
			 escalation_rules, tool_permissions, llm_params, channel_format_rules,
			 activated_at)
		VALUES ($1, 1, 'active', $2::jsonb, $3::jsonb, $4::jsonb, $5::jsonb, $6::jsonb, NOW())
		RETURNING id
	`, schema)

	const conversationPolicy = `{"language": "es-CO", "tone": "neutral_formal"}`
	const toolPermissions = `[
		{"tool_name": "list_doctors", "constraints": {}},
		{"tool_name": "get_doctor_schedule", "constraints": {}},
		{"tool_name": "book_appointment", "constraints": {}},
		{"tool_name": "cancel_appointment", "constraints": {}},
		{"tool_name": "get_patient_appointments", "constraints": {}},
		{"tool_name": "reschedule_appointment", "constraints": {}}
	]`
	const llmParams = `{
		"model": "openai/gpt-4o-mini",
		"temperature": 0.4,
		"max_tokens": 800,
		"system_prompt": "Eres un asistente clínico bilingüe que ayuda a pacientes a consultar, agendar, cancelar y reagendar citas."
	}`
	const channelFormat = `{"telegram": {"markdown": false}, "web": {"markdown": true}}`

	if err := db.Pool.QueryRow(ctx, q2,
		profileID,
		conversationPolicy,
		escalation,
		toolPermissions,
		llmParams,
		channelFormat,
	).Scan(&configID); err != nil {
		return fmt.Errorf("insert agent_config: %w", err)
	}

	// Link the profile back to its active config so /profiles/active can
	// resolve it via a single FK lookup.
	q3 := fmt.Sprintf(`UPDATE %s.agent_profiles SET agent_config_id = $1 WHERE id = $2`, schema)
	if _, err := db.Pool.Exec(ctx, q3, configID, profileID); err != nil {
		return fmt.Errorf("link profile→config: %w", err)
	}

	return nil
}
