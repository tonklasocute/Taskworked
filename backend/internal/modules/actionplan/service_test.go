package actionplan

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// --- fakes ---------------------------------------------------------------

type fakeProjectService struct {
	membership map[uuid.UUID]map[uuid.UUID]bool // projectID -> userID -> isManager
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

// fakeTaskService only implements ListAllResponses — the one method
// GetActionPlanView actually calls.
type fakeTaskService struct {
	responses []task.Response
}

func (f *fakeTaskService) ListAllResponses(context.Context, uuid.UUID, auth.Role, uuid.UUID) ([]task.Response, error) {
	return f.responses, nil
}
func (f *fakeTaskService) Create(context.Context, uuid.UUID, auth.Role, task.CreateRequest) (*task.Response, error) {
	panic("not implemented")
}
func (f *fakeTaskService) Get(context.Context, uuid.UUID, auth.Role, uuid.UUID) (*task.Response, error) {
	panic("not implemented")
}
func (f *fakeTaskService) List(context.Context, uuid.UUID, auth.Role, task.ListFilter) (*task.ListResponse, error) {
	panic("not implemented")
}
func (f *fakeTaskService) Update(context.Context, uuid.UUID, auth.Role, uuid.UUID, task.UpdateRequest) (*task.Response, error) {
	panic("not implemented")
}
func (f *fakeTaskService) Delete(context.Context, uuid.UUID, auth.Role, uuid.UUID) error {
	panic("not implemented")
}
func (f *fakeTaskService) SetTags(context.Context, uuid.UUID, auth.Role, uuid.UUID, task.SetTagsRequest) ([]string, error) {
	panic("not implemented")
}
func (f *fakeTaskService) AddChecklistItem(context.Context, uuid.UUID, auth.Role, uuid.UUID, task.AddChecklistItemRequest) (*task.ChecklistItemResponse, error) {
	panic("not implemented")
}
func (f *fakeTaskService) UpdateChecklistItem(context.Context, uuid.UUID, auth.Role, uuid.UUID, uuid.UUID, task.UpdateChecklistItemRequest) (*task.ChecklistItemResponse, error) {
	panic("not implemented")
}
func (f *fakeTaskService) DeleteChecklistItem(context.Context, uuid.UUID, auth.Role, uuid.UUID, uuid.UUID) error {
	panic("not implemented")
}
func (f *fakeTaskService) ListChecklistItems(context.Context, uuid.UUID, auth.Role, uuid.UUID) ([]task.ChecklistItemResponse, error) {
	panic("not implemented")
}
func (f *fakeTaskService) AddDependency(context.Context, uuid.UUID, auth.Role, uuid.UUID, task.AddDependencyRequest) error {
	panic("not implemented")
}
func (f *fakeTaskService) RemoveDependency(context.Context, uuid.UUID, auth.Role, uuid.UUID, uuid.UUID) error {
	panic("not implemented")
}
func (f *fakeTaskService) GetGanttView(context.Context, uuid.UUID, auth.Role, uuid.UUID) (*task.GanttResponse, error) {
	panic("not implemented")
}

// fakeRepository is an in-memory stand-in for actionplan.Repository.
type fakeRepository struct {
	goals      map[uuid.UUID]*Goal
	milestones map[uuid.UUID]*Milestone
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{goals: make(map[uuid.UUID]*Goal), milestones: make(map[uuid.UUID]*Milestone)}
}

func (f *fakeRepository) CreateGoal(_ context.Context, g *Goal) error {
	g.ID = uuid.New()
	f.goals[g.ID] = g
	return nil
}

func (f *fakeRepository) FindGoal(_ context.Context, id uuid.UUID) (*Goal, error) {
	g, ok := f.goals[id]
	if !ok {
		return nil, ErrNotFound
	}
	return g, nil
}

func (f *fakeRepository) ListGoalsForProject(_ context.Context, projectID uuid.UUID) ([]Goal, error) {
	var result []Goal
	for _, g := range f.goals {
		if g.ProjectID == projectID {
			result = append(result, *g)
		}
	}
	return result, nil
}

func (f *fakeRepository) UpdateGoal(_ context.Context, g *Goal) error {
	f.goals[g.ID] = g
	return nil
}

func (f *fakeRepository) DeleteGoal(_ context.Context, id uuid.UUID) error {
	delete(f.goals, id)
	return nil
}

func (f *fakeRepository) CreateMilestone(_ context.Context, m *Milestone) error {
	m.ID = uuid.New()
	f.milestones[m.ID] = m
	return nil
}

func (f *fakeRepository) FindMilestone(_ context.Context, id uuid.UUID) (*Milestone, error) {
	m, ok := f.milestones[id]
	if !ok {
		return nil, ErrNotFound
	}
	return m, nil
}

func (f *fakeRepository) ListMilestonesForGoal(_ context.Context, goalID uuid.UUID) ([]Milestone, error) {
	var result []Milestone
	for _, m := range f.milestones {
		if m.GoalID == goalID {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (f *fakeRepository) ListMilestonesForProject(_ context.Context, projectID uuid.UUID) ([]Milestone, error) {
	var result []Milestone
	for _, m := range f.milestones {
		if goal, ok := f.goals[m.GoalID]; ok && goal.ProjectID == projectID {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (f *fakeRepository) UpdateMilestone(_ context.Context, m *Milestone) error {
	f.milestones[m.ID] = m
	return nil
}

func (f *fakeRepository) DeleteMilestone(_ context.Context, id uuid.UUID) error {
	delete(f.milestones, id)
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

func TestCreateGoal_ByManagerSucceeds(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, &fakeTaskService{})

	projectID := uuid.New()
	manager := uuid.New()
	projSvc.addMember(projectID, manager, true)

	resp, err := svc.CreateGoal(context.Background(), manager, auth.RoleEmployee, CreateGoalRequest{ProjectID: projectID.String(), Name: "Launch v2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != StatusNotStarted {
		t.Errorf("expected default status %q, got %q", StatusNotStarted, resp.Status)
	}
}

func TestCreateGoal_ByPlainMemberForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, &fakeTaskService{})

	projectID := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, member, false)

	_, err := svc.CreateGoal(context.Background(), member, auth.RoleEmployee, CreateGoalRequest{ProjectID: projectID.String(), Name: "Launch v2"})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestCreateMilestone_UnderGoalByManagerSucceeds(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, &fakeTaskService{})

	projectID := uuid.New()
	manager := uuid.New()
	projSvc.addMember(projectID, manager, true)

	goal, err := svc.CreateGoal(context.Background(), manager, auth.RoleEmployee, CreateGoalRequest{ProjectID: projectID.String(), Name: "Launch v2"})
	if err != nil {
		t.Fatalf("unexpected error creating goal: %v", err)
	}

	milestone, err := svc.CreateMilestone(context.Background(), manager, auth.RoleEmployee, CreateMilestoneRequest{GoalID: goal.ID, Name: "Beta release"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if milestone.GoalID != goal.ID {
		t.Errorf("expected milestone goal_id %q, got %q", goal.ID, milestone.GoalID)
	}
}

func TestUpdateGoal_ByNonManagerForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, &fakeTaskService{})

	projectID := uuid.New()
	manager := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, manager, true)
	projSvc.addMember(projectID, member, false)

	goal, _ := svc.CreateGoal(context.Background(), manager, auth.RoleEmployee, CreateGoalRequest{ProjectID: projectID.String(), Name: "Launch v2"})

	newName := "Renamed"
	_, err := svc.UpdateGoal(context.Background(), member, auth.RoleEmployee, mustParse(t, goal.ID), UpdateGoalRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestGetActionPlanView_ComposesFullHierarchy(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()

	projectID := uuid.New()
	manager := uuid.New()
	projSvc.addMember(projectID, manager, true)

	// Goal/Milestone creation doesn't need the task service, so build them
	// with a bare instance first, then swap in one carrying the tasks that
	// reference the now-known milestone ID.
	setupSvc := NewService(repo, projSvc, &fakeTaskService{})
	goal, err := setupSvc.CreateGoal(context.Background(), manager, auth.RoleEmployee, CreateGoalRequest{ProjectID: projectID.String(), Name: "Launch v2"})
	if err != nil {
		t.Fatalf("unexpected error creating goal: %v", err)
	}
	milestone, err := setupSvc.CreateMilestone(context.Background(), manager, auth.RoleEmployee, CreateMilestoneRequest{GoalID: goal.ID, Name: "Beta"})
	if err != nil {
		t.Fatalf("unexpected error creating milestone: %v", err)
	}

	taskID := uuid.New().String()
	subtaskID := uuid.New().String()
	milestoneID := milestone.ID
	taskResp := task.Response{ID: taskID, Title: "Implement feature", MilestoneID: &milestoneID}
	subtaskResp := task.Response{ID: subtaskID, Title: "Write tests", ParentTaskID: &taskID}
	svc := NewService(repo, projSvc, &fakeTaskService{responses: []task.Response{taskResp, subtaskResp}})

	view, err := svc.GetActionPlanView(context.Background(), manager, auth.RoleEmployee, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(view.Goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(view.Goals))
	}
	if len(view.Goals[0].Milestones) != 1 {
		t.Fatalf("expected 1 milestone, got %d", len(view.Goals[0].Milestones))
	}
	tasks := view.Goals[0].Milestones[0].Tasks
	if len(tasks) != 1 {
		t.Fatalf("expected 1 top-level task under milestone, got %d", len(tasks))
	}
	if tasks[0].ID != taskID {
		t.Errorf("expected task %q, got %q", taskID, tasks[0].ID)
	}
	if len(tasks[0].Subtasks) != 1 || tasks[0].Subtasks[0].ID != subtaskID {
		t.Errorf("expected 1 subtask %q, got %v", subtaskID, tasks[0].Subtasks)
	}
}

func TestGetActionPlanView_NonMemberForbidden(t *testing.T) {
	repo := newFakeRepository()
	projSvc := newFakeProjectService()
	svc := NewService(repo, projSvc, &fakeTaskService{})

	_, err := svc.GetActionPlanView(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
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
