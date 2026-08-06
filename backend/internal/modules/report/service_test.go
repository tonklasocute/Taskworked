package report

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// --- fakes ---------------------------------------------------------------

type fakeProjectService struct {
	membership map[uuid.UUID]map[uuid.UUID]bool
}

func newFakeProjectService() *fakeProjectService {
	return &fakeProjectService{membership: make(map[uuid.UUID]map[uuid.UUID]bool)}
}

func (f *fakeProjectService) addMember(projectID, userID uuid.UUID) {
	if f.membership[projectID] == nil {
		f.membership[projectID] = make(map[uuid.UUID]bool)
	}
	f.membership[projectID][userID] = false
}

func (f *fakeProjectService) CheckMembership(_ context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (bool, bool, error) {
	if auth.IsOrgAdmin(actorRole) {
		return true, true, nil
	}
	_, ok := f.membership[projectID][actorID]
	return ok, false, nil
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

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *apperrors.AppError, got %T (%v)", err, err)
	}
	return appErr.Status
}

func strPtr(s string) *string { return &s }

// --- tests -----------------------------------------------------------------

func TestGetPeriodReport_NonMemberForbidden(t *testing.T) {
	projSvc := newFakeProjectService()
	svc := NewService(projSvc, &fakeTaskService{})

	_, err := svc.GetPeriodReport(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New(), time.Now().AddDate(0, 0, -7), time.Now())
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestGetPeriodReport_CountsCreatedAndCompletedInPeriod(t *testing.T) {
	projSvc := newFakeProjectService()
	projectID := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, member)

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 7, 23, 59, 59, 0, time.UTC)

	inPeriod := from.AddDate(0, 0, 2)
	beforePeriod := from.AddDate(0, 0, -3)

	tasks := []task.Response{
		{ID: "1", CreatedAt: inPeriod, CompletedAt: strPtr(inPeriod.Format(dateLayout)), Status: task.StatusDone},
		{ID: "2", CreatedAt: inPeriod, Status: task.StatusTodo},     // created in period, not completed
		{ID: "3", CreatedAt: beforePeriod, Status: task.StatusTodo}, // outside period entirely
	}
	svc := NewService(projSvc, &fakeTaskService{responses: tasks})

	report, err := svc.GetPeriodReport(context.Background(), member, auth.RoleEmployee, projectID, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Summary.TotalTasks != 3 {
		t.Errorf("expected total 3, got %d", report.Summary.TotalTasks)
	}
	if report.Summary.CreatedInPeriod != 2 {
		t.Errorf("expected 2 created in period, got %d", report.Summary.CreatedInPeriod)
	}
	if report.Summary.CompletedInPeriod != 1 {
		t.Errorf("expected 1 completed in period, got %d", report.Summary.CompletedInPeriod)
	}
	if report.Summary.CompletionRate != 0.5 {
		t.Errorf("expected completion rate 0.5, got %v", report.Summary.CompletionRate)
	}
}

func TestGetPeriodReport_OverdueAndNearDue(t *testing.T) {
	projSvc := newFakeProjectService()
	projectID := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, member)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format(dateLayout)
	inTwoDays := now.AddDate(0, 0, 2).Format(dateLayout)
	inTenDays := now.AddDate(0, 0, 10).Format(dateLayout)

	tasks := []task.Response{
		{ID: "overdue", CreatedAt: now, DueDate: &yesterday, Status: task.StatusTodo},
		{ID: "near-due", CreatedAt: now, DueDate: &inTwoDays, Status: task.StatusTodo},
		{ID: "far-out", CreatedAt: now, DueDate: &inTenDays, Status: task.StatusTodo},
		{ID: "done-but-overdue", CreatedAt: now, DueDate: &yesterday, Status: task.StatusDone}, // shouldn't count, it's done
	}
	svc := NewService(projSvc, &fakeTaskService{responses: tasks})

	report, err := svc.GetPeriodReport(context.Background(), member, auth.RoleEmployee, projectID, now.AddDate(0, 0, -7), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Summary.OverdueCount != 1 {
		t.Errorf("expected 1 overdue, got %d", report.Summary.OverdueCount)
	}
	if len(report.LateTasks) != 1 || report.LateTasks[0].ID != "overdue" {
		t.Errorf("expected late_tasks to contain only 'overdue', got %v", report.LateTasks)
	}
	if report.Summary.NearDueCount != 1 {
		t.Errorf("expected 1 near-due, got %d", report.Summary.NearDueCount)
	}
}

func TestGetPeriodReport_PerformanceOnTimeVsLate(t *testing.T) {
	projSvc := newFakeProjectService()
	projectID := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, member)

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	alice := "alice"

	dueOnTime := "2026-08-10"
	completedOnTime := "2026-08-09"
	dueLate := "2026-08-05"
	completedLate := "2026-08-08"

	tasks := []task.Response{
		{ID: "1", CreatedAt: from, AssigneeID: &alice, DueDate: &dueOnTime, CompletedAt: &completedOnTime, Status: task.StatusDone},
		{ID: "2", CreatedAt: from, AssigneeID: &alice, DueDate: &dueLate, CompletedAt: &completedLate, Status: task.StatusDone},
	}
	svc := NewService(projSvc, &fakeTaskService{responses: tasks})

	report, err := svc.GetPeriodReport(context.Background(), member, auth.RoleEmployee, projectID, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Performance) != 1 {
		t.Fatalf("expected 1 assignee, got %d", len(report.Performance))
	}
	perf := report.Performance[0]
	if perf.CompletedCount != 2 || perf.OnTimeCount != 1 || perf.LateCount != 1 {
		t.Errorf("expected completed=2 onTime=1 late=1, got completed=%d onTime=%d late=%d", perf.CompletedCount, perf.OnTimeCount, perf.LateCount)
	}
	if perf.OnTimeRate != 0.5 {
		t.Errorf("expected on_time_rate 0.5, got %v", perf.OnTimeRate)
	}
}

func TestGetPeriodReport_ProductivityBucketsByDayIncludingZeroDays(t *testing.T) {
	projSvc := newFakeProjectService()
	projectID := uuid.New()
	member := uuid.New()
	projSvc.addMember(projectID, member)

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	completedDay1 := "2026-08-01"

	tasks := []task.Response{
		{ID: "1", CreatedAt: from, CompletedAt: &completedDay1, Status: task.StatusDone},
	}
	svc := NewService(projSvc, &fakeTaskService{responses: tasks})

	report, err := svc.GetPeriodReport(context.Background(), member, auth.RoleEmployee, projectID, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Productivity) != 3 {
		t.Fatalf("expected 3 days (Aug 1-3), got %d", len(report.Productivity))
	}
	if report.Productivity[0].Date != "2026-08-01" || report.Productivity[0].CompletedCount != 1 {
		t.Errorf("expected day 1 to have 1 completion, got %+v", report.Productivity[0])
	}
	if report.Productivity[1].CompletedCount != 0 || report.Productivity[2].CompletedCount != 0 {
		t.Errorf("expected days 2-3 to have 0 completions, got %+v", report.Productivity[1:])
	}
}

func TestExportCSV_ProducesValidCSVWithHeaderAndRows(t *testing.T) {
	projSvc := newFakeProjectService()
	tasks := []task.Response{
		{ID: "1", Title: "Set up CI", Status: task.StatusDone, Priority: task.PriorityHigh, CreatedAt: time.Now()},
		{ID: "2", Title: "Write docs", Status: task.StatusTodo, Priority: task.PriorityLow, CreatedAt: time.Now()},
	}
	svc := NewService(projSvc, &fakeTaskService{responses: tasks})

	data, err := svc.ExportCSV(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatalf("produced invalid CSV: %v", err)
	}
	if len(rows) != 3 { // header + 2 tasks
		t.Fatalf("expected 3 rows (header + 2 tasks), got %d: %v", len(rows), rows)
	}
	if rows[0][0] != "Title" {
		t.Errorf("expected header row to start with Title, got %v", rows[0])
	}
	if rows[1][0] != "Set up CI" || rows[2][0] != "Write docs" {
		t.Errorf("unexpected task rows: %v", rows[1:])
	}
}
