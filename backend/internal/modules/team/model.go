// Package team owns Department and composes the org-wide member
// directory (user + department + online status + workload) from auth,
// task, and presence — it doesn't own users or tasks itself.
package team

import (
	"time"

	"github.com/google/uuid"
)

type Department struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name string    `gorm:"not null;uniqueIndex"`
	// OrganizationID is populated for every department by migration 0002
	// (P1.1) but not yet read anywhere — see auth.User.OrganizationID's
	// comment. Note the uniqueIndex on Name above stays *global* for now;
	// scoping it to (organization_id, name) is part of the deferred
	// constraint-tightening migration, not this one.
	OrganizationID *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt      time.Time
}
