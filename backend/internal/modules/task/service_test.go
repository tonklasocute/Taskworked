package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// --- fakes ---------------------------------------------------------------

// fakeProjectService is a minimal stand-in for project.Service. Only
// CheckMembership is exercised by the task service; everything else
// panics if called, so an accidental dependency shows up immediately.
type fakeProjectService struct {
	// membership[projectID][userID] = isManager
	membership map[uuid.UUID]map[uuid.UUID]bool
}

func newFakeProjectService() *fakeProjectService {
	return &fakeProjectService{membership: make(map[uuid.UUID]map[uuid.UUID]bool)}
}

func (f *fakeProjectService) addMember(projectID, userID uuid.UUID, isManager bool) {
	if f.membership[projectID] == nil {
		f.membership[projectID] = make(map[uuid.UUID]bool)
	}
	f.membership[projectID][userID] = isManager
}

func (f *fakeProjectService) CheckMembership(_ context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (bool, bool, error) {
	if auth.IsOrgAdmin(actorRole) {
		return true, true, nil
	}
	isManager, ok := f.membership[projectID][actorID]
	return ok, ok && isManager, nil
}

func (f *fakeProjectService) Create(context.Context, uuid.UUID, project.CreateRequest) (*project.Response, error) {
	panic("not implemented")
}
func (f *fakeProjectService) Get(context.Context, uuid.UUID, auth.Role, uuid.UUID) (*project.Response, error) {
	panic("not implemented")
}
func (f *fakeProjectService) List(context.Context, uuid.UUID, project.ListFilter) (*project.ListResponse, error) {
	panic("not implemented")
}
func (f *fakeProjectService) Update(context.Context, uuid.UUID, auth.Role, uuid.UUID, project.UpdateRequest) (*project.Response, error) {
	panic("not implemented")
}
func (f *fakeProjectService) Delete(context.Context, uuid.UUID, auth.Role, uuid.UUID) error {
	panic("not implemented")
}
func (f *fakeProjectService) AddMember(context.Context, uuid.UUID, auth.Role, uuid.UUID, project.AddMemberRequest) error {
	panic("not implemented")
}
func (f *fakeProjectService) RemoveMember(context.Context, uuid.UUID, auth.Role, uuid.UUID, uuid.UUID) error {
	panic("not implemented")
}

// fakeRepository is an in-memory stand-in for task.Repository.
type fakeRepository struct {
	tasks     map[uuid.UUID]*Task
	tags      map[uuid.UUID][]string
	checklist map[uuid.UUID]*ChecklistItem
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		tasks:     make(map[uuid.UUID]*Task),
		tags:      make(map[uuid.UUID][]string),
		checklist: make(map[uuid.UUID]*ChecklistItem),
	}
}

func (f *fakeRepository) Create(_ context.Context, t *Task) error {
	t.ID = uuid.New()
	f.tasks[t.ID] = t
	return nil
}

func (f *fakeRepository) FindByID(_ context.Context, id uuid.UUID) (*Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (f *fakeRepository) List(_ context.Context, filter ListFilter) ([]Task, int64, error) {
	var result []Task
	for _, t := range f.tasks {
		if t.ProjectID.String() == filter.ProjectID {
			result = append(result, *t)
		}
	}
	return result, int64(len(result)), nil
}

func (f *fakeRepository) Update(_ context.Context, t *Task) error {
	f.tasks[t.ID] = t
	return nil
}

func (f *fakeRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(f.tasks, id)
	return nil
}

func (f *fakeRepository) SetTags(_ context.Context, taskID uuid.UUID, tags []string) error {
	f.tags[taskID] = tags
	return nil
}

func (f *fakeRepository) Tags(_ context.Context, taskID uuid.UUID) ([]string, error) {
	return f.tags[taskID], nil
}

func (f *fakeRepository) AddChecklistItem(_ context.Context, item *ChecklistItem) error {
	item.ID = uuid.New()
	f.checklist[item.ID] = item
	return nil
}

func (f *fakeRepository) UpdateChecklistItem(_ context.Context, item *ChecklistItem) error {
	f.checklist[item.ID] = item
	return nil
}

func (f *fakeRepository) DeleteChecklistItem(_ context.Context, id uuid.UUID) error {
	delete(f.checklist, id)
	return nil
}

func (f *fakeRepository) FindChecklistItem(_ context.Context, id uuid.UUID) (*ChecklistItem, error) {
	item, ok := f.checklist[id]
	if !ok {
		return nil, ErrNotFound
	}
	return item, nil
}

func (f *fakeRepository) ListChecklistItems(_ context.Context, taskID uuid.UUID) ([]ChecklistItem, error) {
	var items []ChecklistItem
	for _, item := range f.checklist {
		if item.TaskID == taskID {
			items = append(items, *item)
		}
	}
	return items, nil
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

func TestCreate_MemberCanCreateTaskAsReporter(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, member, false)

	resp, err := svc.Create(context.Background(), member, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ReporterID != member.String() {
		t.Errorf("expected reporter %q, got %q", member, resp.ReporterID)
	}
	if resp.Status != StatusBacklog {
		t.Errorf("expected default status %q, got %q", StatusBacklog, resp.Status)
	}
	if resp.Priority != PriorityMedium {
		t.Errorf("expected default priority %q, got %q", PriorityMedium, resp.Priority)
	}
}

func TestCreate_NonMemberIsForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	outsider := uuid.New()

	_, err := svc.Create(context.Background(), outsider, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
	})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestCreate_AssigneeMustBeProjectMember(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	nonMemberAssignee := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	assigneeStr := nonMemberAssignee.String()
	_, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID:  projectID.String(),
		Title:      "Set up CI",
		AssigneeID: &assigneeStr,
	})
	if err == nil {
		t.Fatal("expected bad request error, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestCreate_ParentTaskMustBeSameProject(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectA := uuid.New()
	projectB := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectA, reporter, false)
	projSvc.addMember(projectB, reporter, false)

	parent, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectB.String(),
		Title:     "Parent in project B",
	})
	if err != nil {
		t.Fatalf("unexpected error creating parent: %v", err)
	}

	_, err = svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID:    projectA.String(),
		ParentTaskID: &parent.ID,
		Title:        "Child in project A",
	})
	if err == nil {
		t.Fatal("expected bad request error for cross-project parent, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestUpdate_PlainMemberCannotEditOthersTask(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	bystander := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, bystander, false)

	created, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
	})

	newTitle := "Renamed"
	_, err := svc.Update(context.Background(), bystander, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Title: &newTitle})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestUpdate_AssigneeCanEdit(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, assignee, false)

	assigneeStr := assignee.String()
	created, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID:  projectID.String(),
		Title:      "Set up CI",
		AssigneeID: &assigneeStr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newStatus := StatusDoing
	updated, err := svc.Update(context.Background(), assignee, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Status: &newStatus})
	if err != nil {
		t.Fatalf("expected assignee to edit task, got error: %v", err)
	}
	if updated.Status != StatusDoing {
		t.Errorf("expected status %q, got %q", StatusDoing, updated.Status)
	}
}

func TestUpdate_ProjectManagerCanEditAnyTask(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	manager := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, manager, true)

	created, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
	})

	newTitle := "Renamed by manager"
	updated, err := svc.Update(context.Background(), manager, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Title: &newTitle})
	if err != nil {
		t.Fatalf("expected project manager to edit task, got error: %v", err)
	}
	if updated.Title != "Renamed by manager" {
		t.Errorf("expected title %q, got %q", "Renamed by manager", updated.Title)
	}
}

func TestDelete_OnlyReporterOrManagerCanDelete(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	bystander := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, bystander, false)

	created, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
	})

	err := svc.Delete(context.Background(), bystander, auth.RoleEmployee, mustParse(t, created.ID))
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}

	if err := svc.Delete(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID)); err != nil {
		t.Fatalf("expected reporter to delete task, got error: %v", err)
	}
}

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse uuid %q: %v", s, err)
	}
	return id
}
