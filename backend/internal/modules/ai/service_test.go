package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/report"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// --- fakes -------------------------------------------------------------

type fakeClient struct {
	responseJSON string
	err          error
	calls        int
}

func (f *fakeClient) CompleteJSON(_ context.Context, _, _ string, _ map[string]any, target any) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal([]byte(f.responseJSON), target)
}

type fakeProjectService struct {
	membership map[uuid.UUID]map[uuid.UUID]bool
	members    map[uuid.UUID][]project.MemberDetail
}

func newFakeProjectService() *fakeProjectService {
	return &fakeProjectService{
		membership: make(map[uuid.UUID]map[uuid.UUID]bool),
		members:    make(map[uuid.UUID][]project.MemberDetail),
	}
}

func (f *fakeProjectService) addMember(projectID, userID uuid.UUID) {
	if f.membership[projectID] == nil {
		f.membership[projectID] = make(map[uuid.UUID]bool)
	}
	f.membership[projectID][userID] = true
}

func (f *fakeProjectService) CheckMembership(_ context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (bool, bool, error) {
	if auth.IsOrgAdmin(actorRole) {
		return true, true, nil
	}
	_, ok := f.membership[projectID][actorID]
	return ok, false, nil
}

func (f *fakeProjectService) ListMembers(_ context.Context, _ uuid.UUID, _ auth.Role, id uuid.UUID) ([]project.MemberDetail, error) {
	return f.members[id], nil
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
	activeByAssignee []task.Response
	allResponses     []task.Response
	workload         map[uuid.UUID]int64
}

func (f *fakeTaskService) GetWorkload(_ context.Context, userID uuid.UUID) (int64, error) {
	return f.workload[userID], nil
}
func (f *fakeTaskService) ListActiveByAssignee(context.Context, uuid.UUID) ([]task.Response, error) {
	return f.activeByAssignee, nil
}
func (f *fakeTaskService) ListAllResponses(context.Context, uuid.UUID, auth.Role, uuid.UUID) ([]task.Response, error) {
	return f.allResponses, nil
}
func (f *fakeTaskService) ListCompletedByAssigneeSince(context.Context, uuid.UUID, time.Time) ([]task.Response, error) {
	panic("not implemented")
}
func (f *fakeTaskService) GetMySummary(context.Context, uuid.UUID) (*task.MySummaryResponse, error) {
	panic("not implemented")
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

type fakeReportService struct {
	period *report.PeriodReport
	err    error
}

func (f *fakeReportService) GetPeriodReport(context.Context, uuid.UUID, auth.Role, uuid.UUID, time.Time, time.Time) (*report.PeriodReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.period, nil
}
func (f *fakeReportService) ExportCSV(context.Context, uuid.UUID, auth.Role, uuid.UUID) ([]byte, error) {
	panic("not implemented")
}

func strPtr(s string) *string { return &s }

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *apperrors.AppError, got %T (%v)", err, err)
	}
	return appErr.Status
}

// --- tests ---------------------------------------------------------------

func TestGenerateTasks_NonMemberForbidden(t *testing.T) {
	client := &fakeClient{}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	_, err := svc.GenerateTasks(context.Background(), uuid.New(), auth.RoleEmployee, GenerateTasksRequest{
		ProjectID: uuid.New().String(), Prompt: "build a login page",
	})

	if appErrStatus(t, err) != 403 {
		t.Fatalf("expected 403, got %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("expected no AI call for a non-member, got %d calls", client.calls)
	}
}

func TestGenerateTasks_Success(t *testing.T) {
	actorID, projectID := uuid.New(), uuid.New()
	projSvc := newFakeProjectService()
	projSvc.addMember(projectID, actorID)
	client := &fakeClient{responseJSON: `{"tasks":[{"title":"Design schema","description":"Draft the DB schema","priority":"high","estimate_hours":4}]}`}
	svc := NewService(client, projSvc, &fakeTaskService{}, &fakeReportService{})

	result, err := svc.GenerateTasks(context.Background(), actorID, auth.RoleEmployee, GenerateTasksRequest{
		ProjectID: projectID.String(), Prompt: "build a login page",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Tasks) != 1 || result.Tasks[0].Title != "Design schema" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if client.calls != 1 {
		t.Fatalf("expected exactly 1 AI call, got %d", client.calls)
	}
}

func TestGenerateTasks_InvalidProjectID(t *testing.T) {
	svc := NewService(&fakeClient{}, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	_, err := svc.GenerateTasks(context.Background(), uuid.New(), auth.RoleEmployee, GenerateTasksRequest{
		ProjectID: "not-a-uuid", Prompt: "anything",
	})
	if appErrStatus(t, err) != 400 {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestSummarizeDailyTasks_NoActiveTasks_SkipsAPICall(t *testing.T) {
	client := &fakeClient{}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	result, err := svc.SummarizeDailyTasks(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("expected a canned summary for zero active tasks")
	}
	if client.calls != 0 {
		t.Fatalf("expected no AI call when there are no active tasks, got %d", client.calls)
	}
}

func TestSummarizeDailyTasks_Success(t *testing.T) {
	client := &fakeClient{responseJSON: `{"summary":"You have 2 tasks today, one is overdue."}`}
	taskSvc := &fakeTaskService{activeByAssignee: []task.Response{
		{ID: uuid.New().String(), Title: "Fix bug", Priority: task.PriorityHigh, Status: task.StatusDoing, DueDate: strPtr("2026-08-01")},
	}}
	svc := NewService(client, newFakeProjectService(), taskSvc, &fakeReportService{})

	result, err := svc.SummarizeDailyTasks(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "You have 2 tasks today, one is overdue." {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
}

func TestSummarizeWeeklyReport_PropagatesReportError(t *testing.T) {
	client := &fakeClient{}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{err: apperrors.Forbidden("not a member of this project")})

	_, err := svc.SummarizeWeeklyReport(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if appErrStatus(t, err) != 403 {
		t.Fatalf("expected 403, got %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("expected no AI call when the report lookup fails, got %d", client.calls)
	}
}

func TestSummarizeWeeklyReport_Success(t *testing.T) {
	client := &fakeClient{responseJSON: `{"summary":"Solid week — 5 of 6 tasks landed on time."}`}
	reportSvc := &fakeReportService{period: &report.PeriodReport{
		Summary:     report.Summary{CreatedInPeriod: 6, CompletedInPeriod: 5, CompletionRate: 0.83},
		Performance: []report.AssigneePerformance{{AssigneeID: uuid.New().String(), CompletedCount: 5, OnTimeCount: 4, LateCount: 1}},
	}}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, reportSvc)

	result, err := svc.SummarizeWeeklyReport(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("expected a non-empty summary")
	}
}

func TestEstimateDuration_Success(t *testing.T) {
	client := &fakeClient{responseJSON: `{"estimate_hours":3.5,"reasoning":"Straightforward CRUD form"}`}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	result, err := svc.EstimateDuration(context.Background(), EstimateRequest{Title: "Add settings form", Description: "A simple form"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EstimateHours != 3.5 {
		t.Fatalf("expected 3.5 hours, got %v", result.EstimateHours)
	}
}

func TestSuggestPriority_Success(t *testing.T) {
	client := &fakeClient{responseJSON: `{"priority":"critical","reasoning":"Production is down"}`}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	result, err := svc.SuggestPriority(context.Background(), PriorityRequest{Title: "Fix outage", Description: "Prod API is 500ing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Priority != "critical" {
		t.Fatalf("expected critical, got %v", result.Priority)
	}
}

func TestSuggestAssignee_NoMembers_BadRequest(t *testing.T) {
	actorID, projectID := uuid.New(), uuid.New()
	projSvc := newFakeProjectService()
	projSvc.addMember(projectID, actorID)
	client := &fakeClient{}
	svc := NewService(client, projSvc, &fakeTaskService{}, &fakeReportService{})

	_, err := svc.SuggestAssignee(context.Background(), actorID, auth.RoleEmployee, SuggestAssigneeRequest{
		ProjectID: projectID.String(), Title: "Fix bug", Description: "desc",
	})
	if appErrStatus(t, err) != 400 {
		t.Fatalf("expected 400, got %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("expected no AI call with zero members, got %d", client.calls)
	}
}

func TestSuggestAssignee_Success(t *testing.T) {
	actorID, projectID := uuid.New(), uuid.New()
	memberID := uuid.New()
	projSvc := newFakeProjectService()
	projSvc.addMember(projectID, actorID)
	projSvc.members[projectID] = []project.MemberDetail{{UserID: memberID.String(), Name: "Alice"}}
	taskSvc := &fakeTaskService{workload: map[uuid.UUID]int64{memberID: 2}}
	client := &fakeClient{responseJSON: `{"assignee_id":"` + memberID.String() + `","assignee_name":"Alice","reasoning":"Lowest current workload"}`}
	svc := NewService(client, projSvc, taskSvc, &fakeReportService{})

	result, err := svc.SuggestAssignee(context.Background(), actorID, auth.RoleEmployee, SuggestAssigneeRequest{
		ProjectID: projectID.String(), Title: "Fix bug", Description: "desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AssigneeID != memberID.String() {
		t.Fatalf("expected %s, got %s", memberID.String(), result.AssigneeID)
	}
}

func TestPredictLateTasks_NoEligibleTasks_SkipsAPICall(t *testing.T) {
	actorID, projectID := uuid.New(), uuid.New()
	projSvc := newFakeProjectService()
	projSvc.addMember(projectID, actorID)
	client := &fakeClient{}
	taskSvc := &fakeTaskService{allResponses: []task.Response{
		{ID: uuid.New().String(), Title: "Done task", Status: task.StatusDone, DueDate: strPtr("2026-08-01")},
		{ID: uuid.New().String(), Title: "No due date", Status: task.StatusDoing, DueDate: nil},
	}}
	svc := NewService(client, projSvc, taskSvc, &fakeReportService{})

	result, err := svc.PredictLateTasks(context.Background(), actorID, auth.RoleEmployee, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AtRisk) != 0 {
		t.Fatalf("expected no at-risk tasks, got %+v", result.AtRisk)
	}
	if client.calls != 0 {
		t.Fatalf("expected no AI call when nothing is eligible, got %d", client.calls)
	}
}

func TestPredictLateTasks_Success(t *testing.T) {
	actorID, projectID := uuid.New(), uuid.New()
	taskID := uuid.New()
	projSvc := newFakeProjectService()
	projSvc.addMember(projectID, actorID)
	taskSvc := &fakeTaskService{allResponses: []task.Response{
		{ID: taskID.String(), Title: "Overdue task", Status: task.StatusDoing, DueDate: strPtr("2026-01-01")},
	}}
	client := &fakeClient{responseJSON: `{"at_risk":[{"task_id":"` + taskID.String() + `","title":"Overdue task","reasoning":"Past its due date"}]}`}
	svc := NewService(client, projSvc, taskSvc, &fakeReportService{})

	result, err := svc.PredictLateTasks(context.Background(), actorID, auth.RoleEmployee, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AtRisk) != 1 || result.AtRisk[0].TaskID != taskID.String() {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestAnalyzeTeamProductivity_NoPerformance_SkipsAPICall(t *testing.T) {
	client := &fakeClient{}
	reportSvc := &fakeReportService{period: &report.PeriodReport{}}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, reportSvc)

	result, err := svc.AnalyzeTeamProductivity(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("expected a canned summary when there is no performance data")
	}
	if client.calls != 0 {
		t.Fatalf("expected no AI call with zero performance rows, got %d", client.calls)
	}
}

func TestAnalyzeTeamProductivity_Success(t *testing.T) {
	client := &fakeClient{responseJSON: `{"summary":"Team velocity is steady with one late outlier."}`}
	reportSvc := &fakeReportService{period: &report.PeriodReport{
		Performance: []report.AssigneePerformance{{AssigneeID: uuid.New().String(), CompletedCount: 10, OnTimeCount: 9, LateCount: 1, OnTimeRate: 0.9}},
	}}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, reportSvc)

	result, err := svc.AnalyzeTeamProductivity(context.Background(), uuid.New(), auth.RoleEmployee, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("expected a non-empty summary")
	}
}

func TestGenerateMeetingSummary_Success(t *testing.T) {
	client := &fakeClient{responseJSON: `{"summary":"Team agreed to ship v2 next sprint.","action_items":["Finalize API contract","Write migration script"]}`}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	result, err := svc.GenerateMeetingSummary(context.Background(), MeetingSummaryRequest{Notes: "We discussed v2 launch timing and agreed to ship next sprint."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ActionItems) != 2 {
		t.Fatalf("expected 2 action items, got %+v", result.ActionItems)
	}
}

func TestClientError_Propagates(t *testing.T) {
	client := &fakeClient{err: apperrors.Internal("AI request rejected: overloaded")}
	svc := NewService(client, newFakeProjectService(), &fakeTaskService{}, &fakeReportService{})

	_, err := svc.EstimateDuration(context.Background(), EstimateRequest{Title: "x", Description: "y"})
	if appErrStatus(t, err) != 500 {
		t.Fatalf("expected 500, got %v", err)
	}
}
