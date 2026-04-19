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

func GetProfiles(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := config.DB.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, name, description, scheduling_flow_rules, escalation_rules,
		                    allowed_specialties, allowed_locations, agent_config_id
		             FROM %s.agent_profiles`, schema),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar perfiles"})
		return
	}
	defer rows.Close()

	profiles := []models.AgentProfile{}
	for rows.Next() {
		var p models.AgentProfile
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

	var req models.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var p models.AgentProfile
	err := config.DB.QueryRow(c.Request.Context(),
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

	var req models.UpdateProfileRequest
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

	var p models.AgentProfile
	err := config.DB.QueryRow(c.Request.Context(), query, args...).Scan(
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
