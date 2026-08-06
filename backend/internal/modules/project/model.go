package project

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPlanning  Status = "planning"
	StatusActive    Status = "active"
	StatusOnHold    Status = "on_hold"
	StatusCompleted Status = "completed"
	StatusArchived  Status = "archived"
)

type Project struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"not null"`
	Description string
	OwnerID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Status      Status    `gorm:"type:varchar(20);not null;default:'planning'"`
	DueDate     *time.Time
	Color       string
	Icon        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MemberRole is project-scoped and distinct from the org-level Role in the
// auth module (e.g. an Employee can still be a project Owner).
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleMember MemberRole = "member"
)

type Member struct {
	ProjectID uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Role      MemberRole `gorm:"type:varchar(20);not null;default:'member'"`
	CreatedAt time.Time
}
