package team

import (
	"context"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// AuthService is the slice of auth.Service the directory needs — kept as
// a local interface (rather than importing auth.Service's full surface)
// so tests can fake just this.
type AuthService interface {
	ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]auth.UserResponse, error)
	GetOrganizationID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// WorkloadService is the slice of task.Service the directory needs.
type WorkloadService interface {
	GetWorkload(ctx context.Context, userID uuid.UUID) (int64, error)
}

// PresenceChecker abstracts the Redis-backed online check so the service
// is testable without a real Redis instance.
type PresenceChecker interface {
	IsOnline(ctx context.Context, userID string) bool
}

type Service interface {
	CreateDepartment(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateDepartmentRequest) (*DepartmentResponse, error)
	DeleteDepartment(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error
	GetDirectory(ctx context.Context, actorID uuid.UUID) (*DirectoryResponse, error)
}

type service struct {
	repo     Repository
	authSvc  AuthService
	taskSvc  WorkloadService
	presence PresenceChecker
}

func NewService(repo Repository, authSvc AuthService, taskSvc WorkloadService, presence PresenceChecker) Service {
	return &service{repo: repo, authSvc: authSvc, taskSvc: taskSvc, presence: presence}
}

func (s *service) CreateDepartment(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateDepartmentRequest) (*DepartmentResponse, error) {
	if !auth.IsOrgAdmin(actorRole) {
		return nil, apperrors.Forbidden("only an admin can manage departments")
	}
	actorOrgID, err := s.authSvc.GetOrganizationID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	d := &Department{Name: req.Name, OrganizationID: &actorOrgID}
	if err := s.repo.CreateDepartment(ctx, d); err != nil {
		return nil, apperrors.Conflict("a department with this name already exists")
	}
	resp := toDepartmentResponse(d)
	return &resp, nil
}

func (s *service) DeleteDepartment(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error {
	if !auth.IsOrgAdmin(actorRole) {
		return apperrors.Forbidden("only an admin can manage departments")
	}
	actorOrgID, err := s.authSvc.GetOrganizationID(ctx, actorID)
	if err != nil {
		return err
	}
	// Confirm the department belongs to the actor's own organization
	// before deleting — an org admin's reach stops at their own
	// organization's boundary, same as everywhere else in P1.2. A
	// wrong-organization ID and a genuinely nonexistent one both just
	// look "not found" here, not "forbidden", for the same no-leak reason
	// as project.Service.CheckMembership.
	d, err := s.repo.FindDepartment(ctx, id)
	if err != nil {
		return apperrors.NotFound("department not found")
	}
	if d.OrganizationID == nil || *d.OrganizationID != actorOrgID {
		return apperrors.NotFound("department not found")
	}
	if err := s.repo.DeleteDepartment(ctx, id); err != nil {
		return apperrors.Internal("failed to delete department")
	}
	return nil
}

func (s *service) GetDirectory(ctx context.Context, actorID uuid.UUID) (*DirectoryResponse, error) {
	actorOrgID, err := s.authSvc.GetOrganizationID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	users, err := s.authSvc.ListUsersByOrganization(ctx, actorOrgID)
	if err != nil {
		return nil, err
	}
	departments, err := s.repo.ListDepartments(ctx, actorOrgID)
	if err != nil {
		return nil, apperrors.Internal("failed to list departments")
	}
	deptNameByID := make(map[string]string, len(departments))
	for _, d := range departments {
		deptNameByID[d.ID.String()] = d.Name
	}

	members := make([]DirectoryEntry, len(users))
	for i, u := range users {
		userID, err := uuid.Parse(u.ID)
		if err != nil {
			continue
		}
		workload, err := s.taskSvc.GetWorkload(ctx, userID)
		if err != nil {
			return nil, err
		}

		entry := DirectoryEntry{
			UserID:    u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Role:      string(u.Role),
			AvatarURL: u.AvatarURL,
			Online:    s.presence.IsOnline(ctx, u.ID),
			Workload:  workload,
		}
		if u.DepartmentID != nil {
			entry.DepartmentID = *u.DepartmentID
			entry.DepartmentName = deptNameByID[*u.DepartmentID]
		}
		members[i] = entry
	}

	deptResponses := make([]DepartmentResponse, len(departments))
	for i := range departments {
		deptResponses[i] = toDepartmentResponse(&departments[i])
	}

	return &DirectoryResponse{Members: members, Departments: deptResponses}, nil
}
