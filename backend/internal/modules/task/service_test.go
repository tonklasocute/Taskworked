package task

import (
	"context"
	"errors"
	"testing"
	"time"

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
func (f *fakeProjectService) ListMembers(context.Context, uuid.UUID, auth.Role, uuid.UUID) ([]project.MemberDetail, error) {
	panic("not implemented")
}

// fakeRepository is an in-memory stand-in for task.Repository.
type fakeRepository struct {
	tasks        map[uuid.UUID]*Task
	tags         map[uuid.UUID][]string
	checklist    map[uuid.UUID]*ChecklistItem
	dependencies []Dependency
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

func (f *fakeRepository) ListAllForProject(_ context.Context, projectID uuid.UUID) ([]Task, error) {
	var result []Task
	for _, t := range f.tasks {
		if t.ProjectID == projectID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (f *fakeRepository) AddDependency(_ context.Context, d *Dependency) error {
	for _, existing := range f.dependencies {
		if existing.TaskID == d.TaskID && existing.DependsOnID == d.DependsOnID {
			return errors.New("dependency already exists")
		}
	}
	f.dependencies = append(f.dependencies, *d)
	return nil
}

func (f *fakeRepository) RemoveDependency(_ context.Context, taskID, dependsOnID uuid.UUID) error {
	for i, d := range f.dependencies {
		if d.TaskID == taskID && d.DependsOnID == dependsOnID {
			f.dependencies = append(f.dependencies[:i], f.dependencies[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeRepository) ListDependenciesForProject(_ context.Context, projectID uuid.UUID) ([]Dependency, error) {
	var result []Dependency
	for _, d := range f.dependencies {
		if task, ok := f.tasks[d.TaskID]; ok && task.ProjectID == projectID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (f *fakeRepository) CountActiveByAssignee(_ context.Context, assigneeID uuid.UUID) (int64, error) {
	var count int64
	for _, t := range f.tasks {
		if t.AssigneeID != nil && *t.AssigneeID == assigneeID && t.Status != StatusDone {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepository) ListActiveByAssignee(_ context.Context, assigneeID uuid.UUID) ([]Task, error) {
	var result []Task
	for _, t := range f.tasks {
		if t.AssigneeID != nil && *t.AssigneeID == assigneeID && t.Status != StatusDone {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (f *fakeRepository) ListCompletedByAssigneeSince(_ context.Context, assigneeID uuid.UUID, since time.Time) ([]Task, error) {
	var result []Task
	for _, t := range f.tasks {
		if t.AssigneeID != nil && *t.AssigneeID == assigneeID && t.Status == StatusDone && t.CompletedAt != nil && !t.CompletedAt.Before(since) {
			result = append(result, *t)
		}
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

// --- tests -----------------------------------------------------------------

func TestCreate_MemberCanCreateTaskAsReporter(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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
	svc := NewService(repo, projSvc, nil, nil)

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

func TestCreate_StartDateAfterDueDateIsRejected(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	start := "2026-08-10"
	due := "2026-08-01"
	_, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
		StartDate: &start,
		DueDate:   &due,
	})
	if err == nil {
		t.Fatal("expected bad request error, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestUpdate_StartDateAfterExistingDueDateIsRejected(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	due := "2026-08-01"
	created, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
		DueDate:   &due,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lateStart := "2026-08-10"
	_, err = svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{StartDate: &lateStart})
	if err == nil {
		t.Fatal("expected bad request error, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestUpdate_ValidStartAndDueDateSpanIsAccepted(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	created, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(),
		Title:     "Set up CI",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	start := "2026-08-01"
	due := "2026-08-05"
	updated, err := svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{StartDate: &start, DueDate: &due})
	if err != nil {
		t.Fatalf("expected valid span to be accepted, got error: %v", err)
	}
	if updated.StartDate == nil || updated.DueDate == nil {
		t.Fatal("expected both start_date and due_date to be set")
	}
	if *updated.StartDate != start || *updated.DueDate != due {
		t.Errorf("expected start_date=%q due_date=%q, got start=%q due=%q", start, due, *updated.StartDate, *updated.DueDate)
	}
}

func TestAddDependency_Success(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	a, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "A"})
	b, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "B"})

	err := svc.AddDependency(context.Background(), reporter, auth.RoleEmployee, mustParse(t, b.ID), AddDependencyRequest{DependsOnTaskID: a.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deps, err := repo.ListDependenciesForProject(context.Background(), projectID)
	if err != nil || len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %v (err=%v)", deps, err)
	}
}

func TestAddDependency_SelfDependencyRejected(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	a, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "A"})

	err := svc.AddDependency(context.Background(), reporter, auth.RoleEmployee, mustParse(t, a.ID), AddDependencyRequest{DependsOnTaskID: a.ID})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestAddDependency_CrossProjectRejected(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectA, projectB := uuid.New(), uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectA, reporter, false)
	projSvc.addMember(projectB, reporter, false)

	a, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectA.String(), Title: "A"})
	b, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectB.String(), Title: "B"})

	err := svc.AddDependency(context.Background(), reporter, auth.RoleEmployee, mustParse(t, b.ID), AddDependencyRequest{DependsOnTaskID: a.ID})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestAddDependency_CycleRejected(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	a, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "A"})
	b, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "B"})

	// B depends on A.
	if err := svc.AddDependency(context.Background(), reporter, auth.RoleEmployee, mustParse(t, b.ID), AddDependencyRequest{DependsOnTaskID: a.ID}); err != nil {
		t.Fatalf("unexpected error setting up B->A: %v", err)
	}

	// Now try to make A depend on B — would close the loop.
	err := svc.AddDependency(context.Background(), reporter, auth.RoleEmployee, mustParse(t, a.ID), AddDependencyRequest{DependsOnTaskID: b.ID})
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if status := appErrStatus(t, err); status != 400 {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestAddDependency_ByNonEditorForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	bystander := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, bystander, false)

	a, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "A"})
	b, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "B"})

	err := svc.AddDependency(context.Background(), bystander, auth.RoleEmployee, mustParse(t, b.ID), AddDependencyRequest{DependsOnTaskID: a.ID})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestGetGanttView_ReturnsTasksDepsAndCriticalPath(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	start, due := "2026-08-01", "2026-08-03"
	a, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "A", StartDate: &start, DueDate: &due})
	b, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "B", StartDate: &due, DueDate: &due})

	if err := svc.AddDependency(context.Background(), reporter, auth.RoleEmployee, mustParse(t, b.ID), AddDependencyRequest{DependsOnTaskID: a.ID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	view, err := svc.GetGanttView(context.Background(), reporter, auth.RoleEmployee, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(view.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(view.Tasks))
	}
	if len(view.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(view.Dependencies))
	}
	if len(view.CriticalPath) != 2 || view.CriticalPath[0] != a.ID || view.CriticalPath[1] != b.ID {
		t.Errorf("expected critical path [A, B], got %v", view.CriticalPath)
	}
}

func TestGetGanttView_NonMemberForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	outsider := uuid.New()

	_, err := svc.GetGanttView(context.Background(), outsider, auth.RoleEmployee, projectID)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestListAllResponses_ReturnsAllProjectTasksForMember(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)

	svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "A"})
	svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "B"})

	responses, err := svc.ListAllResponses(context.Background(), reporter, auth.RoleEmployee, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(responses) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(responses))
	}
}

func TestListAllResponses_NonMemberForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	_, err := svc.ListAllResponses(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestUpdate_MovingToDoneSetsCompletedAt(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	created, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Set up CI"})

	done := StatusDone
	updated, err := svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Status: &done})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestUpdate_MovingAwayFromDoneClearsCompletedAt(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	created, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Set up CI"})

	done := StatusDone
	svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Status: &done})

	reopened := StatusDoing
	updated, err := svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Status: &reopened})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CompletedAt != nil {
		t.Errorf("expected completed_at to be cleared, got %v", *updated.CompletedAt)
	}
}

func TestGetWorkload_CountsActiveTasksAcrossProjectsExcludingDone(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectA := uuid.New()
	projectB := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	projSvc.addMember(projectA, reporter, false)
	projSvc.addMember(projectA, assignee, false)
	projSvc.addMember(projectB, reporter, false)
	projSvc.addMember(projectB, assignee, false)

	assigneeStr := assignee.String()
	t1, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectA.String(), Title: "A1", AssigneeID: &assigneeStr})
	_, _ = svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectB.String(), Title: "B1", AssigneeID: &assigneeStr})
	_, _ = svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectB.String(), Title: "B2 unassigned"})

	// Mark t1 done — it should drop out of the workload count.
	done := StatusDone
	svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, t1.ID), UpdateRequest{Status: &done})

	count, err := svc.GetWorkload(context.Background(), assignee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected workload 1 (only B1 active), got %d", count)
	}
}

type fakeNotifier struct {
	notified []uuid.UUID
}

func (f *fakeNotifier) NotifyAssignment(_ context.Context, assigneeID uuid.UUID, _, _ string) error {
	f.notified = append(f.notified, assigneeID)
	return nil
}

func TestCreate_NotifiesAssigneeWhenSetAtCreation(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	notifier := &fakeNotifier{}
	svc := NewService(repo, projSvc, nil, notifier)

	projectID := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, assignee, false)

	assigneeStr := assignee.String()
	_, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(), Title: "Set up CI", AssigneeID: &assigneeStr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifier.notified) != 1 || notifier.notified[0] != assignee {
		t.Errorf("expected assignee to be notified once, got %v", notifier.notified)
	}
}

func TestUpdate_NotifiesNewAssigneeOnChange(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	notifier := &fakeNotifier{}
	svc := NewService(repo, projSvc, nil, notifier)

	projectID := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, assignee, false)

	created, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Set up CI"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assigneeStr := assignee.String()
	_, err = svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{AssigneeID: &assigneeStr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifier.notified) != 1 || notifier.notified[0] != assignee {
		t.Errorf("expected assignee to be notified once, got %v", notifier.notified)
	}
}

func TestUpdate_DoesNotNotifyWhenAssigneeUnchanged(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	notifier := &fakeNotifier{}
	svc := NewService(repo, projSvc, nil, notifier)

	projectID := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, assignee, false)

	assigneeStr := assignee.String()
	created, err := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{
		ProjectID: projectID.String(), Title: "Set up CI", AssigneeID: &assigneeStr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	notifier.notified = nil // clear the creation-time notification

	newTitle := "Renamed"
	_, err = svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, created.ID), UpdateRequest{Title: &newTitle, AssigneeID: &assigneeStr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifier.notified) != 0 {
		t.Errorf("expected no notification for unchanged assignee, got %v", notifier.notified)
	}
}

func TestListActiveByAssignee_ExcludesDoneAndOtherUsers(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	other := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, assignee, false)
	projSvc.addMember(projectID, other, false)

	assigneeStr, otherStr := assignee.String(), other.String()
	active, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Active", AssigneeID: &assigneeStr})
	svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Someone else's", AssigneeID: &otherStr})
	done, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Done", AssigneeID: &assigneeStr})
	doneStatus := StatusDone
	svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, done.ID), UpdateRequest{Status: &doneStatus})

	results, err := svc.ListActiveByAssignee(context.Background(), assignee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != active.ID {
		t.Errorf("expected only the active task, got %v", results)
	}
}

func TestListCompletedByAssigneeSince_OnlyRecentCompletions(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, nil, nil)

	projectID := uuid.New()
	reporter := uuid.New()
	assignee := uuid.New()
	projSvc.addMember(projectID, reporter, false)
	projSvc.addMember(projectID, assignee, false)

	assigneeStr := assignee.String()
	recent, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Recent", AssigneeID: &assigneeStr})
	doneStatus := StatusDone
	svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, recent.ID), UpdateRequest{Status: &doneStatus})

	// Backdate its CompletedAt directly via the repo to simulate an old completion.
	old, _ := svc.Create(context.Background(), reporter, auth.RoleEmployee, CreateRequest{ProjectID: projectID.String(), Title: "Old", AssigneeID: &assigneeStr})
	svc.Update(context.Background(), reporter, auth.RoleEmployee, mustParse(t, old.ID), UpdateRequest{Status: &doneStatus})
	oldTask, _ := repo.FindByID(context.Background(), mustParse(t, old.ID))
	longAgo := time.Now().AddDate(0, 0, -30)
	oldTask.CompletedAt = &longAgo
	repo.Update(context.Background(), oldTask)

	results, err := svc.ListCompletedByAssigneeSince(context.Background(), assignee, time.Now().AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != recent.ID {
		t.Errorf("expected only the recent completion, got %v", results)
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
