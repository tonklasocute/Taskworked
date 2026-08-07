package ai

// -- Generate Tasks --

type GenerateTasksRequest struct {
	ProjectID string `json:"project_id" validate:"required,uuid"`
	Prompt    string `json:"prompt" validate:"required,min=3,max=2000"`
}

type TaskSuggestion struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Priority      string  `json:"priority"`
	EstimateHours float64 `json:"estimate_hours"`
}

type GenerateTasksResponse struct {
	Tasks []TaskSuggestion `json:"tasks"`
}

var generateTasksSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"tasks": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":          map[string]any{"type": "string"},
					"description":    map[string]any{"type": "string"},
					"priority":       map[string]any{"type": "string", "enum": []string{"critical", "high", "medium", "low"}},
					"estimate_hours": map[string]any{"type": "number"},
				},
				"required":             []string{"title", "description", "priority", "estimate_hours"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"tasks"},
	"additionalProperties": false,
}

// -- Summaries --

type SummaryResponse struct {
	Summary string `json:"summary"`
}

var summarySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{"type": "string"},
	},
	"required":             []string{"summary"},
	"additionalProperties": false,
}

// -- Estimate Duration --

type EstimateRequest struct {
	Title       string `json:"title" validate:"required,min=2,max=200"`
	Description string `json:"description" validate:"max=5000"`
}

type EstimateResponse struct {
	EstimateHours float64 `json:"estimate_hours"`
	Reasoning     string  `json:"reasoning"`
}

var estimateSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"estimate_hours": map[string]any{"type": "number"},
		"reasoning":      map[string]any{"type": "string"},
	},
	"required":             []string{"estimate_hours", "reasoning"},
	"additionalProperties": false,
}

// -- Suggest Priority --

type PriorityRequest struct {
	Title       string `json:"title" validate:"required,min=2,max=200"`
	Description string `json:"description" validate:"max=5000"`
}

type PriorityResponse struct {
	Priority  string `json:"priority"`
	Reasoning string `json:"reasoning"`
}

var prioritySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"priority":  map[string]any{"type": "string", "enum": []string{"critical", "high", "medium", "low"}},
		"reasoning": map[string]any{"type": "string"},
	},
	"required":             []string{"priority", "reasoning"},
	"additionalProperties": false,
}

// -- Suggest Assignee --

type SuggestAssigneeRequest struct {
	ProjectID   string `json:"project_id" validate:"required,uuid"`
	Title       string `json:"title" validate:"required,min=2,max=200"`
	Description string `json:"description" validate:"max=5000"`
}

type AssigneeSuggestion struct {
	AssigneeID   string `json:"assignee_id"`
	AssigneeName string `json:"assignee_name"`
	Reasoning    string `json:"reasoning"`
}

var suggestAssigneeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"assignee_id":   map[string]any{"type": "string"},
		"assignee_name": map[string]any{"type": "string"},
		"reasoning":     map[string]any{"type": "string"},
	},
	"required":             []string{"assignee_id", "assignee_name", "reasoning"},
	"additionalProperties": false,
}

// -- Predict Late Tasks --

type LateTaskRisk struct {
	TaskID    string `json:"task_id"`
	Title     string `json:"title"`
	Reasoning string `json:"reasoning"`
}

type PredictLateTasksResponse struct {
	AtRisk []LateTaskRisk `json:"at_risk"`
}

var predictLateTasksSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"at_risk": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id":   map[string]any{"type": "string"},
					"title":     map[string]any{"type": "string"},
					"reasoning": map[string]any{"type": "string"},
				},
				"required":             []string{"task_id", "title", "reasoning"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"at_risk"},
	"additionalProperties": false,
}

// -- Meeting Summary --

type MeetingSummaryRequest struct {
	Notes string `json:"notes" validate:"required,min=10,max=20000"`
}

type MeetingSummaryResponse struct {
	Summary     string   `json:"summary"`
	ActionItems []string `json:"action_items"`
}

var meetingSummarySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary":      map[string]any{"type": "string"},
		"action_items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required":             []string{"summary", "action_items"},
	"additionalProperties": false,
}
