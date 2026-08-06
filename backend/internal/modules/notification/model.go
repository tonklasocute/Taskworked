// Package notification fans out events to a user through up to three
// channels — in-app (persisted + WebSocket), email, and LINE Notify —
// gated by that user's own Preference. It composes auth (who to notify),
// task (what to notify them about, for the scheduled digests), and
// realtime (how to push it live); task depends back on this package only
// through a small local interface (task.Notifier) to avoid an import
// cycle — see cmd/api/main.go's taskNotifierHolder for how that's wired.
package notification

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeAssignment    Type = "assignment"
	TypeDailyDigest   Type = "daily_digest"
	TypeEODDigest     Type = "eod_digest"
	TypeWeeklyDigest  Type = "weekly_digest"
	TypeMonthlyDigest Type = "monthly_digest"
)

type Notification struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Type      Type      `gorm:"type:varchar(30);not null"`
	Title     string    `gorm:"not null"`
	Body      string
	Link      string
	Read      bool `gorm:"not null;default:false"`
	CreatedAt time.Time
}

// Preference is one row per user, created lazily (defaults apply until a
// user sets their own). LineNotifyToken is the user's own personal LINE
// Notify token (from notify-bot.line.me) — there's no org-wide LINE
// account that can message arbitrary users, each person connects their own.
type Preference struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	EmailEnabled    bool      `gorm:"not null;default:true"`
	LineEnabled     bool      `gorm:"not null;default:false"`
	LineNotifyToken string
}
