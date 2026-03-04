package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
)

type ToolRegistryService interface {
	Create(ctx context.Context, req CreateToolRequest) (*domain.Tool, error)
	List(ctx context.Context) ([]*domain.Tool, error)
	Update(ctx context.Context, toolID uuid.UUID, req UpdateToolRequest) (*domain.Tool, error)
}

type CreateToolRequest struct {
	Name              string          `json:"name" binding:"required"`
	Description       *string         `json:"description"`
	OpenAIFunctionDef json.RawMessage `json:"openai_function_def" binding:"required"`
}

type UpdateToolRequest struct {
	Description       *string         `json:"description"`
	OpenAIFunctionDef json.RawMessage `json:"openai_function_def"`
	IsActive          *bool           `json:"is_active"`
}

type toolRegistryService struct {
	repo repository.ToolRepository
}

func NewToolRegistryService(repo repository.ToolRepository) ToolRegistryService {
	return &toolRegistryService{repo: repo}
}

func (s *toolRegistryService) Create(ctx context.Context, req CreateToolRequest) (*domain.Tool, error) {
	if err := validateOpenAIFunctionDef(req.OpenAIFunctionDef); err != nil {
		return nil, err
	}

	// Check for duplicate name
	existing, _ := s.repo.GetByName(ctx, req.Name)
	if existing != nil {
		return nil, fmt.Errorf("a tool named %q already exists: %w", req.Name, apperrors.ErrConflict)
	}

	t := &domain.Tool{
		ID:                uuid.New(),
		Name:              req.Name,
		Description:       req.Description,
		OpenAIFunctionDef: req.OpenAIFunctionDef,
		IsActive:          true,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *toolRegistryService) List(ctx context.Context) ([]*domain.Tool, error) {
	return s.repo.List(ctx)
}

func (s *toolRegistryService) Update(ctx context.Context, toolID uuid.UUID, req UpdateToolRequest) (*domain.Tool, error) {
	t, err := s.repo.GetByID(ctx, toolID)
	if err != nil {
		return nil, err
	}

	if req.Description != nil {
		t.Description = req.Description
	}
	if req.OpenAIFunctionDef != nil {
		if err := validateOpenAIFunctionDef(req.OpenAIFunctionDef); err != nil {
			return nil, err
		}
		t.OpenAIFunctionDef = req.OpenAIFunctionDef
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// validateOpenAIFunctionDef checks the function definition has at minimum a "parameters" field.
func validateOpenAIFunctionDef(raw json.RawMessage) error {
	var def map[string]json.RawMessage
	if err := json.Unmarshal(raw, &def); err != nil {
		return fmt.Errorf("openai_function_def must be valid JSON: %w", apperrors.ErrValidation)
	}
	if _, ok := def["parameters"]; !ok {
		return fmt.Errorf("openai_function_def is missing required field: parameters: %w", apperrors.ErrValidation)
	}
	return nil
}
