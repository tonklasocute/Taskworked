// Package report computes read-only analytics over a project's tasks —
// no new persisted entities, everything here is derived on request from
// task.Service's data. "Weekly Report" and "Monthly Report" from the spec
// are just this same period report with different from/to presets chosen
// by the frontend, not separate code paths.
package report

import "github.com/khomkrittk/taskworked/backend/internal/modules/task"

type Summary struct {
	TotalTasks        int     `json:"total_tasks"`
	CreatedInPeriod   int     `json:"created_in_period"`
	CompletedInPeriod int     `json:"completed_in_period"`
	CompletionRate    float64 `json:"completion_rate"` // completed_in_period / created_in_period
	OverdueCount      int     `json:"overdue_count"`   // as of now, not period-scoped
	NearDueCount      int     `json:"near_due_count"`  // due within the next 3 days, as of now
}

// AssigneePerformance covers tasks completed within the period, grouped by
// assignee — "on time" means completed on or before its due date.
type AssigneePerformance struct {
	AssigneeID     string  `json:"assignee_id"`
	CompletedCount int     `json:"completed_count"`
	OnTimeCount    int     `json:"on_time_count"`
	LateCount      int     `json:"late_count"`
	OnTimeRate     float64 `json:"on_time_rate"`
}

type ProductivityPoint struct {
	Date           string `json:"date"`
	CompletedCount int    `json:"completed_count"`
}

type PeriodReport struct {
	From         string                `json:"from"`
	To           string                `json:"to"`
	Summary      Summary               `json:"summary"`
	LateTasks    []task.Response       `json:"late_tasks"`
	Performance  []AssigneePerformance `json:"performance"`
	Productivity []ProductivityPoint   `json:"productivity"`
}
