package task

import "time"

type CreateRequest struct {
	ProjectID     string   `json:"project_id" validate:"required,uuid"`
	ParentTaskID  *string  `json:"parent_task_id" validate:"omitempty,uuid"`
	Title         string   `json:"title" validate:"required,min=2,max=200"`
	Description   string   `json:"description" validate:"max=5000"`
	Priority      Priority `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	StartDate     *string  `json:"start_date" validate:"omitempty,datetime=2006-01-02"`
	DueDate       *string  `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	EstimateHours *float64 `json:"estimate_hours" validate:"omitempty,gte=0"`
	AssigneeID    *string  `json:"assignee_id" validate:"omitempty,uuid"`
	MilestoneID   *string  `json:"milestone_id" validate:"omitempty,uuid"`
	Tags          []string `json:"tags" validate:"omitempty,dive,min=1,max=30"`
}

type UpdateRequest struct {
	Title         *string   `json:"title" validate:"omitempty,min=2,max=200"`
	Description   *string   `json:"description" validate:"omitempty,max=5000"`
	Priority      *Priority `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	Status        *Status   `json:"status" validate:"omitempty,oneof=backlog todo doing review testing done blocked"`
	StartDate     *string   `json:"start_date" validate:"omitempty,datetime=2006-01-02"`
	DueDate       *string   `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	EstimateHours *float64  `json:"estimate_hours" validate:"omitempty,gte=0"`
	AssigneeID    *string   `json:"assignee_id" validate:"omitempty,uuid"`
	MilestoneID   *string   `json:"milestone_id" validate:"omitempty,uuid"`
}

type AddChecklistItemRequest struct {
	Text string `json:"text" validate:"required,min=1,max=300"`
}

type UpdateChecklistItemRequest struct {
	Text *string `json:"text" validate:"omitempty,min=1,max=300"`
	Done *bool   `json:"done"`
}

type SetTagsRequest struct {
	Tags []string `json:"tags" validate:"required,dive,min=1,max=30"`
}

type AddDependencyRequest struct {
	DependsOnTaskID string `json:"depends_on_task_id" validate:"required,uuid"`
}

type DependencyResponse struct {
	TaskID          string `json:"task_id"`
	DependsOnTaskID string `json:"depends_on_task_id"`
}

// GanttResponse is a read-model combining a project's tasks, their
// dependency edges, and the computed critical path — everything the Gantt
// view needs for one render.
type GanttResponse struct {
	Tasks        []Response           `json:"tasks"`
	Dependencies []DependencyResponse `json:"dependencies"`
	CriticalPath []string             `json:"critical_path"` // task IDs, in path order
	ProjectDays  int                  `json:"project_days"`  // total span of the critical path, in days
}

type ListFilter struct {
	ProjectID  string
	Status     Status
	Priority   Priority
	AssigneeID string
	Search     string
	Page       int
	PageSize   int
}

type Response struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"project_id"`
	ParentTaskID *string  `json:"parent_task_id,omitempty"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Priority     Priority `json:"priority"`
	Status       Status   `json:"status"`
	// StartDate/DueDate are calendar dates (day granularity, as enforced
	// by the "YYYY-MM-DD" request format), so they're serialized as plain
	// date strings rather than full RFC3339 timestamps — a timestamp would
	// carry a spurious time-of-day and timezone offset that date-only
	// clients (the Calendar view) would have to strip back out.
	StartDate     *string   `json:"start_date,omitempty"`
	DueDate       *string   `json:"due_date,omitempty"`
	CompletedAt   *string   `json:"completed_at,omitempty"`
	EstimateHours *float64  `json:"estimate_hours,omitempty"`
	AssigneeID    *string   `json:"assignee_id,omitempty"`
	MilestoneID   *string   `json:"milestone_id,omitempty"`
	ReporterID    string    `json:"reporter_id"`
	Tags          []string  `json:"tags"`
	CreatedAt     time.Time `json:"created_at"`
}

type ChecklistItemResponse struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Position int    `json:"position"`
}

type ListResponse struct {
	Items    []Response `json:"items"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

func toResponse(t *Task, tags []string) Response {
	var parentID *string
	if t.ParentTaskID != nil {
		s := t.ParentTaskID.String()
		parentID = &s
	}
	var assigneeID *string
	if t.AssigneeID != nil {
		s := t.AssigneeID.String()
		assigneeID = &s
	}
	var milestoneID *string
	if t.MilestoneID != nil {
		s := t.MilestoneID.String()
		milestoneID = &s
	}
	if tags == nil {
		tags = []string{}
	}

	return Response{
		ID:            t.ID.String(),
		ProjectID:     t.ProjectID.String(),
		ParentTaskID:  parentID,
		Title:         t.Title,
		Description:   t.Description,
		Priority:      t.Priority,
		Status:        t.Status,
		StartDate:     formatDate(t.StartDate),
		DueDate:       formatDate(t.DueDate),
		CompletedAt:   formatDate(t.CompletedAt),
		EstimateHours: t.EstimateHours,
		AssigneeID:    assigneeID,
		MilestoneID:   milestoneID,
		ReporterID:    t.ReporterID.String(),
		Tags:          tags,
		CreatedAt:     t.CreatedAt,
	}
}

func toChecklistResponse(c *ChecklistItem) ChecklistItemResponse {
	return ChecklistItemResponse{ID: c.ID.String(), Text: c.Text, Done: c.Done, Position: c.Position}
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}
