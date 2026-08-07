// Package ai implements the AI Assistant module: task generation, daily and
// weekly summaries, duration/priority/assignee suggestions, late-task
// prediction, team productivity analysis, and meeting summarization — each
// a thin prompt built from other modules' data, sent through Client, and
// parsed back into a typed response via structured outputs.
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/report"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

type Service interface {
	GenerateTasks(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req GenerateTasksRequest) (*GenerateTasksResponse, error)
	SummarizeDailyTasks(ctx context.Context, actorID uuid.UUID) (*SummaryResponse, error)
	SummarizeWeeklyReport(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*SummaryResponse, error)
	EstimateDuration(ctx context.Context, req EstimateRequest) (*EstimateResponse, error)
	SuggestPriority(ctx context.Context, req PriorityRequest) (*PriorityResponse, error)
	SuggestAssignee(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req SuggestAssigneeRequest) (*AssigneeSuggestion, error)
	PredictLateTasks(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*PredictLateTasksResponse, error)
	AnalyzeTeamProductivity(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*SummaryResponse, error)
	GenerateMeetingSummary(ctx context.Context, req MeetingSummaryRequest) (*MeetingSummaryResponse, error)
}

type service struct {
	client     Client
	projectSvc project.Service
	taskSvc    task.Service
	reportSvc  report.Service
}

func NewService(client Client, projectSvc project.Service, taskSvc task.Service, reportSvc report.Service) Service {
	return &service{client: client, projectSvc: projectSvc, taskSvc: taskSvc, reportSvc: reportSvc}
}

func (s *service) requireMembership(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) error {
	isMember, _, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, projectID)
	if err != nil {
		return apperrors.Internal("failed to check project membership")
	}
	if !isMember {
		return apperrors.Forbidden("not a member of this project")
	}
	return nil
}

const taskAssistantSystemPrompt = "You are the AI assistant embedded in Taskworked, a team task management platform. " +
	"You help break down work, estimate effort, and analyze project data. Be concrete and concise."

func (s *service) GenerateTasks(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req GenerateTasksRequest) (*GenerateTasksResponse, error) {
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid project_id")
	}
	if err := s.requireMembership(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	userPrompt := fmt.Sprintf(
		"Break the following goal down into 3 to 8 concrete, actionable tasks for a project team. "+
			"Each task needs a short title, a one-paragraph description, a priority (critical, high, medium, or low), "+
			"and a realistic estimate in hours.\n\nGoal: %s", req.Prompt)

	var result GenerateTasksResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, generateTasksSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) SummarizeDailyTasks(ctx context.Context, actorID uuid.UUID) (*SummaryResponse, error) {
	tasks, err := s.taskSvc.ListActiveByAssignee(ctx, actorID)
	if err != nil {
		return nil, apperrors.Internal("failed to load active tasks")
	}
	if len(tasks) == 0 {
		return &SummaryResponse{Summary: "You have no active tasks assigned right now — a good time to pick up new work."}, nil
	}

	var b strings.Builder
	for _, t := range tasks {
		due := "no due date"
		if t.DueDate != nil {
			due = *t.DueDate
		}
		fmt.Fprintf(&b, "- [%s] %s (status: %s, due: %s)\n", t.Priority, t.Title, t.Status, due)
	}

	userPrompt := "Write a short, friendly daily digest (2-4 sentences) summarizing this person's active tasks for today, " +
		"calling out anything overdue or urgent first:\n\n" + b.String()

	var result SummaryResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, summarySchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) SummarizeWeeklyReport(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*SummaryResponse, error) {
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	period, err := s.reportSvc.GetPeriodReport(ctx, actorID, actorRole, projectID, from, to)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Tasks created: %d, completed: %d (completion rate %.0f%%), overdue: %d, near due: %d.\n",
		period.Summary.CreatedInPeriod, period.Summary.CompletedInPeriod, period.Summary.CompletionRate*100,
		period.Summary.OverdueCount, period.Summary.NearDueCount)
	for _, p := range period.Performance {
		fmt.Fprintf(&b, "- Assignee %s: %d completed, %d on time, %d late\n", p.AssigneeID, p.CompletedCount, p.OnTimeCount, p.LateCount)
	}

	userPrompt := "Write a short weekly status report (one paragraph) for a project stakeholder based on this data:\n\n" + b.String()

	var result SummaryResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, summarySchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) EstimateDuration(ctx context.Context, req EstimateRequest) (*EstimateResponse, error) {
	userPrompt := fmt.Sprintf("Estimate how many hours this task should take, and briefly explain why.\n\nTitle: %s\nDescription: %s", req.Title, req.Description)

	var result EstimateResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, estimateSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) SuggestPriority(ctx context.Context, req PriorityRequest) (*PriorityResponse, error) {
	userPrompt := fmt.Sprintf(
		"Suggest a priority (critical, high, medium, or low) for this task, and briefly explain why.\n\nTitle: %s\nDescription: %s",
		req.Title, req.Description)

	var result PriorityResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, prioritySchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) SuggestAssignee(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req SuggestAssigneeRequest) (*AssigneeSuggestion, error) {
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid project_id")
	}
	if err := s.requireMembership(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	members, err := s.projectSvc.ListMembers(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, apperrors.BadRequest("project has no members to assign")
	}

	var b strings.Builder
	for _, m := range members {
		workload := int64(0)
		if id, parseErr := uuid.Parse(m.UserID); parseErr == nil {
			workload, _ = s.taskSvc.GetWorkload(ctx, id)
		}
		fmt.Fprintf(&b, "- id: %s, name: %s, current active tasks: %d\n", m.UserID, m.Name, workload)
	}

	userPrompt := fmt.Sprintf(
		"Pick the best project member to assign this task to, favoring members with lower current workload when skills are unclear. "+
			"Return exactly one of the given member IDs as assignee_id.\n\nTask title: %s\nDescription: %s\n\nMembers:\n%s",
		req.Title, req.Description, b.String())

	var result AssigneeSuggestion
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, suggestAssigneeSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) PredictLateTasks(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*PredictLateTasksResponse, error) {
	if err := s.requireMembership(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	tasks, err := s.taskSvc.ListAllResponses(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	count := 0
	for _, t := range tasks {
		if t.Status == task.StatusDone || t.DueDate == nil {
			continue
		}
		estimate := "unknown"
		if t.EstimateHours != nil {
			estimate = fmt.Sprintf("%.1fh", *t.EstimateHours)
		}
		fmt.Fprintf(&b, "- id: %s, title: %s, status: %s, due: %s, estimate: %s\n", t.ID, t.Title, t.Status, *t.DueDate, estimate)
		count++
	}
	if count == 0 {
		return &PredictLateTasksResponse{AtRisk: []LateTaskRisk{}}, nil
	}

	userPrompt := "Given these open tasks with their due dates, statuses, and estimates, identify which are at risk of finishing late " +
		"(overdue, due very soon relative to remaining estimated effort, or blocked). For each, give a one-sentence reason. " +
		"Only include tasks genuinely at risk — leave healthy tasks out.\n\n" + b.String()

	var result PredictLateTasksResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, predictLateTasksSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) AnalyzeTeamProductivity(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*SummaryResponse, error) {
	to := time.Now()
	from := to.AddDate(0, 0, -30)
	period, err := s.reportSvc.GetPeriodReport(ctx, actorID, actorRole, projectID, from, to)
	if err != nil {
		return nil, err
	}
	if len(period.Performance) == 0 {
		return &SummaryResponse{Summary: "No completed tasks in the last 30 days to analyze yet."}, nil
	}

	var b strings.Builder
	for _, p := range period.Performance {
		fmt.Fprintf(&b, "- Assignee %s: %d completed, %d on time, %d late (on-time rate %.0f%%)\n",
			p.AssigneeID, p.CompletedCount, p.OnTimeCount, p.LateCount, p.OnTimeRate*100)
	}

	userPrompt := "Analyze this team's productivity over the last 30 days based on per-assignee completion data. " +
		"Highlight standout performers and any areas of concern, in a short paragraph:\n\n" + b.String()

	var result SummaryResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, summarySchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) GenerateMeetingSummary(ctx context.Context, req MeetingSummaryRequest) (*MeetingSummaryResponse, error) {
	userPrompt := "Summarize these meeting notes into a short paragraph and a list of clear action items:\n\n" + req.Notes

	var result MeetingSummaryResponse
	if err := s.client.CompleteJSON(ctx, taskAssistantSystemPrompt, userPrompt, meetingSummarySchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
