// Package organization owns the Organization entity — the top-level
// multi-tenancy boundary introduced in P1.1. Deliberately minimal for now:
// just the model and enough repository surface for the migration's default
// organization to be readable from Go. No service, no handler, no routes —
// nothing in the API surface consumes this yet, and no other module's
// authorization checks read organization_id yet either. That lands in
// P1.2+. See docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md.
package organization

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// DefaultOrganizationID is the fixed ID migration 0002 assigns to the one
// organization every pre-existing user/department/project was backfilled
// into. Exported so later phases (and tests) can reference it without
// hardcoding the UUID string in multiple places.
var DefaultOrganizationID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Organization struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"not null"`
	Slug        string    `gorm:"not null;uniqueIndex"`
	Description string
	Logo        string
	Status      Status `gorm:"type:varchar(20);not null;default:'active'"`
	// Settings is an open-ended JSON bag (e.g. feature flags, branding)
	// rather than a growing list of dedicated columns — matches the
	// migration's `jsonb default '{}'` column.
	Settings  map[string]any `gorm:"type:jsonb;serializer:json;not null;default:'{}'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
