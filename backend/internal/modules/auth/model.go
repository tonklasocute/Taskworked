package auth

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleManager    Role = "manager"
	RoleLeader     Role = "leader"
	RoleEmployee   Role = "employee"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"not null"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	Role         Role      `gorm:"type:varchar(20);not null;default:'employee'"`
	AvatarURL    string
	// DepartmentID points at a team.Department. Owned here (not in the
	// team module) for the same reason task.MilestoneID lives on Task:
	// it's a value the user record carries, and only the owning module
	// needs the type it belongs to.
	DepartmentID *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
