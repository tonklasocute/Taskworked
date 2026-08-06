package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Refresh(ctx context.Context, req RefreshRequest) (*AuthResponse, error)
	Logout(ctx context.Context, userID string) error

	// ListUsers is the org-wide directory — visible to any authenticated
	// user, same as seeing colleagues' names in a Slack workspace.
	ListUsers(ctx context.Context) ([]UserResponse, error)
	// GetUsersByIDs lets other modules (project, for enriching a member
	// list with names) resolve a batch of IDs without depending on the
	// full auth.Repository.
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]UserResponse, error)
	UpdateRole(ctx context.Context, actorRole Role, targetUserID uuid.UUID, req UpdateRoleRequest) (*UserResponse, error)
	UpdateDepartment(ctx context.Context, actorRole Role, targetUserID uuid.UUID, req UpdateDepartmentRequest) (*UserResponse, error)
}

type service struct {
	repo   Repository
	tokens *TokenService
}

func NewService(repo Repository, tokens *TokenService) Service {
	return &service{repo: repo, tokens: tokens}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if _, err := s.repo.FindByEmail(ctx, req.Email); err == nil {
		return nil, apperrors.Conflict("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Internal("failed to hash password")
	}

	user := &User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         RoleEmployee,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, apperrors.Internal("failed to create user")
	}

	return s.issueTokens(ctx, user)
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, apperrors.Unauthorized("invalid email or password")
		}
		return nil, apperrors.Internal("failed to look up user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.Unauthorized("invalid email or password")
	}

	return s.issueTokens(ctx, user)
}

func (s *service) Refresh(ctx context.Context, req RefreshRequest) (*AuthResponse, error) {
	userID, err := s.tokens.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, apperrors.Unauthorized("invalid or expired refresh token")
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Unauthorized("invalid or expired refresh token")
	}

	return s.issueTokens(ctx, user)
}

func (s *service) Logout(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return apperrors.BadRequest("invalid user id")
	}
	return s.tokens.RevokeRefreshToken(ctx, id)
}

func (s *service) ListUsers(ctx context.Context) ([]UserResponse, error) {
	users, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list users")
	}
	responses := make([]UserResponse, len(users))
	for i := range users {
		responses[i] = toUserResponse(&users[i])
	}
	return responses, nil
}

func (s *service) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]UserResponse, error) {
	users, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperrors.Internal("failed to look up users")
	}
	responses := make([]UserResponse, len(users))
	for i := range users {
		responses[i] = toUserResponse(&users[i])
	}
	return responses, nil
}

func (s *service) UpdateRole(ctx context.Context, actorRole Role, targetUserID uuid.UUID, req UpdateRoleRequest) (*UserResponse, error) {
	if !IsOrgAdmin(actorRole) {
		return nil, apperrors.Forbidden("only an admin can change roles")
	}
	if err := s.repo.UpdateRole(ctx, targetUserID, req.Role); err != nil {
		return nil, apperrors.Internal("failed to update role")
	}
	user, err := s.repo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, apperrors.NotFound("user not found")
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *service) UpdateDepartment(ctx context.Context, actorRole Role, targetUserID uuid.UUID, req UpdateDepartmentRequest) (*UserResponse, error) {
	if !IsOrgAdmin(actorRole) {
		return nil, apperrors.Forbidden("only an admin can change departments")
	}
	var departmentID *uuid.UUID
	if req.DepartmentID != nil && *req.DepartmentID != "" {
		parsed, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, apperrors.BadRequest("invalid department_id")
		}
		departmentID = &parsed
	}
	if err := s.repo.UpdateDepartment(ctx, targetUserID, departmentID); err != nil {
		return nil, apperrors.Internal("failed to update department")
	}
	user, err := s.repo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, apperrors.NotFound("user not found")
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func (s *service) issueTokens(ctx context.Context, user *User) (*AuthResponse, error) {
	access, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return nil, apperrors.Internal("failed to issue access token")
	}
	refresh, err := s.tokens.IssueRefreshToken(ctx, user)
	if err != nil {
		return nil, apperrors.Internal("failed to issue refresh token")
	}

	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         toUserResponse(user),
	}, nil
}
