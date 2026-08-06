package task

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

type Service interface {
	Create(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateRequest) (*Response, error)
	Get(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) (*Response, error)
	List(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, filter ListFilter) (*ListResponse, error)
	Update(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateRequest) (*Response, error)
	Delete(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error

	SetTags(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req SetTagsRequest) ([]string, error)

	AddChecklistItem(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID uuid.UUID, req AddChecklistItemRequest) (*ChecklistItemResponse, error)
	UpdateChecklistItem(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID, itemID uuid.UUID, req UpdateChecklistItemRequest) (*ChecklistItemResponse, error)
	DeleteChecklistItem(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID, itemID uuid.UUID) error
	ListChecklistItems(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID uuid.UUID) ([]ChecklistItemResponse, error)

	AddDependency(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID uuid.UUID, req AddDependencyRequest) error
	RemoveDependency(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID, dependsOnID uuid.UUID) error
	GetGanttView(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*GanttResponse, error)

	// ListAllResponses returns every task in a project as Response DTOs
	// (unpaginated), membership-checked. Shared by any module that needs
	// a project's full task set — currently Gantt and Action Plan.
	ListAllResponses(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) ([]Response, error)

	// GetWorkload counts a user's active (non-Done) tasks across every
	// project. No membership check — it's a count, not task details, and
	// is intended to be visible org-wide via the Team directory.
	GetWorkload(ctx context.Context, userID uuid.UUID) (int64, error)

	// ListActiveByAssignee and ListCompletedByAssigneeSince back the
	// notification digests. Same no-membership-check rationale as
	// GetWorkload: the notification module already knows who it's
	// messaging, this just supplies the content.
	ListActiveByAssignee(ctx context.Context, userID uuid.UUID) ([]Response, error)
	ListCompletedByAssigneeSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]Response, error)
}

// EventPublisher lets the service broadcast task changes over WebSocket
// without importing the realtime package directly — *realtime.Publisher
// satisfies this structurally. A nil publisher (e.g. in unit tests) means
// broadcasting is silently skipped.
type EventPublisher interface {
	Publish(ctx context.Context, projectID string, event any) error
}

// Event is the payload broadcast to a project's WebSocket subscribers.
type Event struct {
	Type   string    `json:"type"` // "task.created" | "task.updated" | "task.deleted"
	Task   *Response `json:"task,omitempty"`
	TaskID string    `json:"task_id,omitempty"`
}

// Notifier lets the service raise a personal notification when a task is
// assigned, without importing the notification package directly (which
// would create an import cycle: notification depends on task.Service for
// digest queries). A nil notifier (e.g. in unit tests) means notifying is
// silently skipped.
type Notifier interface {
	NotifyAssignment(ctx context.Context, assigneeID uuid.UUID, taskID, taskTitle string) error
}

// Gamifier lets the service award EXP/badges/streak/mission progress when
// a task is completed, without importing the gamification package
// directly (same import-cycle reason as Notifier: gamification depends on
// task.Service for its own perfect-week badge check). A nil gamifier
// (e.g. in unit tests) means awarding is silently skipped.
type Gamifier interface {
	OnTaskCompleted(ctx context.Context, userID uuid.UUID, completedAt time.Time, dueDate *time.Time, priority Priority) error
}

type service struct {
	repo       Repository
	projectSvc project.Service
	publisher  EventPublisher
	notifier   Notifier
	gamifier   Gamifier
}

func NewService(repo Repository, projectSvc project.Service, publisher EventPublisher, notifier Notifier, gamifier Gamifier) Service {
	return &service{repo: repo, projectSvc: projectSvc, publisher: publisher, notifier: notifier, gamifier: gamifier}
}

func (s *service) publish(ctx context.Context, projectID uuid.UUID, event Event) {
	if s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, projectID.String(), event)
}

func (s *service) Create(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateRequest) (*Response, error) {
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid project_id")
	}

	isMember, _, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to check project membership")
	}
	if !isMember {
		return nil, apperrors.Forbidden("not a member of this project")
	}

	startDate, dueDate, err := parseDateRange(req.StartDate, req.DueDate)
	if err != nil {
		return nil, err
	}

	var parentTaskID *uuid.UUID
	if req.ParentTaskID != nil {
		parsed, err := uuid.Parse(*req.ParentTaskID)
		if err != nil {
			return nil, apperrors.BadRequest("invalid parent_task_id")
		}
		parent, err := s.repo.FindByID(ctx, parsed)
		if err != nil {
			return nil, apperrors.BadRequest("parent task not found")
		}
		if parent.ProjectID != projectID {
			return nil, apperrors.BadRequest("parent task must belong to the same project")
		}
		parentTaskID = &parsed
	}

	assigneeID, err := s.validatedAssignee(ctx, actorRole, projectID, req.AssigneeID)
	if err != nil {
		return nil, err
	}

	// MilestoneID is deliberately not cross-validated against the
	// actionplan module here (that would require task -> actionplan
	// import, and actionplan already depends on task for its hierarchy
	// view — a cycle). The UI only ever offers milestones scoped to the
	// current project, so a stray cross-project ID would only come from
	// direct API misuse, not normal use, and it's still gated by the same
	// edit permissions as everything else on the task.
	milestoneID, err := parseOptionalUUID(req.MilestoneID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid milestone_id")
	}

	priority := req.Priority
	if priority == "" {
		priority = PriorityMedium
	}

	t := &Task{
		ProjectID:     projectID,
		ParentTaskID:  parentTaskID,
		Title:         req.Title,
		Description:   req.Description,
		Priority:      priority,
		Status:        StatusBacklog,
		StartDate:     startDate,
		DueDate:       dueDate,
		EstimateHours: req.EstimateHours,
		MilestoneID:   milestoneID,
		AssigneeID:    assigneeID,
		ReporterID:    actorID,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, apperrors.Internal("failed to create task")
	}

	if len(req.Tags) > 0 {
		if err := s.repo.SetTags(ctx, t.ID, req.Tags); err != nil {
			return nil, apperrors.Internal("failed to set tags")
		}
	}

	resp := toResponse(t, req.Tags)
	s.publish(ctx, t.ProjectID, Event{Type: "task.created", Task: &resp})

	if t.AssigneeID != nil && s.notifier != nil {
		_ = s.notifier.NotifyAssignment(ctx, *t.AssigneeID, t.ID.String(), t.Title)
	}

	return &resp, nil
}

func (s *service) Get(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) (*Response, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}

	if err := s.requireMembership(ctx, actorID, actorRole, t.ProjectID); err != nil {
		return nil, err
	}

	tags, err := s.repo.Tags(ctx, t.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to load tags")
	}

	resp := toResponse(t, tags)
	return &resp, nil
}

func (s *service) List(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, filter ListFilter) (*ListResponse, error) {
	projectID, err := uuid.Parse(filter.ProjectID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid project_id")
	}
	if err := s.requireMembership(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	tasks, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.Internal("failed to list tasks")
	}

	items := make([]Response, len(tasks))
	for i := range tasks {
		tags, err := s.repo.Tags(ctx, tasks[i].ID)
		if err != nil {
			return nil, apperrors.Internal("failed to load tags")
		}
		items[i] = toResponse(&tasks[i], tags)
	}

	return &ListResponse{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (s *service) Update(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateRequest) (*Response, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}

	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return nil, err
	}

	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	justCompleted := false
	if req.Status != nil {
		if *req.Status == StatusDone && t.Status != StatusDone {
			now := time.Now()
			t.CompletedAt = &now
			justCompleted = true
		} else if *req.Status != StatusDone {
			t.CompletedAt = nil
		}
		t.Status = *req.Status
	}
	if req.EstimateHours != nil {
		t.EstimateHours = req.EstimateHours
	}
	if req.StartDate != nil {
		startDate, err := parseDueDate(req.StartDate)
		if err != nil {
			return nil, apperrors.BadRequest("invalid start_date, expected YYYY-MM-DD")
		}
		t.StartDate = startDate
	}
	if req.DueDate != nil {
		dueDate, err := parseDueDate(req.DueDate)
		if err != nil {
			return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
		}
		t.DueDate = dueDate
	}
	if t.StartDate != nil && t.DueDate != nil && t.StartDate.After(*t.DueDate) {
		return nil, apperrors.BadRequest("start_date must not be after due_date")
	}
	previousAssigneeID := t.AssigneeID
	if req.AssigneeID != nil {
		assigneeID, err := s.validatedAssignee(ctx, actorRole, t.ProjectID, req.AssigneeID)
		if err != nil {
			return nil, err
		}
		t.AssigneeID = assigneeID
	}
	if req.MilestoneID != nil {
		milestoneID, err := parseOptionalUUID(req.MilestoneID)
		if err != nil {
			return nil, apperrors.BadRequest("invalid milestone_id")
		}
		t.MilestoneID = milestoneID
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, apperrors.Internal("failed to update task")
	}

	tags, err := s.repo.Tags(ctx, t.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to load tags")
	}

	resp := toResponse(t, tags)
	s.publish(ctx, t.ProjectID, Event{Type: "task.updated", Task: &resp})

	assigneeChanged := t.AssigneeID != nil && (previousAssigneeID == nil || *previousAssigneeID != *t.AssigneeID)
	if assigneeChanged && s.notifier != nil {
		_ = s.notifier.NotifyAssignment(ctx, *t.AssigneeID, t.ID.String(), t.Title)
	}

	if justCompleted && s.gamifier != nil {
		// Credit whoever's assigned; if nobody is, credit the actor who
		// marked it done (they did the work either way).
		recipient := actorID
		if t.AssigneeID != nil {
			recipient = *t.AssigneeID
		}
		_ = s.gamifier.OnTaskCompleted(ctx, recipient, *t.CompletedAt, t.DueDate, t.Priority)
	}

	return &resp, nil
}

func (s *service) Delete(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return notFoundOrInternal(err)
	}

	if !auth.IsOrgAdmin(actorRole) {
		_, isManager, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, t.ProjectID)
		if err != nil {
			return apperrors.Internal("failed to check permissions")
		}
		if actorID != t.ReporterID && !isManager {
			return apperrors.Forbidden("only the reporter or a project manager can delete this task")
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return apperrors.Internal("failed to delete task")
	}
	s.publish(ctx, t.ProjectID, Event{Type: "task.deleted", TaskID: id.String()})
	return nil
}

func (s *service) SetTags(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req SetTagsRequest) ([]string, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return nil, err
	}
	if err := s.repo.SetTags(ctx, id, req.Tags); err != nil {
		return nil, apperrors.Internal("failed to set tags")
	}
	return req.Tags, nil
}

func (s *service) AddChecklistItem(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID uuid.UUID, req AddChecklistItemRequest) (*ChecklistItemResponse, error) {
	t, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return nil, err
	}

	existing, err := s.repo.ListChecklistItems(ctx, taskID)
	if err != nil {
		return nil, apperrors.Internal("failed to load checklist")
	}

	item := &ChecklistItem{TaskID: taskID, Text: req.Text, Position: len(existing)}
	if err := s.repo.AddChecklistItem(ctx, item); err != nil {
		return nil, apperrors.Internal("failed to add checklist item")
	}

	resp := toChecklistResponse(item)
	return &resp, nil
}

func (s *service) UpdateChecklistItem(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID, itemID uuid.UUID, req UpdateChecklistItemRequest) (*ChecklistItemResponse, error) {
	t, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return nil, err
	}

	item, err := s.repo.FindChecklistItem(ctx, itemID)
	if err != nil || item.TaskID != taskID {
		return nil, apperrors.NotFound("checklist item not found")
	}

	if req.Text != nil {
		item.Text = *req.Text
	}
	if req.Done != nil {
		item.Done = *req.Done
	}
	if err := s.repo.UpdateChecklistItem(ctx, item); err != nil {
		return nil, apperrors.Internal("failed to update checklist item")
	}

	resp := toChecklistResponse(item)
	return &resp, nil
}

func (s *service) DeleteChecklistItem(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID, itemID uuid.UUID) error {
	t, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return notFoundOrInternal(err)
	}
	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return err
	}

	item, err := s.repo.FindChecklistItem(ctx, itemID)
	if err != nil || item.TaskID != taskID {
		return apperrors.NotFound("checklist item not found")
	}

	if err := s.repo.DeleteChecklistItem(ctx, itemID); err != nil {
		return apperrors.Internal("failed to delete checklist item")
	}
	return nil
}

func (s *service) ListChecklistItems(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID uuid.UUID) ([]ChecklistItemResponse, error) {
	t, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if err := s.requireMembership(ctx, actorID, actorRole, t.ProjectID); err != nil {
		return nil, err
	}

	items, err := s.repo.ListChecklistItems(ctx, taskID)
	if err != nil {
		return nil, apperrors.Internal("failed to load checklist")
	}

	resp := make([]ChecklistItemResponse, len(items))
	for i := range items {
		resp[i] = toChecklistResponse(&items[i])
	}
	return resp, nil
}

func (s *service) AddDependency(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID uuid.UUID, req AddDependencyRequest) error {
	t, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return notFoundOrInternal(err)
	}
	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return err
	}

	dependsOnID, err := uuid.Parse(req.DependsOnTaskID)
	if err != nil {
		return apperrors.BadRequest("invalid depends_on_task_id")
	}
	if dependsOnID == taskID {
		return apperrors.BadRequest("a task cannot depend on itself")
	}

	dependsOn, err := s.repo.FindByID(ctx, dependsOnID)
	if err != nil {
		return apperrors.BadRequest("depends_on_task_id not found")
	}
	if dependsOn.ProjectID != t.ProjectID {
		return apperrors.BadRequest("dependencies must be within the same project")
	}

	existing, err := s.repo.ListDependenciesForProject(ctx, t.ProjectID)
	if err != nil {
		return apperrors.Internal("failed to load existing dependencies")
	}
	successors := make(map[uuid.UUID][]uuid.UUID, len(existing))
	for _, d := range existing {
		successors[d.DependsOnID] = append(successors[d.DependsOnID], d.TaskID)
	}
	if canReach(successors, taskID, dependsOnID) {
		return apperrors.BadRequest("this dependency would create a cycle")
	}

	if err := s.repo.AddDependency(ctx, &Dependency{TaskID: taskID, DependsOnID: dependsOnID}); err != nil {
		return apperrors.Conflict("dependency already exists")
	}
	return nil
}

func (s *service) RemoveDependency(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, taskID, dependsOnID uuid.UUID) error {
	t, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return notFoundOrInternal(err)
	}
	if err := s.requireCanEdit(ctx, actorID, actorRole, t); err != nil {
		return err
	}

	if err := s.repo.RemoveDependency(ctx, taskID, dependsOnID); err != nil {
		return apperrors.Internal("failed to remove dependency")
	}
	return nil
}

func (s *service) ListAllResponses(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) ([]Response, error) {
	if err := s.requireMembership(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	tasks, err := s.repo.ListAllForProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to load tasks")
	}

	responses := make([]Response, len(tasks))
	for i := range tasks {
		tags, err := s.repo.Tags(ctx, tasks[i].ID)
		if err != nil {
			return nil, apperrors.Internal("failed to load tags")
		}
		responses[i] = toResponse(&tasks[i], tags)
	}
	return responses, nil
}

func (s *service) GetWorkload(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.repo.CountActiveByAssignee(ctx, userID)
	if err != nil {
		return 0, apperrors.Internal("failed to count workload")
	}
	return count, nil
}

func (s *service) ListActiveByAssignee(ctx context.Context, userID uuid.UUID) ([]Response, error) {
	tasks, err := s.repo.ListActiveByAssignee(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to list active tasks")
	}
	responses := make([]Response, len(tasks))
	for i := range tasks {
		responses[i] = toResponse(&tasks[i], nil)
	}
	return responses, nil
}

func (s *service) ListCompletedByAssigneeSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]Response, error) {
	tasks, err := s.repo.ListCompletedByAssigneeSince(ctx, userID, since)
	if err != nil {
		return nil, apperrors.Internal("failed to list completed tasks")
	}
	responses := make([]Response, len(tasks))
	for i := range tasks {
		responses[i] = toResponse(&tasks[i], nil)
	}
	return responses, nil
}

func (s *service) GetGanttView(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*GanttResponse, error) {
	if err := s.requireMembership(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	tasks, err := s.repo.ListAllForProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to load tasks")
	}
	deps, err := s.repo.ListDependenciesForProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to load dependencies")
	}

	taskResponses := make([]Response, len(tasks))
	for i := range tasks {
		tags, err := s.repo.Tags(ctx, tasks[i].ID)
		if err != nil {
			return nil, apperrors.Internal("failed to load tags")
		}
		taskResponses[i] = toResponse(&tasks[i], tags)
	}

	depResponses := make([]DependencyResponse, len(deps))
	for i, d := range deps {
		depResponses[i] = DependencyResponse{TaskID: d.TaskID.String(), DependsOnTaskID: d.DependsOnID.String()}
	}

	criticalPath, projectDays, err := computeCriticalPath(tasks, deps)
	if err != nil {
		// A cycle here means data got into an inconsistent state outside
		// AddDependency's own cycle check (e.g. a since-removed guard) —
		// still render the board, just without a critical path.
		criticalPath, projectDays = nil, 0
	}
	criticalPathIDs := make([]string, len(criticalPath))
	for i, id := range criticalPath {
		criticalPathIDs[i] = id.String()
	}

	return &GanttResponse{
		Tasks:        taskResponses,
		Dependencies: depResponses,
		CriticalPath: criticalPathIDs,
		ProjectDays:  projectDays,
	}, nil
}

// requireMembership allows any project member (or org admin) to read.
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

// requireCanEdit allows the task's assignee, its reporter, a project
// manager (owner-role member), or an org admin to modify it.
func (s *service) requireCanEdit(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, t *Task) error {
	if auth.IsOrgAdmin(actorRole) {
		return nil
	}
	if actorID == t.ReporterID || (t.AssigneeID != nil && actorID == *t.AssigneeID) {
		return nil
	}

	_, isManager, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, t.ProjectID)
	if err != nil {
		return apperrors.Internal("failed to check permissions")
	}
	if !isManager {
		return apperrors.Forbidden("only the assignee, reporter, or a project manager can modify this task")
	}
	return nil
}

// validatedAssignee ensures a given assignee ID is actually a member of
// the project. It always checks membership as auth.RoleEmployee — the
// assignee's own org role is irrelevant here and must never trigger
// CheckMembership's org-admin bypass, or an admin assignee would be
// accepted without a real membership row.
func (s *service) validatedAssignee(ctx context.Context, _ auth.Role, projectID uuid.UUID, assigneeID *string) (*uuid.UUID, error) {
	if assigneeID == nil || *assigneeID == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*assigneeID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid assignee_id")
	}
	isMember, _, err := s.projectSvc.CheckMembership(ctx, parsed, auth.RoleEmployee, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to check assignee membership")
	}
	if !isMember {
		return nil, apperrors.BadRequest("assignee must be a member of the project")
	}
	return &parsed, nil
}

func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperrors.NotFound("task not found")
	}
	return apperrors.Internal("failed to look up task")
}

func parseDueDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseDateRange parses start/due date strings and, if both are present,
// enforces start <= due — the invariant the Calendar's drag/resize UI
// relies on to render a sane span.
func parseDateRange(startStr, dueStr *string) (start, due *time.Time, err error) {
	start, err = parseDueDate(startStr)
	if err != nil {
		return nil, nil, apperrors.BadRequest("invalid start_date, expected YYYY-MM-DD")
	}
	due, err = parseDueDate(dueStr)
	if err != nil {
		return nil, nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
	}
	if start != nil && due != nil && start.After(*due) {
		return nil, nil, apperrors.BadRequest("start_date must not be after due_date")
	}
	return start, due, nil
}

func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
