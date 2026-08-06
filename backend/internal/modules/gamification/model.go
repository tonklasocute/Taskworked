// Package gamification reacts to task completions (via task.Gamifier,
// called through the same late-binding holder pattern as
// notification.Service — see cmd/api/main.go's taskGamifierHolder) to
// award EXP, level up, extend streaks, unlock badges, and progress
// missions. Scope notes on the spec's terms:
//   - "Achievements" and "Badge" are the same mechanic here — one system,
//     not two parallel ones.
//   - "Quest" / "Daily/Weekly/Monthly Mission" are a fixed, non-configurable
//     rule set computed from existing task data (MissionProgress just
//     tracks whether this period's reward was already granted), not a
//     full quest-authoring engine.
//   - "Team Ranking" = department ranking — this app has no separate
//     "Team" entity beyond Department.
//   - "Evolution Character" is a display tier computed from Level (see
//     dto.go's levelTitle), not a separate asset/entity system.
package gamification

import (
	"time"

	"github.com/google/uuid"
)

// Character is a user's gamification profile — created lazily on first
// task completion.
type Character struct {
	UserID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	EXP                int       `gorm:"not null;default:0"`
	Level              int       `gorm:"not null;default:1"`
	TotalCompleted     int       `gorm:"not null;default:0"`
	CurrentStreak      int       `gorm:"not null;default:0"`
	LongestStreak      int       `gorm:"not null;default:0"`
	LastCompletionDate *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type BadgeCode string

const (
	BadgeEarlyBird    BadgeCode = "early_bird"
	BadgeFastWorker   BadgeCode = "fast_worker"   // 10 tasks completed
	BadgeHundredTasks BadgeCode = "hundred_tasks" // 100 tasks completed
	BadgeLegend       BadgeCode = "legend"        // 500 tasks completed
	BadgePerfectWeek  BadgeCode = "perfect_week"  // 3+ on-time completions, 0 late, in trailing 7 days
	BadgeStreak7      BadgeCode = "seven_day_streak"
)

type Badge struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code      BadgeCode `gorm:"type:varchar(30);primaryKey"`
	AwardedAt time.Time
}

type MissionType string

const (
	MissionDaily   MissionType = "daily"
	MissionWeekly  MissionType = "weekly"
	MissionMonthly MissionType = "monthly"
)

// MissionProgress tracks completions within one period (a calendar day,
// ISO week, or month) so the reward is granted exactly once when the
// threshold is first crossed.
type MissionProgress struct {
	UserID    uuid.UUID   `gorm:"type:uuid;primaryKey"`
	Type      MissionType `gorm:"type:varchar(10);primaryKey"`
	PeriodKey string      `gorm:"primaryKey"` // e.g. "2026-08-06", "2026-W32", "2026-08"
	Count     int         `gorm:"not null;default:0"`
	Rewarded  bool        `gorm:"not null;default:false"`
	UpdatedAt time.Time
}

// Mission thresholds and their one-time EXP reward.
const (
	DailyMissionThreshold   = 3
	DailyMissionReward      = 30
	WeeklyMissionThreshold  = 15
	WeeklyMissionReward     = 150
	MonthlyMissionThreshold = 50
	MonthlyMissionReward    = 500
)
