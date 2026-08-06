// Package team owns Department and composes the org-wide member
// directory (user + department + online status + workload) from auth,
// task, and presence — it doesn't own users or tasks itself.
package team

import (
	"time"

	"github.com/google/uuid"
)

type Department struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"not null;uniqueIndex"`
	CreatedAt time.Time
}
