package project

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

type Service interface {
	Create(ctx context.Context, actorID uuid.UUID, req CreateRequest) (*Response, error)
	Get(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) (*Response, error)
	List(ctx context.Context, actorID uuid.UUID, filter ListFilter) (*ListResponse, error)
	Update(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateRequest) (*Response, error)
	Delete(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error
	AddMember(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req AddMemberRequest) error
	RemoveMember(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id, targetUserID uuid.UUID) error

	// CheckMembership lets other modules (e.g. task) authorize
	// project-scoped actions without depending on this module's
	// repository directly. isMember is true for org admins even without
	// an explicit membership row.
	CheckMembership(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (isMember bool, isManager bool, err error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, actorID uuid.UUID, req CreateRequest) (*Response, error) {
	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
	}

	p := &Project{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     actorID,
		Status:      StatusPlanning,
		DueDate:     dueDate,
		Color:       req.Color,
		Icon:        req.Icon,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, apperrors.Internal("failed to create project")
	}

	if err := s.repo.AddMember(ctx, &Member{ProjectID: p.ID, UserID: actorID, Role: MemberRoleOwner}); err != nil {
		return nil, apperrors.Internal("failed to add owner as member")
	}

	resp := toResponse(p)
	return &resp, nil
}

func (s *service) Get(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) (*Response, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}

	if !auth.IsOrgAdmin(actorRole) {
		if _, err := s.repo.FindMember(ctx, id, actorID); err != nil {
			return nil, apperrors.Forbidden("not a member of this project")
		}
	}

	resp := toResponse(p)
	return &resp, nil
}

func (s *service) List(ctx context.Context, actorID uuid.UUID, filter ListFilter) (*ListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	projects, total, err := s.repo.ListForMember(ctx, actorID, filter)
	if err != nil {
		return nil, apperrors.Internal("failed to list projects")
	}

	items := make([]Response, len(projects))
	for i := range projects {
		items[i] = toResponse(&projects[i])
	}

	return &ListResponse{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (s *service) Update(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateRequest) (*Response, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}

	allowed, err := s.canManage(ctx, actorID, actorRole, id)
	if err != nil {
		return nil, apperrors.Internal("failed to check permissions")
	}
	if !allowed {
		return nil, apperrors.Forbidden("only the project owner or an admin can update this project")
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.Color != nil {
		p.Color = *req.Color
	}
	if req.Icon != nil {
		p.Icon = *req.Icon
	}
	if req.DueDate != nil {
		dueDate, err := parseDueDate(req.DueDate)
		if err != nil {
			return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
		}
		p.DueDate = dueDate
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, apperrors.Internal("failed to update project")
	}

	resp := toResponse(p)
	return &resp, nil
}

func (s *service) Delete(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return notFoundOrInternal(err)
	}

	allowed, err := s.canManage(ctx, actorID, actorRole, id)
	if err != nil {
		return apperrors.Internal("failed to check permissions")
	}
	if !allowed {
		return apperrors.Forbidden("only the project owner or an admin can delete this project")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return apperrors.Internal("failed to delete project")
	}
	return nil
}

func (s *service) AddMember(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req AddMemberRequest) error {
	allowed, err := s.canManage(ctx, actorID, actorRole, id)
	if err != nil {
		return apperrors.Internal("failed to check permissions")
	}
	if !allowed {
		return apperrors.Forbidden("only the project owner or an admin can add members")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return apperrors.BadRequest("invalid user_id")
	}

	if err := s.repo.AddMember(ctx, &Member{ProjectID: id, UserID: userID, Role: MemberRoleMember}); err != nil {
		return apperrors.Conflict("user is already a member or does not exist")
	}
	return nil
}

func (s *service) RemoveMember(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id, targetUserID uuid.UUID) error {
	allowed, err := s.canManage(ctx, actorID, actorRole, id)
	if err != nil {
		return apperrors.Internal("failed to check permissions")
	}
	if !allowed {
		return apperrors.Forbidden("only the project owner or an admin can remove members")
	}

	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return notFoundOrInternal(err)
	}
	if p.OwnerID == targetUserID {
		return apperrors.BadRequest("cannot remove the project owner")
	}

	if err := s.repo.RemoveMember(ctx, id, targetUserID); err != nil {
		return apperrors.Internal("failed to remove member")
	}
	return nil
}

// canManage reports whether actor is an org admin/super admin, or the
// project's own owner-role member.
func (s *service) canManage(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (bool, error) {
	if auth.IsOrgAdmin(actorRole) {
		return true, nil
	}
	member, err := s.repo.FindMember(ctx, projectID, actorID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return member.Role == MemberRoleOwner, nil
}

func (s *service) CheckMembership(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (bool, bool, error) {
	if auth.IsOrgAdmin(actorRole) {
		return true, true, nil
	}
	member, err := s.repo.FindMember(ctx, projectID, actorID)
	if errors.Is(err, ErrNotFound) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, member.Role == MemberRoleOwner, nil
}

func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperrors.NotFound("project not found")
	}
	return apperrors.Internal("failed to look up project")
}

func parseDueDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
