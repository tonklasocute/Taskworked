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
	// MilestoneID is optional and opt-in: it places this task under a
	// Milestone in the Action Plan hierarchy (Goal -> Milestone -> Task ->
	// Subtask, owned by the actionplan module). Not to be confused with
	// the Gantt view's "milestone" rendering, which is just a zero-duration
	// task (start_date == due_date) drawn as a diamond — that's a display
	// convention, not this field. A task with MilestoneID unset is a
	// normal, unplanned task and still shows up in every other view.
	MilestoneID *uuid.UUID `gorm:"type:uuid;index"`
	// CompletedAt is set automatically the moment Status transitions to
	// StatusDone, and cleared if it's later moved away from Done (see
	// service.Update). Reports rely on this instead of UpdatedAt, which
	// changes on every edit and wouldn't reliably mean "finished".
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

type Comment struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID    uuid.UUID `gorm:"type:uuid;not null;index"`
	AuthorID  uuid.UUID `gorm:"type:uuid;not null"`
	Body      string    `gorm:"not null"`
	CreatedAt time.Time
}

// Attachment stores metadata only — the file bytes live in the object
// store (MinIO) under ObjectKey. Deleting an Attachment row without also
// removing the object would leak storage; the service deletes both
// together (see service.DeleteAttachment).
type Attachment struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID      uuid.UUID `gorm:"type:uuid;not null;index"`
	UploaderID  uuid.UUID `gorm:"type:uuid;not null"`
	FileName    string    `gorm:"not null"`
	ContentType string    `gorm:"not null"`
	SizeBytes   int64     `gorm:"not null"`
	ObjectKey   string    `gorm:"not null"`
	CreatedAt   time.Time
}

// Watcher is a task a user opted into notifications for, independent of
// being the assignee or reporter. Composite key: a user watches a given
// task at most once.
type Watcher struct {
	TaskID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}
