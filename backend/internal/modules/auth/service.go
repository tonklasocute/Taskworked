package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/organization"
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
	//
	// Deliberately NOT organization-scoped: callers that need a tenant-safe
	// directory (e.g. team.Service.GetDirectory, the /team endpoint) must
	// use ListUsersByOrganization instead. This one stays because it's also
	// used by the notification digest cron, where returning cross-org users
	// is not a leak — each digest only ever surfaces that specific user's
	// own data (see P1.2 audit notes), never another user's.
	ListUsers(ctx context.Context) ([]UserResponse, error)
	// ListUsersByOrganization is the tenant-safe directory query — use this,
	// not ListUsers, for anything that displays the result to a user (as
	// opposed to iterating internally per-user like the digest cron does).
	ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]UserResponse, error)
	// GetUsersByIDs lets other modules (project, for enriching a member
	// list with names) resolve a batch of IDs without depending on the
	// full auth.Repository.
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]UserResponse, error)
	// GetOrganizationID is the trusted, server-side source of truth for
	// "what organization does this already-authenticated user belong to"
	// — this is what every tenant-isolation check in the codebase resolves
	// the actor's org from, rather than trusting any client-supplied value.
	// See docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md.
	GetOrganizationID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	// UpdateRole and UpdateDepartment take actorID (not just actorRole) so
	// they can confirm the target user belongs to the actor's own
	// organization before mutating anything — an org admin's reach stops
	// at their own organization's boundary, same as every other
	// authorization check in the codebase (see the P1.2 tenant isolation
	// audit). Without this, any admin could change any user's role or
	// department system-wide regardless of organization, which is exactly
	// the cross-tenant vertical-privilege-escalation path P1.2 exists to
	// close.
	UpdateRole(ctx context.Context, actorID uuid.UUID, actorRole Role, targetUserID uuid.UUID, req UpdateRoleRequest) (*UserResponse, error)
	UpdateDepartment(ctx context.Context, actorID uuid.UUID, actorRole Role, targetUserID uuid.UUID, req UpdateDepartmentRequest) (*UserResponse, error)
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

	count, countErr := s.repo.Count(ctx)
	// TODO(P1.5+): every registration is assigned to the single default
	// organization created by migration 0002 — there's no invitation flow
	// or organization-creation API yet for a registrant to join/create a
	// specific organization. This matches today's actual single-company
	// deployment reality and keeps existing registration working; revisit
	// once invitations (or self-serve org creation) exist.
	defaultOrgID := organization.DefaultOrganizationID
	user := &User{
		Name:           req.Name,
		Email:          req.Email,
		PasswordHash:   string(hash),
		Role:           bootstrapRole(count, countErr),
		OrganizationID: &defaultOrgID,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, apperrors.Internal("failed to create user")
	}

	return s.issueTokens(ctx, user)
}

// bootstrapRole decides the role a newly registering account gets. The
// very first account in the system has nobody around yet to grant it
// admin access, so it grants itself — the same self-bootstrap pattern
// most admin panels use (WordPress, Django, ...). Every account after
// that is a plain employee; promotion happens via UpdateRole.
//
// A failed Count query fails safe to the default (employee) rather than
// risking an unintended admin grant — better to require a manual promotion
// than to silently hand out admin because a query hiccupped.
//
// ponytail: no locking against the race where two registrations both see
// count==0 concurrently — acceptable for a one-time bootstrap event, not
// a hot path. Upgrade with a unique partial index if that ever matters.
func bootstrapRole(existingUserCount int64, countErr error) Role {
	if countErr == nil && existingUserCount == 0 {
		return RoleSuperAdmin
	}
	return RoleEmployee
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

func (s *service) ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]UserResponse, error) {
	users, err := s.repo.ListByOrganization(ctx, organizationID)
	if err != nil {
		return nil, apperrors.Internal("failed to list users")
	}
	responses := make([]UserResponse, len(users))
	for i := range users {
		responses[i] = toUserResponse(&users[i])
	}
	return responses, nil
}

// GetOrganizationID is the trusted lookup every tenant-isolation check in
// the codebase uses to resolve "what org does this authenticated user
// belong to" — see the Service interface doc comment. A user with no
// organization assigned (shouldn't happen after P1.1's backfill + this
// package's own registration/issueTokens fixes, but the column is still
// nullable at the DB level) fails closed rather than silently comparing
// against a zero UUID, which could otherwise let two such users match each
// other by coincidence.
func (s *service) GetOrganizationID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return uuid.Nil, apperrors.Internal("failed to look up user")
	}
	if user.OrganizationID == nil {
		return uuid.Nil, apperrors.Internal("user has no organization assigned")
	}
	return *user.OrganizationID, nil
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

func (s *service) UpdateRole(ctx context.Context, actorID uuid.UUID, actorRole Role, targetUserID uuid.UUID, req UpdateRoleRequest) (*UserResponse, error) {
	if !IsOrgAdmin(actorRole) {
		return nil, apperrors.Forbidden("only an admin can change roles")
	}
	if err := s.requireSameOrganization(ctx, actorID, targetUserID); err != nil {
		return nil, err
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

func (s *service) UpdateDepartment(ctx context.Context, actorID uuid.UUID, actorRole Role, targetUserID uuid.UUID, req UpdateDepartmentRequest) (*UserResponse, error) {
	if !IsOrgAdmin(actorRole) {
		return nil, apperrors.Forbidden("only an admin can change departments")
	}
	if err := s.requireSameOrganization(ctx, actorID, targetUserID); err != nil {
		return nil, err
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

// requireSameOrganization confirms targetUserID belongs to the same
// organization as actorID, returning NotFound (not Forbidden) on a
// mismatch — identical to a nonexistent target user, so an admin can't
// use this endpoint to probe whether a given user ID exists in another
// organization (see the P1.2 tenant isolation audit §25).
func (s *service) requireSameOrganization(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	actorOrgID, err := s.GetOrganizationID(ctx, actorID)
	if err != nil {
		return err
	}
	targetOrgID, err := s.GetOrganizationID(ctx, targetUserID)
	if err != nil {
		return apperrors.NotFound("user not found")
	}
	if actorOrgID != targetOrgID {
		return apperrors.NotFound("user not found")
	}
	return nil
}

func (s *service) issueTokens(ctx context.Context, user *User) (*AuthResponse, error) {
	// Fail closed rather than issue a token with an empty org claim: every
	// tenant-isolation check in the codebase treats "no organization" as
	// "matches nothing", so a claim-less token would just see empty lists
	// everywhere rather than a clear error — worse for debugging a real
	// data problem (a user that somehow has no organization) than refusing
	// to log them in at all.
	if user.OrganizationID == nil {
		return nil, apperrors.Internal("user has no organization assigned")
	}

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
