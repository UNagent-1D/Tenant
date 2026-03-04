package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// e164Pattern matches E.164 phone numbers e.g. +573001234567
var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

type EndUserService interface {
	LookupByPhone(ctx context.Context, slug, cellphone string) (*domain.EndUser, error)
	LookupByNationalID(ctx context.Context, slug, nationalID string) (*domain.EndUser, error)
	Register(ctx context.Context, slug string, req RegisterEndUserRequest) (*domain.EndUser, error)
}

type RegisterEndUserRequest struct {
	FullName    string  `json:"full_name" binding:"required"`
	NationalID  string  `json:"national_id" binding:"required"`
	Cellphone   string  `json:"cellphone" binding:"required"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"`
	ExternalRef *string `json:"external_ref"`
}

type endUserService struct {
	repo repository.EndUserRepository
	pool *pgxpool.Pool
}

func NewEndUserService(repo repository.EndUserRepository, pool *pgxpool.Pool) EndUserService {
	return &endUserService{repo: repo, pool: pool}
}

func (s *endUserService) LookupByPhone(ctx context.Context, slug, cellphone string) (*domain.EndUser, error) {
	if !e164Pattern.MatchString(cellphone) {
		return nil, fmt.Errorf("cellphone number format is invalid — use E.164 format e.g. +573001234567: %w", apperrors.ErrValidation)
	}
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.GetByPhone(ctx, conn, cellphone) // nil means not found (anti-enumeration)
}

func (s *endUserService) LookupByNationalID(ctx context.Context, slug, nationalID string) (*domain.EndUser, error) {
	if nationalID == "" {
		return nil, fmt.Errorf("national_id cannot be empty: %w", apperrors.ErrValidation)
	}
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.GetByNationalID(ctx, conn, nationalID)
}

func (s *endUserService) Register(ctx context.Context, slug string, req RegisterEndUserRequest) (*domain.EndUser, error) {
	if !e164Pattern.MatchString(req.Cellphone) {
		return nil, fmt.Errorf("cellphone number format is invalid — use E.164 format e.g. +573001234567: %w", apperrors.ErrValidation)
	}
	if req.FullName == "" || req.NationalID == "" || req.Cellphone == "" {
		return nil, fmt.Errorf("full_name, national_id, and cellphone are required: %w", apperrors.ErrValidation)
	}

	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	eu := &domain.EndUser{
		ID:          uuid.New(),
		FullName:    req.FullName,
		NationalID:  &req.NationalID,
		Cellphone:   &req.Cellphone,
		Email:       req.Email,
		DateOfBirth: req.DateOfBirth,
		ExternalRef: req.ExternalRef,
		IsActive:    true,
	}
	if err := s.repo.Create(ctx, conn, eu); err != nil {
		return nil, err
	}
	return eu, nil
}
