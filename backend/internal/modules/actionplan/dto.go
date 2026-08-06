package actionplan

import (
	"time"

	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
)

type CreateGoalRequest struct {
	ProjectID   string  `json:"project_id" validate:"required,uuid"`
	Name        string  `json:"name" validate:"required,min=2,max=200"`
	Description string  `json:"description" validate:"max=2000"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
}

type UpdateGoalRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	Status      *Status `json:"status" validate:"omitempty,oneof=not_started in_progress completed"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
}

type GoalResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	DueDate     *string   `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateMilestoneRequest struct {
	GoalID      string  `json:"goal_id" validate:"required,uuid"`
	Name        string  `json:"name" validate:"required,min=2,max=200"`
	Description string  `json:"description" validate:"max=2000"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
}

type UpdateMilestoneRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	Status      *Status `json:"status" validate:"omitempty,oneof=not_started in_progress completed"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
}

type MilestoneResponse struct {
	ID          string    `json:"id"`
	GoalID      string    `json:"goal_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	DueDate     *string   `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskNode nests a task's own subtasks (one level, matching the spec's
// Goal -> Milestone -> Task -> Subtask hierarchy — subtasks don't nest
// further).
type TaskNode struct {
	task.Response
	Subtasks []task.Response `json:"subtasks"`
}

type MilestoneNode struct {
	MilestoneResponse
	Tasks []TaskNode `json:"tasks"`
}

type GoalNode struct {
	GoalResponse
	Milestones []MilestoneNode `json:"milestones"`
}

type ActionPlanView struct {
	Goals []GoalNode `json:"goals"`
}

func toGoalResponse(g *Goal) GoalResponse {
	return GoalResponse{
		ID:          g.ID.String(),
		ProjectID:   g.ProjectID.String(),
		Name:        g.Name,
		Description: g.Description,
		Status:      g.Status,
		DueDate:     formatDate(g.DueDate),
		CreatedAt:   g.CreatedAt,
	}
}

func toMilestoneResponse(m *Milestone) MilestoneResponse {
	return MilestoneResponse{
		ID:          m.ID.String(),
		GoalID:      m.GoalID.String(),
		Name:        m.Name,
		Description: m.Description,
		Status:      m.Status,
		DueDate:     formatDate(m.DueDate),
		CreatedAt:   m.CreatedAt,
	}
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}
