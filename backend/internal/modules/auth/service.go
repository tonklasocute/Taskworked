package auth

import (
	"context"
	"errors"

	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Refresh(ctx context.Context, req RefreshRequest) (*AuthResponse, error)
	Logout(ctx context.Context, userID string) error
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
