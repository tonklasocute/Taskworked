package actionplan

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

type Service interface {
	CreateGoal(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateGoalRequest) (*GoalResponse, error)
	UpdateGoal(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateGoalRequest) (*GoalResponse, error)
	DeleteGoal(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error

	CreateMilestone(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateMilestoneRequest) (*MilestoneResponse, error)
	UpdateMilestone(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateMilestoneRequest) (*MilestoneResponse, error)
	DeleteMilestone(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error

	GetActionPlanView(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*ActionPlanView, error)
}

type service struct {
	repo       Repository
	projectSvc project.Service
	taskSvc    task.Service
}

func NewService(repo Repository, projectSvc project.Service, taskSvc task.Service) Service {
	return &service{repo: repo, projectSvc: projectSvc, taskSvc: taskSvc}
}

func (s *service) CreateGoal(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateGoalRequest) (*GoalResponse, error) {
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid project_id")
	}
	if err := s.requireManage(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}

	dueDate, err := parseDate(req.DueDate)
	if err != nil {
		return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
	}

	g := &Goal{ProjectID: projectID, Name: req.Name, Description: req.Description, Status: StatusNotStarted, DueDate: dueDate}
	if err := s.repo.CreateGoal(ctx, g); err != nil {
		return nil, apperrors.Internal("failed to create goal")
	}

	resp := toGoalResponse(g)
	return &resp, nil
}

func (s *service) UpdateGoal(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateGoalRequest) (*GoalResponse, error) {
	g, err := s.repo.FindGoal(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err, "goal")
	}
	if err := s.requireManage(ctx, actorID, actorRole, g.ProjectID); err != nil {
		return nil, err
	}

	if req.Name != nil {
		g.Name = *req.Name
	}
	if req.Description != nil {
		g.Description = *req.Description
	}
	if req.Status != nil {
		g.Status = *req.Status
	}
	if req.DueDate != nil {
		dueDate, err := parseDate(req.DueDate)
		if err != nil {
			return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
		}
		g.DueDate = dueDate
	}

	if err := s.repo.UpdateGoal(ctx, g); err != nil {
		return nil, apperrors.Internal("failed to update goal")
	}

	resp := toGoalResponse(g)
	return &resp, nil
}

func (s *service) DeleteGoal(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error {
	g, err := s.repo.FindGoal(ctx, id)
	if err != nil {
		return notFoundOrInternal(err, "goal")
	}
	if err := s.requireManage(ctx, actorID, actorRole, g.ProjectID); err != nil {
		return err
	}

	if err := s.repo.DeleteGoal(ctx, id); err != nil {
		return apperrors.Internal("failed to delete goal")
	}
	return nil
}

func (s *service) CreateMilestone(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, req CreateMilestoneRequest) (*MilestoneResponse, error) {
	goalID, err := uuid.Parse(req.GoalID)
	if err != nil {
		return nil, apperrors.BadRequest("invalid goal_id")
	}
	goal, err := s.repo.FindGoal(ctx, goalID)
	if err != nil {
		return nil, apperrors.BadRequest("goal not found")
	}
	if err := s.requireManage(ctx, actorID, actorRole, goal.ProjectID); err != nil {
		return nil, err
	}

	dueDate, err := parseDate(req.DueDate)
	if err != nil {
		return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
	}

	m := &Milestone{GoalID: goalID, Name: req.Name, Description: req.Description, Status: StatusNotStarted, DueDate: dueDate}
	if err := s.repo.CreateMilestone(ctx, m); err != nil {
		return nil, apperrors.Internal("failed to create milestone")
	}

	resp := toMilestoneResponse(m)
	return &resp, nil
}

func (s *service) UpdateMilestone(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID, req UpdateMilestoneRequest) (*MilestoneResponse, error) {
	m, err := s.repo.FindMilestone(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err, "milestone")
	}
	goal, err := s.repo.FindGoal(ctx, m.GoalID)
	if err != nil {
		return nil, notFoundOrInternal(err, "goal")
	}
	if err := s.requireManage(ctx, actorID, actorRole, goal.ProjectID); err != nil {
		return nil, err
	}

	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Description != nil {
		m.Description = *req.Description
	}
	if req.Status != nil {
		m.Status = *req.Status
	}
	if req.DueDate != nil {
		dueDate, err := parseDate(req.DueDate)
		if err != nil {
			return nil, apperrors.BadRequest("invalid due_date, expected YYYY-MM-DD")
		}
		m.DueDate = dueDate
	}

	if err := s.repo.UpdateMilestone(ctx, m); err != nil {
		return nil, apperrors.Internal("failed to update milestone")
	}

	resp := toMilestoneResponse(m)
	return &resp, nil
}

func (s *service) DeleteMilestone(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, id uuid.UUID) error {
	m, err := s.repo.FindMilestone(ctx, id)
	if err != nil {
		return notFoundOrInternal(err, "milestone")
	}
	goal, err := s.repo.FindGoal(ctx, m.GoalID)
	if err != nil {
		return notFoundOrInternal(err, "goal")
	}
	if err := s.requireManage(ctx, actorID, actorRole, goal.ProjectID); err != nil {
		return err
	}

	if err := s.repo.DeleteMilestone(ctx, id); err != nil {
		return apperrors.Internal("failed to delete milestone")
	}
	return nil
}

func (s *service) GetActionPlanView(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) (*ActionPlanView, error) {
	isMember, _, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to check project membership")
	}
	if !isMember {
		return nil, apperrors.Forbidden("not a member of this project")
	}

	goals, err := s.repo.ListGoalsForProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to load goals")
	}
	milestones, err := s.repo.ListMilestonesForProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to load milestones")
	}
	tasks, err := s.taskSvc.ListAllResponses(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, err
	}

	milestonesByGoal := make(map[uuid.UUID][]Milestone, len(goals))
	for _, m := range milestones {
		milestonesByGoal[m.GoalID] = append(milestonesByGoal[m.GoalID], m)
	}

	tasksByMilestone := make(map[string][]task.Response)
	subtasksByParent := make(map[string][]task.Response)
	for _, t := range tasks {
		if t.ParentTaskID != nil {
			subtasksByParent[*t.ParentTaskID] = append(subtasksByParent[*t.ParentTaskID], t)
			continue
		}
		if t.MilestoneID != nil {
			tasksByMilestone[*t.MilestoneID] = append(tasksByMilestone[*t.MilestoneID], t)
		}
	}

	view := ActionPlanView{Goals: make([]GoalNode, len(goals))}
	for gi, g := range goals {
		goalMilestones := milestonesByGoal[g.ID]
		milestoneNodes := make([]MilestoneNode, len(goalMilestones))
		for mi, m := range goalMilestones {
			milestoneTasks := tasksByMilestone[m.ID.String()]
			taskNodes := make([]TaskNode, len(milestoneTasks))
			for ti, t := range milestoneTasks {
				taskNodes[ti] = TaskNode{Response: t, Subtasks: subtasksByParent[t.ID]}
			}
			milestoneNodes[mi] = MilestoneNode{MilestoneResponse: toMilestoneResponse(&m), Tasks: taskNodes}
		}
		view.Goals[gi] = GoalNode{GoalResponse: toGoalResponse(&g), Milestones: milestoneNodes}
	}

	return &view, nil
}

// requireManage allows an org admin/super admin or the project's own
// owner-role member to manage the Goal/Milestone structure — mirrors
// project.Service's own update/delete permission, since Goals are a
// project-structural artifact like the project itself, not a
// create-freely artifact like tasks.
func (s *service) requireManage(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) error {
	_, isManager, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, projectID)
	if err != nil {
		return apperrors.Internal("failed to check permissions")
	}
	if !isManager {
		return apperrors.Forbidden("only the project owner or an admin can manage the action plan")
	}
	return nil
}

func notFoundOrInternal(err error, what string) error {
	if errors.Is(err, ErrNotFound) {
		return apperrors.NotFound(what + " not found")
	}
	return apperrors.Internal("failed to look up " + what)
}

func parseDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
