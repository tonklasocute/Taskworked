package project

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// fakeRepository is an in-memory stand-in for Repository, so the service's
// permission rules can be tested without a real database.
type fakeRepository struct {
	projects map[uuid.UUID]*Project
	members  map[uuid.UUID]map[uuid.UUID]*Member // projectID -> userID -> member
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		projects: make(map[uuid.UUID]*Project),
		members:  make(map[uuid.UUID]map[uuid.UUID]*Member),
	}
}

func (f *fakeRepository) Create(_ context.Context, p *Project) error {
	p.ID = uuid.New()
	f.projects[p.ID] = p
	return nil
}

func (f *fakeRepository) FindByID(_ context.Context, id uuid.UUID) (*Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (f *fakeRepository) ListForMember(_ context.Context, userID uuid.UUID, filter ListFilter) ([]Project, int64, error) {
	var result []Project
	for projectID, members := range f.members {
		if _, ok := members[userID]; ok {
			result = append(result, *f.projects[projectID])
		}
	}
	return result, int64(len(result)), nil
}

func (f *fakeRepository) Update(_ context.Context, p *Project) error {
	f.projects[p.ID] = p
	return nil
}

func (f *fakeRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(f.projects, id)
	return nil
}

func (f *fakeRepository) AddMember(_ context.Context, m *Member) error {
	if f.members[m.ProjectID] == nil {
		f.members[m.ProjectID] = make(map[uuid.UUID]*Member)
	}
	f.members[m.ProjectID][m.UserID] = m
	return nil
}

func (f *fakeRepository) RemoveMember(_ context.Context, projectID, userID uuid.UUID) error {
	delete(f.members[projectID], userID)
	return nil
}

func (f *fakeRepository) FindMember(_ context.Context, projectID, userID uuid.UUID) (*Member, error) {
	m, ok := f.members[projectID][userID]
	if !ok {
		return nil, ErrNotFound
	}
	return m, nil
}

func (f *fakeRepository) ListMembers(_ context.Context, projectID uuid.UUID) ([]Member, error) {
	var result []Member
	for _, m := range f.members[projectID] {
		result = append(result, *m)
	}
	return result, nil
}

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *apperrors.AppError, got %T (%v)", err, err)
	}
	return appErr.Status
}

func TestCreate_AddsCreatorAsOwnerMember(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()

	resp, err := svc.Create(context.Background(), owner, CreateRequest{Name: "Website Revamp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	projectID, _ := uuid.Parse(resp.ID)
	member, err := repo.FindMember(context.Background(), projectID, owner)
	if err != nil {
		t.Fatalf("expected creator to be a member: %v", err)
	}
	if member.Role != MemberRoleOwner {
		t.Errorf("expected creator role %q, got %q", MemberRoleOwner, member.Role)
	}
	if resp.Status != StatusPlanning {
		t.Errorf("expected default status %q, got %q", StatusPlanning, resp.Status)
	}
}

func TestGet_NonMemberIsForbidden(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()
	outsider := uuid.New()

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)

	_, err := svc.Get(context.Background(), outsider, auth.RoleEmployee, projectID)
	if err == nil {
		t.Fatal("expected forbidden error for non-member, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestGet_OrgAdminCanViewAnyProject(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()
	admin := uuid.New()

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)

	if _, err := svc.Get(context.Background(), admin, auth.RoleAdmin, projectID); err != nil {
		t.Fatalf("expected org admin to view any project, got error: %v", err)
	}
}

func TestUpdate_PlainMemberCannotUpdate(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()
	member := uuid.New()

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)
	_ = repo.AddMember(context.Background(), &Member{ProjectID: projectID, UserID: member, Role: MemberRoleMember})

	newName := "Renamed"
	_, err := svc.Update(context.Background(), member, auth.RoleEmployee, projectID, UpdateRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected forbidden error for plain member, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestUpdate_OwnerCanUpdate(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)

	newName := "Renamed"
	updated, err := svc.Update(context.Background(), owner, auth.RoleEmployee, projectID, UpdateRequest{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Errorf("expected name %q, got %q", "Renamed", updated.Name)
	}
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)

	err := svc.RemoveMember(context.Background(), owner, auth.RoleEmployee, projectID, owner)
	if err == nil {
		t.Fatal("expected error when removing the owner, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

// fakeAuthService is a minimal stand-in for auth.Service — only
// GetUsersByIDs is exercised by ListMembers.
type fakeAuthService struct {
	usersByID map[uuid.UUID]auth.UserResponse
}

func newFakeAuthService(users ...auth.UserResponse) *fakeAuthService {
	f := &fakeAuthService{usersByID: make(map[uuid.UUID]auth.UserResponse)}
	for _, u := range users {
		id, _ := uuid.Parse(u.ID)
		f.usersByID[id] = u
	}
	return f
}

func (f *fakeAuthService) GetUsersByIDs(_ context.Context, ids []uuid.UUID) ([]auth.UserResponse, error) {
	var result []auth.UserResponse
	for _, id := range ids {
		if u, ok := f.usersByID[id]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (f *fakeAuthService) Register(context.Context, auth.RegisterRequest) (*auth.AuthResponse, error) {
	panic("not implemented")
}
func (f *fakeAuthService) Login(context.Context, auth.LoginRequest) (*auth.AuthResponse, error) {
	panic("not implemented")
}
func (f *fakeAuthService) Refresh(context.Context, auth.RefreshRequest) (*auth.AuthResponse, error) {
	panic("not implemented")
}
func (f *fakeAuthService) Logout(context.Context, string) error { panic("not implemented") }
func (f *fakeAuthService) ListUsers(context.Context) ([]auth.UserResponse, error) {
	panic("not implemented")
}
func (f *fakeAuthService) UpdateRole(context.Context, auth.Role, uuid.UUID, auth.UpdateRoleRequest) (*auth.UserResponse, error) {
	panic("not implemented")
}
func (f *fakeAuthService) UpdateDepartment(context.Context, auth.Role, uuid.UUID, auth.UpdateDepartmentRequest) (*auth.UserResponse, error) {
	panic("not implemented")
}

func TestListMembers_ReturnsEnrichedDetails(t *testing.T) {
	repo := newFakeRepository()
	owner := uuid.New()
	authSvc := newFakeAuthService(auth.UserResponse{ID: owner.String(), Name: "Alice", Email: "alice@example.com"})
	svc := NewService(repo, authSvc)

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)

	members, err := svc.ListMembers(context.Background(), owner, auth.RoleEmployee, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 1 || members[0].Name != "Alice" {
		t.Errorf("expected 1 enriched member named Alice, got %v", members)
	}
}

func TestListMembers_NonMemberForbidden(t *testing.T) {
	repo := newFakeRepository()
	owner := uuid.New()
	outsider := uuid.New()
	svc := NewService(repo, newFakeAuthService())

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)

	_, err := svc.ListMembers(context.Background(), outsider, auth.RoleEmployee, projectID)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestAddMember_ByPlainMemberIsForbidden(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, nil)
	owner := uuid.New()
	member := uuid.New()
	newUser := uuid.New()

	resp, _ := svc.Create(context.Background(), owner, CreateRequest{Name: "Internal Tool"})
	projectID, _ := uuid.Parse(resp.ID)
	_ = repo.AddMember(context.Background(), &Member{ProjectID: projectID, UserID: member, Role: MemberRoleMember})

	err := svc.AddMember(context.Background(), member, auth.RoleEmployee, projectID, AddMemberRequest{UserID: newUser.String()})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}
