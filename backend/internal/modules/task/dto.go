package task

import "time"

type CreateRequest struct {
	ProjectID     string   `json:"project_id" validate:"required,uuid"`
	ParentTaskID  *string  `json:"parent_task_id" validate:"omitempty,uuid"`
	Title         string   `json:"title" validate:"required,min=2,max=200"`
	Description   string   `json:"description" validate:"max=5000"`
	Priority      Priority `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	DueDate       *string  `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	EstimateHours *float64 `json:"estimate_hours" validate:"omitempty,gte=0"`
	AssigneeID    *string  `json:"assignee_id" validate:"omitempty,uuid"`
	Tags          []string `json:"tags" validate:"omitempty,dive,min=1,max=30"`
}

type UpdateRequest struct {
	Title         *string   `json:"title" validate:"omitempty,min=2,max=200"`
	Description   *string   `json:"description" validate:"omitempty,max=5000"`
	Priority      *Priority `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	Status        *Status   `json:"status" validate:"omitempty,oneof=backlog todo doing review testing done blocked"`
	DueDate       *string   `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	EstimateHours *float64  `json:"estimate_hours" validate:"omitempty,gte=0"`
	AssigneeID    *string   `json:"assignee_id" validate:"omitempty,uuid"`
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
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	ParentTaskID  *string    `json:"parent_task_id,omitempty"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Priority      Priority   `json:"priority"`
	Status        Status     `json:"status"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	EstimateHours *float64   `json:"estimate_hours,omitempty"`
	AssigneeID    *string    `json:"assignee_id,omitempty"`
	ReporterID    string     `json:"reporter_id"`
	Tags          []string   `json:"tags"`
	CreatedAt     time.Time  `json:"created_at"`
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
		DueDate:       t.DueDate,
		EstimateHours: t.EstimateHours,
		AssigneeID:    assigneeID,
		ReporterID:    t.ReporterID.String(),
		Tags:          tags,
		CreatedAt:     t.CreatedAt,
	}
}

func toChecklistResponse(c *ChecklistItem) ChecklistItemResponse {
	return ChecklistItemResponse{ID: c.ID.String(), Text: c.Text, Done: c.Done, Position: c.Position}
}
