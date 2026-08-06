package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// fakeRepository is an in-memory stand-in for Repository. Only the
// directory-related methods are exercised by these tests (Register/Login
// need a real TokenService + Redis, out of scope here).
type fakeRepository struct {
	users map[uuid.UUID]*User
}

func newFakeRepository(users ...*User) *fakeRepository {
	repo := &fakeRepository{users: make(map[uuid.UUID]*User)}
	for _, u := range users {
		repo.users[u.ID] = u
	}
	return repo
}

func (f *fakeRepository) Create(context.Context, *User) error { panic("not implemented") }
func (f *fakeRepository) FindByEmail(context.Context, string) (*User, error) {
	panic("not implemented")
}

func (f *fakeRepository) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepository) UpdatePassword(context.Context, uuid.UUID, string) error {
	panic("not implemented")
}

func (f *fakeRepository) ListAll(_ context.Context) ([]User, error) {
	var result []User
	for _, u := range f.users {
		result = append(result, *u)
	}
	return result, nil
}

func (f *fakeRepository) FindByIDs(_ context.Context, ids []uuid.UUID) ([]User, error) {
	var result []User
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			result = append(result, *u)
		}
	}
	return result, nil
}

func (f *fakeRepository) UpdateRole(_ context.Context, id uuid.UUID, role Role) error {
	u, ok := f.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.Role = role
	return nil
}

func (f *fakeRepository) UpdateDepartment(_ context.Context, id uuid.UUID, departmentID *uuid.UUID) error {
	u, ok := f.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.DepartmentID = departmentID
	return nil
}

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *apperrors.AppError, got %T (%v)", err, err)
	}
	return appErr.Status
}

func TestListUsers_ReturnsAllUsers(t *testing.T) {
	alice := &User{ID: uuid.New(), Name: "Alice", Email: "alice@example.com", Role: RoleEmployee}
	bob := &User{ID: uuid.New(), Name: "Bob", Email: "bob@example.com", Role: RoleManager}
	svc := NewService(newFakeRepository(alice, bob), nil)

	users, err := svc.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestGetUsersByIDs_ReturnsOnlyRequested(t *testing.T) {
	alice := &User{ID: uuid.New(), Name: "Alice", Email: "alice@example.com", Role: RoleEmployee}
	bob := &User{ID: uuid.New(), Name: "Bob", Email: "bob@example.com", Role: RoleManager}
	svc := NewService(newFakeRepository(alice, bob), nil)

	users, err := svc.GetUsersByIDs(context.Background(), []uuid.UUID{alice.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 || users[0].ID != alice.ID.String() {
		t.Errorf("expected only alice, got %v", users)
	}
}

func TestUpdateRole_ByNonAdminForbidden(t *testing.T) {
	target := &User{ID: uuid.New(), Name: "Target", Email: "target@example.com", Role: RoleEmployee}
	svc := NewService(newFakeRepository(target), nil)

	_, err := svc.UpdateRole(context.Background(), RoleEmployee, target.ID, UpdateRoleRequest{Role: RoleAdmin})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestUpdateRole_ByAdminSucceeds(t *testing.T) {
	target := &User{ID: uuid.New(), Name: "Target", Email: "target@example.com", Role: RoleEmployee}
	svc := NewService(newFakeRepository(target), nil)

	updated, err := svc.UpdateRole(context.Background(), RoleAdmin, target.ID, UpdateRoleRequest{Role: RoleManager})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Role != RoleManager {
		t.Errorf("expected role %q, got %q", RoleManager, updated.Role)
	}
}

func TestUpdateDepartment_ByNonAdminForbidden(t *testing.T) {
	target := &User{ID: uuid.New(), Name: "Target", Email: "target@example.com", Role: RoleEmployee}
	svc := NewService(newFakeRepository(target), nil)

	deptID := uuid.New().String()
	_, err := svc.UpdateDepartment(context.Background(), RoleEmployee, target.ID, UpdateDepartmentRequest{DepartmentID: &deptID})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestUpdateDepartment_ByAdminSucceeds(t *testing.T) {
	target := &User{ID: uuid.New(), Name: "Target", Email: "target@example.com", Role: RoleEmployee}
	svc := NewService(newFakeRepository(target), nil)

	deptID := uuid.New().String()
	updated, err := svc.UpdateDepartment(context.Background(), RoleSuperAdmin, target.ID, UpdateDepartmentRequest{DepartmentID: &deptID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DepartmentID == nil || *updated.DepartmentID != deptID {
		t.Errorf("expected department_id %q, got %v", deptID, updated.DepartmentID)
	}
}
