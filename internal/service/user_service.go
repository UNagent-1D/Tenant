package service

import (
	"context"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Create(ctx context.Context, req CreateUserRequest, callerRole string, callerTenantID *uuid.UUID) (*domain.User, error)
	List(ctx context.Context, tenantID *uuid.UUID, page, limit int) ([]*domain.User, int, error)
	Update(ctx context.Context, uid uuid.UUID, req UpdateUserRequest, callerID uuid.UUID, callerRole string, callerTenantID *uuid.UUID) (*domain.User, error)
}

type CreateUserRequest struct {
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=8"`
	Role     domain.UserRole `json:"role" binding:"required"`
	TenantID *uuid.UUID      `json:"tenant_id"`
}

type UpdateUserRequest struct {
	Role     *domain.UserRole `json:"role"`
	IsActive *bool            `json:"is_active"`
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) Create(ctx context.Context, req CreateUserRequest, callerRole string, callerTenantID *uuid.UUID) (*domain.User, error) {
	if !isValidRole(req.Role) {
		return nil, fmt.Errorf("role must be one of: app_admin, tenant_admin, tenant_operator: %w", apperrors.ErrValidation)
	}

	// tenant_admin can only create tenant_operator in their own tenant
	if callerRole == "tenant_admin" {
		if req.Role != domain.RoleTenantOperator {
			return nil, fmt.Errorf("tenant_admin cannot create %s users: %w", req.Role, apperrors.ErrForbidden)
		}
		if callerTenantID == nil || req.TenantID == nil || *req.TenantID != *callerTenantID {
			return nil, fmt.Errorf("tenant_admin cannot create users for a different tenant: %w", apperrors.ErrForbidden)
		}
	}

	// app_admin must not be scoped to a tenant
	if req.Role == domain.RoleAppAdmin && req.TenantID != nil {
		return nil, fmt.Errorf("app_admin users must not have a tenant_id: %w", apperrors.ErrValidation)
	}
	// tenant_admin and tenant_operator must have a tenant_id
	if req.Role != domain.RoleAppAdmin && req.TenantID == nil {
		return nil, fmt.Errorf("tenant_id is required for role %s: %w", req.Role, apperrors.ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		TenantID:     req.TenantID,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *userService) List(ctx context.Context, tenantID *uuid.UUID, page, limit int) ([]*domain.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, tenantID, page, limit)
}

func (s *userService) Update(ctx context.Context, uid uuid.UUID, req UpdateUserRequest, callerID uuid.UUID, callerRole string, callerTenantID *uuid.UUID) (*domain.User, error) {
	if uid == callerID && req.IsActive != nil && !*req.IsActive {
		return nil, fmt.Errorf("cannot deactivate your own account: %w", apperrors.ErrForbidden)
	}

	u, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	// tenant_admin can only manage users in their own tenant
	if callerRole == "tenant_admin" {
		if u.TenantID == nil || callerTenantID == nil || *u.TenantID != *callerTenantID {
			return nil, fmt.Errorf("access denied: tenant mismatch: %w", apperrors.ErrForbidden)
		}
		// tenant_admin cannot promote to app_admin
		if req.Role != nil && *req.Role == domain.RoleAppAdmin {
			return nil, fmt.Errorf("tenant_admin cannot create or modify app_admin users: %w", apperrors.ErrForbidden)
		}
	}

	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func isValidRole(r domain.UserRole) bool {
	switch r {
	case domain.RoleAppAdmin, domain.RoleTenantAdmin, domain.RoleTenantOperator:
		return true
	}
	return false
}
