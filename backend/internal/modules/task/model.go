package task

import (
	"time"

	"github.com/google/uuid"
)

type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

type Status string

const (
	StatusBacklog Status = "backlog"
	StatusTodo    Status = "todo"
	StatusDoing   Status = "doing"
	StatusReview  Status = "review"
	StatusTesting Status = "testing"
	StatusDone    Status = "done"
	StatusBlocked Status = "blocked"
)

type Task struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	ParentTaskID *uuid.UUID `gorm:"type:uuid;index"`
	Title        string     `gorm:"not null"`
	Description  string
	Priority     Priority `gorm:"type:varchar(20);not null;default:'medium'"`
	Status       Status   `gorm:"type:varchar(20);not null;default:'backlog'"`
	// StartDate is optional. When set (with DueDate), the task renders as a
	// multi-day span on the Calendar; when nil, DueDate alone places it as
	// a single-day event.
	StartDate     *time.Time
	DueDate       *time.Time
	EstimateHours *float64
	AssigneeID    *uuid.UUID `gorm:"type:uuid;index"`
	ReporterID    uuid.UUID  `gorm:"type:uuid;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ChecklistItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Text      string    `gorm:"not null"`
	Done      bool      `gorm:"not null;default:false"`
	Position  int       `gorm:"not null;default:0"`
	CreatedAt time.Time
}

type Tag struct {
	TaskID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Tag    string    `gorm:"primaryKey"`
}

// Dependency is a finish-to-start dependency for the Gantt view: TaskID
// (the successor) cannot start until DependsOnID (the predecessor)
// finishes. Other dependency types (start-to-start, etc.) aren't modeled —
// FS covers the overwhelming majority of real-world usage.
type Dependency struct {
	TaskID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	DependsOnID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt   time.Time
}
