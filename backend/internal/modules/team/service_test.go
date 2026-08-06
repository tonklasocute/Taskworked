package team

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// --- fakes ---------------------------------------------------------------

type fakeAuthService struct {
	users []auth.UserResponse
}

func (f *fakeAuthService) ListUsers(context.Context) ([]auth.UserResponse, error) {
	return f.users, nil
}

type fakeWorkloadService struct {
	workloadByUser map[string]int64
}

func (f *fakeWorkloadService) GetWorkload(_ context.Context, userID uuid.UUID) (int64, error) {
	return f.workloadByUser[userID.String()], nil
}

type fakePresenceChecker struct {
	online map[string]bool
}

func (f *fakePresenceChecker) IsOnline(_ context.Context, userID string) bool {
	return f.online[userID]
}

type fakeRepository struct {
	departments map[uuid.UUID]*Department
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{departments: make(map[uuid.UUID]*Department)}
}

func (f *fakeRepository) CreateDepartment(_ context.Context, d *Department) error {
	for _, existing := range f.departments {
		if existing.Name == d.Name {
			return apperrors.Conflict("duplicate")
		}
	}
	d.ID = uuid.New()
	f.departments[d.ID] = d
	return nil
}

func (f *fakeRepository) ListDepartments(_ context.Context) ([]Department, error) {
	var result []Department
	for _, d := range f.departments {
		result = append(result, *d)
	}
	return result, nil
}

func (f *fakeRepository) FindDepartment(_ context.Context, id uuid.UUID) (*Department, error) {
	d, ok := f.departments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

func (f *fakeRepository) DeleteDepartment(_ context.Context, id uuid.UUID) error {
	delete(f.departments, id)
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

// --- tests -----------------------------------------------------------------

func TestCreateDepartment_ByNonAdminForbidden(t *testing.T) {
	svc := NewService(newFakeRepository(), &fakeAuthService{}, &fakeWorkloadService{}, &fakePresenceChecker{})

	_, err := svc.CreateDepartment(context.Background(), auth.RoleEmployee, CreateDepartmentRequest{Name: "Engineering"})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestCreateDepartment_ByAdminSucceeds(t *testing.T) {
	svc := NewService(newFakeRepository(), &fakeAuthService{}, &fakeWorkloadService{}, &fakePresenceChecker{})

	resp, err := svc.CreateDepartment(context.Background(), auth.RoleAdmin, CreateDepartmentRequest{Name: "Engineering"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Engineering" {
		t.Errorf("expected name 'Engineering', got %q", resp.Name)
	}
}

func TestGetDirectory_ComposesUserDepartmentOnlineAndWorkload(t *testing.T) {
	repo := newFakeRepository()
	dept, _ := (&service{repo: repo}).CreateDepartment(context.Background(), auth.RoleAdmin, CreateDepartmentRequest{Name: "Engineering"})

	alice := auth.UserResponse{ID: uuid.New().String(), Name: "Alice", Email: "alice@example.com", Role: auth.RoleEmployee, DepartmentID: &dept.ID}
	bob := auth.UserResponse{ID: uuid.New().String(), Name: "Bob", Email: "bob@example.com", Role: auth.RoleManager}

	authSvc := &fakeAuthService{users: []auth.UserResponse{alice, bob}}
	taskSvc := &fakeWorkloadService{workloadByUser: map[string]int64{alice.ID: 3, bob.ID: 0}}
	presence := &fakePresenceChecker{online: map[string]bool{alice.ID: true}}

	svc := NewService(repo, authSvc, taskSvc, presence)

	dir, err := svc.GetDirectory(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dir.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(dir.Members))
	}

	byID := make(map[string]DirectoryEntry)
	for _, m := range dir.Members {
		byID[m.UserID] = m
	}

	aliceEntry := byID[alice.ID]
	if aliceEntry.DepartmentName != "Engineering" {
		t.Errorf("expected Alice's department to be Engineering, got %q", aliceEntry.DepartmentName)
	}
	if !aliceEntry.Online {
		t.Error("expected Alice to be online")
	}
	if aliceEntry.Workload != 3 {
		t.Errorf("expected Alice's workload 3, got %d", aliceEntry.Workload)
	}

	bobEntry := byID[bob.ID]
	if bobEntry.Online {
		t.Error("expected Bob to be offline")
	}
	if bobEntry.DepartmentName != "" {
		t.Errorf("expected Bob to have no department, got %q", bobEntry.DepartmentName)
	}
}

func TestDeleteDepartment_ByNonAdminForbidden(t *testing.T) {
	svc := NewService(newFakeRepository(), &fakeAuthService{}, &fakeWorkloadService{}, &fakePresenceChecker{})

	err := svc.DeleteDepartment(context.Background(), auth.RoleEmployee, uuid.New())
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}
