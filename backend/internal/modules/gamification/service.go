package gamification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	"github.com/khomkrittk/taskworked/backend/internal/modules/team"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// TaskService is the slice of task.Service the perfect-week badge check
// needs.
type TaskService interface {
	ListCompletedByAssigneeSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]task.Response, error)
}

// TeamService is the slice of team.Service the leaderboard needs — the
// directory already carries name + department in one call, and is
// organization-scoped to actorID's own organization (see
// team.Service.GetDirectory).
type TeamService interface {
	GetDirectory(ctx context.Context, actorID uuid.UUID) (*team.DirectoryResponse, error)
}

type Service interface {
	// OnTaskCompleted satisfies task.Gamifier structurally — task.Service
	// calls this through that local interface, never importing this
	// package directly (see the package doc comment in model.go).
	OnTaskCompleted(ctx context.Context, userID uuid.UUID, completedAt time.Time, dueDate *time.Time, priority task.Priority) error

	GetProfile(ctx context.Context, actorID uuid.UUID) (*ProfileResponse, error)
	// GetLeaderboard is scoped to actorID's own organization — see
	// GetDirectory's comment above and the implementation below for how.
	GetLeaderboard(ctx context.Context, actorID uuid.UUID) (*LeaderboardResponse, error)
}

type service struct {
	repo    Repository
	taskSvc TaskService
	teamSvc TeamService
}

func NewService(repo Repository, taskSvc TaskService, teamSvc TeamService) Service {
	return &service{repo: repo, taskSvc: taskSvc, teamSvc: teamSvc}
}

func (s *service) OnTaskCompleted(ctx context.Context, userID uuid.UUID, completedAt time.Time, dueDate *time.Time, priority task.Priority) error {
	char, err := s.repo.FindCharacter(ctx, userID)
	if err != nil {
		if err != ErrNotFound {
			return apperrors.Internal("failed to load character")
		}
		char = &Character{UserID: userID}
	}

	onTime := dueDate == nil || !completedAt.After(*dueDate)
	delta := 20
	if onTime {
		delta += 50
	} else {
		delta -= 20
	}
	if priority == task.PriorityCritical {
		delta += 100
	}

	char.EXP += delta
	if char.EXP < 0 {
		char.EXP = 0
	}
	char.Level = levelForEXP(char.EXP)
	char.TotalCompleted++

	today := truncateDay(completedAt)
	switch {
	case char.LastCompletionDate == nil:
		char.CurrentStreak = 1
	case truncateDay(*char.LastCompletionDate).Equal(today):
		// already logged a completion today — streak unchanged
	case truncateDay(*char.LastCompletionDate).AddDate(0, 0, 1).Equal(today):
		char.CurrentStreak++
	default:
		char.CurrentStreak = 1
	}
	char.LastCompletionDate = &today
	if char.CurrentStreak > char.LongestStreak {
		char.LongestStreak = char.CurrentStreak
	}

	if err := s.repo.SaveCharacter(ctx, char); err != nil {
		return apperrors.Internal("failed to save character")
	}

	s.checkCountBadges(ctx, userID, char.TotalCompleted)
	if char.CurrentStreak >= 7 {
		s.awardBadgeIfMissing(ctx, userID, BadgeStreak7)
	}
	if onTime {
		s.awardBadgeIfMissing(ctx, userID, BadgeEarlyBird)
		s.checkPerfectWeek(ctx, userID, completedAt)
	}

	s.bumpMission(ctx, char, MissionDaily, dailyPeriodKey(completedAt), DailyMissionThreshold, DailyMissionReward)
	s.bumpMission(ctx, char, MissionWeekly, weeklyPeriodKey(completedAt), WeeklyMissionThreshold, WeeklyMissionReward)
	s.bumpMission(ctx, char, MissionMonthly, monthlyPeriodKey(completedAt), MonthlyMissionThreshold, MonthlyMissionReward)

	return nil
}

func (s *service) checkCountBadges(ctx context.Context, userID uuid.UUID, total int) {
	if total >= 10 {
		s.awardBadgeIfMissing(ctx, userID, BadgeFastWorker)
	}
	if total >= 100 {
		s.awardBadgeIfMissing(ctx, userID, BadgeHundredTasks)
	}
	if total >= 500 {
		s.awardBadgeIfMissing(ctx, userID, BadgeLegend)
	}
}

// checkPerfectWeek awards BadgePerfectWeek once a user has at least 3
// on-time completions in the trailing 7 days with zero late ones in that
// same window.
func (s *service) checkPerfectWeek(ctx context.Context, userID uuid.UUID, now time.Time) {
	since := now.AddDate(0, 0, -7)
	completed, err := s.taskSvc.ListCompletedByAssigneeSince(ctx, userID, since)
	if err != nil {
		return
	}

	onTimeCount, lateCount := 0, 0
	for _, t := range completed {
		if t.CompletedAt == nil {
			continue
		}
		completedAt, err := time.Parse("2006-01-02", *t.CompletedAt)
		if err != nil {
			continue
		}
		late := false
		if t.DueDate != nil {
			due, err := time.Parse("2006-01-02", *t.DueDate)
			if err == nil {
				late = completedAt.After(due)
			}
		}
		if late {
			lateCount++
		} else {
			onTimeCount++
		}
	}

	if onTimeCount >= 3 && lateCount == 0 {
		s.awardBadgeIfMissing(ctx, userID, BadgePerfectWeek)
	}
}

func (s *service) awardBadgeIfMissing(ctx context.Context, userID uuid.UUID, code BadgeCode) {
	has, err := s.repo.HasBadge(ctx, userID, code)
	if err != nil || has {
		return
	}
	_ = s.repo.AwardBadge(ctx, &Badge{UserID: userID, Code: code, AwardedAt: time.Now()})
}

// bumpMission increments this period's progress and, the first time the
// threshold is crossed, grants the one-time EXP reward (re-saving the
// character char points at, which the caller already persisted once this
// call — a second save here is deliberate: the reward can only be known
// after the completion-count increment above).
func (s *service) bumpMission(ctx context.Context, char *Character, typ MissionType, periodKey string, threshold, reward int) {
	m, err := s.repo.FindMissionProgress(ctx, char.UserID, typ, periodKey)
	if err != nil {
		if err != ErrNotFound {
			return
		}
		m = &MissionProgress{UserID: char.UserID, Type: typ, PeriodKey: periodKey}
	}

	m.Count++
	newlyCompleted := !m.Rewarded && m.Count >= threshold
	if newlyCompleted {
		m.Rewarded = true
		char.EXP += reward
		char.Level = levelForEXP(char.EXP)
		_ = s.repo.SaveCharacter(ctx, char)
	}
	_ = s.repo.SaveMissionProgress(ctx, m)
}

func (s *service) GetProfile(ctx context.Context, actorID uuid.UUID) (*ProfileResponse, error) {
	char, err := s.repo.FindCharacter(ctx, actorID)
	if err != nil {
		if err != ErrNotFound {
			return nil, apperrors.Internal("failed to load character")
		}
		char = &Character{UserID: actorID, Level: levelForEXP(0)}
	}

	badges, err := s.repo.ListBadges(ctx, actorID)
	if err != nil {
		return nil, apperrors.Internal("failed to load badges")
	}
	badgeResponses := make([]BadgeResponse, len(badges))
	for i, b := range badges {
		badgeResponses[i] = BadgeResponse{Code: b.Code, AwardedAt: b.AwardedAt.Format(time.RFC3339)}
	}

	now := time.Now()
	missions := []MissionResponse{
		s.missionStatus(ctx, actorID, MissionDaily, dailyPeriodKey(now), DailyMissionThreshold, DailyMissionReward),
		s.missionStatus(ctx, actorID, MissionWeekly, weeklyPeriodKey(now), WeeklyMissionThreshold, WeeklyMissionReward),
		s.missionStatus(ctx, actorID, MissionMonthly, monthlyPeriodKey(now), MonthlyMissionThreshold, MonthlyMissionReward),
	}

	return &ProfileResponse{
		Character: toCharacterResponse(char),
		Badges:    badgeResponses,
		Missions:  missions,
	}, nil
}

func (s *service) missionStatus(ctx context.Context, userID uuid.UUID, typ MissionType, periodKey string, threshold, reward int) MissionResponse {
	m, err := s.repo.FindMissionProgress(ctx, userID, typ, periodKey)
	if err != nil {
		m = &MissionProgress{}
	}
	return MissionResponse{Type: typ, Count: m.Count, Threshold: threshold, Reward: reward, Completed: m.Rewarded}
}

func (s *service) GetLeaderboard(ctx context.Context, actorID uuid.UUID) (*LeaderboardResponse, error) {
	// The leaderboard is intentionally an organization-level feature, not
	// a global one — see the P1.2 tenant isolation audit §12. Characters
	// are keyed only by user_id with no organization_id of their own (see
	// gamification.Character), so scoping happens by intersecting against
	// the actor's own organization's member directory below, rather than
	// a direct query filter — every character NOT found in that directory
	// is filtered out entirely (see the "ok" check below), not merely
	// displayed with a blanked-out name as the pre-P1.2 code did.
	characters, err := s.repo.ListCharacters(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to load leaderboard")
	}

	directory, err := s.teamSvc.GetDirectory(ctx, actorID)
	if err != nil {
		return nil, err
	}
	memberByID := make(map[string]team.DirectoryEntry, len(directory.Members))
	for _, m := range directory.Members {
		memberByID[m.UserID] = m
	}

	individuals := make([]LeaderboardEntry, 0, len(characters))
	deptTotals := make(map[string]*DepartmentLeaderboardEntry)
	for _, c := range characters {
		member, ok := memberByID[c.UserID.String()]
		if !ok {
			// Not a member of the actor's organization — exclude entirely,
			// don't just fall back to displaying their raw user ID as a
			// name (that was the pre-P1.2 behavior and leaked the fact
			// that a character with that ID exists at all, across org
			// boundaries).
			continue
		}
		individuals = append(individuals, LeaderboardEntry{UserID: c.UserID.String(), Name: member.Name, Level: c.Level, EXP: c.EXP})

		if member.DepartmentID != "" {
			dept := deptTotals[member.DepartmentID]
			if dept == nil {
				dept = &DepartmentLeaderboardEntry{DepartmentID: member.DepartmentID, DepartmentName: member.DepartmentName}
				deptTotals[member.DepartmentID] = dept
			}
			dept.TotalEXP += c.EXP
			dept.MemberCount++
		}
	}

	departments := make([]DepartmentLeaderboardEntry, 0, len(deptTotals))
	for _, d := range deptTotals {
		departments = append(departments, *d)
	}
	sortDepartmentsByEXPDesc(departments)

	return &LeaderboardResponse{Individuals: individuals, Departments: departments}, nil
}

func sortDepartmentsByEXPDesc(departments []DepartmentLeaderboardEntry) {
	for i := 1; i < len(departments); i++ {
		for j := i; j > 0 && departments[j].TotalEXP > departments[j-1].TotalEXP; j-- {
			departments[j], departments[j-1] = departments[j-1], departments[j]
		}
	}
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func dailyPeriodKey(t time.Time) string { return t.Format("2006-01-02") }

func weeklyPeriodKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func monthlyPeriodKey(t time.Time) string { return t.Format("2006-01") }
