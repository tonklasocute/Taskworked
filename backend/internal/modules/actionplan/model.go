// Package actionplan owns the Goal -> Milestone hierarchy that sits above
// Tasks (Task -> Subtask is already covered by task.Task's ParentTaskID).
// A task opts into this hierarchy via its own MilestoneID; tasks with no
// MilestoneID are unaffected and still appear in every other view.
package actionplan

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusNotStarted Status = "not_started"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

type Goal struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null"`
	Description string
	Status      Status `gorm:"type:varchar(20);not null;default:'not_started'"`
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Milestone struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	GoalID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null"`
	Description string
	Status      Status `gorm:"type:varchar(20);not null;default:'not_started'"`
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
