package service

import (
	"context"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentProfileService interface {
	Create(ctx context.Context, slug string, req CreateAgentProfileRequest) (*domain.AgentProfile, error)
	List(ctx context.Context, slug string) ([]*domain.AgentProfile, error)
	Update(ctx context.Context, slug string, profileID uuid.UUID, req UpdateAgentProfileRequest) (*domain.AgentProfile, error)
}

type CreateAgentProfileRequest struct {
	Name                string          `json:"name" binding:"required"`
	Description         *string         `json:"description"`
	SchedulingFlowRules []byte          `json:"scheduling_flow_rules"`
	EscalationRules     []byte          `json:"escalation_rules"`
	AllowedSpecialties  []string        `json:"allowed_specialties" binding:"required,min=1"`
	AllowedLocations    []string        `json:"allowed_locations"`
}

type UpdateAgentProfileRequest struct {
	Name                *string  `json:"name"`
	Description         *string  `json:"description"`
	SchedulingFlowRules []byte   `json:"scheduling_flow_rules"`
	EscalationRules     []byte   `json:"escalation_rules"`
	AllowedSpecialties  []string `json:"allowed_specialties"`
	AllowedLocations    []string `json:"allowed_locations"`
}

type agentProfileService struct {
	repo repository.AgentProfileRepository
	pool *pgxpool.Pool
}

func NewAgentProfileService(repo repository.AgentProfileRepository, pool *pgxpool.Pool) AgentProfileService {
	return &agentProfileService{repo: repo, pool: pool}
}

func (s *agentProfileService) Create(ctx context.Context, slug string, req CreateAgentProfileRequest) (*domain.AgentProfile, error) {
	if len(req.AllowedSpecialties) == 0 {
		return nil, fmt.Errorf("allowed_specialties cannot be empty: %w", apperrors.ErrValidation)
	}

	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	p := &domain.AgentProfile{
		ID:                  uuid.New(),
		Name:                req.Name,
		Description:         req.Description,
		SchedulingFlowRules: req.SchedulingFlowRules,
		EscalationRules:     req.EscalationRules,
		AllowedSpecialties:  req.AllowedSpecialties,
		AllowedLocations:    req.AllowedLocations,
	}
	if len(p.SchedulingFlowRules) == 0 {
		p.SchedulingFlowRules = []byte("{}")
	}
	if len(p.EscalationRules) == 0 {
		p.EscalationRules = []byte("{}")
	}

	if err := s.repo.Create(ctx, conn, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *agentProfileService) List(ctx context.Context, slug string) ([]*domain.AgentProfile, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.List(ctx, conn)
}

func (s *agentProfileService) Update(ctx context.Context, slug string, profileID uuid.UUID, req UpdateAgentProfileRequest) (*domain.AgentProfile, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	p, err := s.repo.GetByID(ctx, conn, profileID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = req.Description
	}
	if len(req.SchedulingFlowRules) > 0 {
		p.SchedulingFlowRules = req.SchedulingFlowRules
	}
	if len(req.EscalationRules) > 0 {
		p.EscalationRules = req.EscalationRules
	}
	if req.AllowedSpecialties != nil {
		p.AllowedSpecialties = req.AllowedSpecialties
	}
	if req.AllowedLocations != nil {
		p.AllowedLocations = req.AllowedLocations
	}

	if err := s.repo.Update(ctx, conn, p); err != nil {
		return nil, err
	}
	return p, nil
}
