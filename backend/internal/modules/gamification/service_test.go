package gamification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	"github.com/khomkrittk/taskworked/backend/internal/modules/team"
)

// --- fakes ---------------------------------------------------------------

type fakeRepository struct {
	characters map[uuid.UUID]*Character
	badges     map[uuid.UUID]map[BadgeCode]bool
	missions   map[string]*MissionProgress // key: userID|type|periodKey
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		characters: make(map[uuid.UUID]*Character),
		badges:     make(map[uuid.UUID]map[BadgeCode]bool),
		missions:   make(map[string]*MissionProgress),
	}
}

func (f *fakeRepository) FindCharacter(_ context.Context, userID uuid.UUID) (*Character, error) {
	c, ok := f.characters[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (f *fakeRepository) SaveCharacter(_ context.Context, c *Character) error {
	cp := *c
	f.characters[c.UserID] = &cp
	return nil
}

func (f *fakeRepository) ListCharacters(_ context.Context) ([]Character, error) {
	var result []Character
	for _, c := range f.characters {
		result = append(result, *c)
	}
	// Mirror the real repo's ORDER BY exp DESC.
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].EXP > result[j-1].EXP; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result, nil
}

func (f *fakeRepository) HasBadge(_ context.Context, userID uuid.UUID, code BadgeCode) (bool, error) {
	return f.badges[userID][code], nil
}

func (f *fakeRepository) AwardBadge(_ context.Context, b *Badge) error {
	if f.badges[b.UserID] == nil {
		f.badges[b.UserID] = make(map[BadgeCode]bool)
	}
	f.badges[b.UserID][b.Code] = true
	return nil
}

func (f *fakeRepository) ListBadges(_ context.Context, userID uuid.UUID) ([]Badge, error) {
	var result []Badge
	for code := range f.badges[userID] {
		result = append(result, Badge{UserID: userID, Code: code, AwardedAt: time.Now()})
	}
	return result, nil
}

func (f *fakeRepository) missionKey(userID uuid.UUID, typ MissionType, periodKey string) string {
	return userID.String() + "|" + string(typ) + "|" + periodKey
}

func (f *fakeRepository) FindMissionProgress(_ context.Context, userID uuid.UUID, typ MissionType, periodKey string) (*MissionProgress, error) {
	m, ok := f.missions[f.missionKey(userID, typ, periodKey)]
	if !ok {
		return nil, ErrNotFound
	}
	return m, nil
}

func (f *fakeRepository) SaveMissionProgress(_ context.Context, m *MissionProgress) error {
	cp := *m
	f.missions[f.missionKey(m.UserID, m.Type, m.PeriodKey)] = &cp
	return nil
}

type fakeTaskService struct {
	completedByUser map[string][]task.Response
}

func (f *fakeTaskService) ListCompletedByAssigneeSince(_ context.Context, userID uuid.UUID, _ time.Time) ([]task.Response, error) {
	return f.completedByUser[userID.String()], nil
}

type fakeTeamService struct {
	directory *team.DirectoryResponse
}

func (f *fakeTeamService) GetDirectory(context.Context, uuid.UUID) (*team.DirectoryResponse, error) {
	return f.directory, nil
}

// --- tests -----------------------------------------------------------------

func TestOnTaskCompleted_OnTimeAwardsBaseAndBonusEXP(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	dueDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC) // before due date

	if err := svc.OnTaskCompleted(context.Background(), userID, completedAt, &dueDate, task.PriorityMedium); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	char, _ := repo.FindCharacter(context.Background(), userID)
	if char.EXP != 70 { // 20 base + 50 on-time bonus
		t.Errorf("expected 70 EXP, got %d", char.EXP)
	}
}

func TestOnTaskCompleted_CriticalTaskAddsBonus(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	completedAt := time.Now()

	svc.OnTaskCompleted(context.Background(), userID, completedAt, nil, task.PriorityCritical)

	char, _ := repo.FindCharacter(context.Background(), userID)
	if char.EXP != 170 { // 20 base + 50 on-time (no due date = on time) + 100 critical
		t.Errorf("expected 170 EXP, got %d", char.EXP)
	}
}

func TestOnTaskCompleted_LatePenalizesEXP(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	dueDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC) // after due date

	svc.OnTaskCompleted(context.Background(), userID, completedAt, &dueDate, task.PriorityMedium)

	char, _ := repo.FindCharacter(context.Background(), userID)
	if char.EXP != 0 { // 20 base - 20 late penalty
		t.Errorf("expected 0 EXP, got %d", char.EXP)
	}
}

func TestOnTaskCompleted_StreakBuildsAcrossConsecutiveDays(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	day1 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC) // gap — day 3 skipped

	svc.OnTaskCompleted(context.Background(), userID, day1, nil, task.PriorityLow)
	svc.OnTaskCompleted(context.Background(), userID, day2, nil, task.PriorityLow)
	char, _ := repo.FindCharacter(context.Background(), userID)
	if char.CurrentStreak != 2 {
		t.Fatalf("expected streak 2 after two consecutive days, got %d", char.CurrentStreak)
	}

	svc.OnTaskCompleted(context.Background(), userID, day4, nil, task.PriorityLow)
	char, _ = repo.FindCharacter(context.Background(), userID)
	if char.CurrentStreak != 1 {
		t.Errorf("expected streak to reset to 1 after a gap, got %d", char.CurrentStreak)
	}
	if char.LongestStreak != 2 {
		t.Errorf("expected longest streak to remain 2, got %d", char.LongestStreak)
	}
}

func TestOnTaskCompleted_SameDayCompletionDoesNotDoubleCountStreak(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	morning := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	evening := time.Date(2026, 8, 1, 20, 0, 0, 0, time.UTC)

	svc.OnTaskCompleted(context.Background(), userID, morning, nil, task.PriorityLow)
	svc.OnTaskCompleted(context.Background(), userID, evening, nil, task.PriorityLow)

	char, _ := repo.FindCharacter(context.Background(), userID)
	if char.CurrentStreak != 1 {
		t.Errorf("expected streak to stay 1 for two completions on the same day, got %d", char.CurrentStreak)
	}
}

func TestOnTaskCompleted_SevenDayStreakAwardsBadge(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	start := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		svc.OnTaskCompleted(context.Background(), userID, start.AddDate(0, 0, i), nil, task.PriorityLow)
	}

	has, _ := repo.HasBadge(context.Background(), userID, BadgeStreak7)
	if !has {
		t.Error("expected seven_day_streak badge to be awarded")
	}
}

func TestOnTaskCompleted_CountBadgesAwardedAtThresholds(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	for i := 0; i < 10; i++ {
		svc.OnTaskCompleted(context.Background(), userID, time.Now(), nil, task.PriorityLow)
	}

	has, _ := repo.HasBadge(context.Background(), userID, BadgeFastWorker)
	if !has {
		t.Error("expected fast_worker badge after 10 completions")
	}
	has, _ = repo.HasBadge(context.Background(), userID, BadgeHundredTasks)
	if has {
		t.Error("did not expect hundred_tasks badge after only 10 completions")
	}
}

func TestOnTaskCompleted_DailyMissionRewardGrantedOnceThenNotAgain(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	userID := uuid.New()
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

	// DailyMissionThreshold is 3 — complete exactly 3 tasks today.
	for i := 0; i < 3; i++ {
		svc.OnTaskCompleted(context.Background(), userID, now, nil, task.PriorityLow)
	}
	char, _ := repo.FindCharacter(context.Background(), userID)
	expAfterThree := char.EXP // 3 * 70 (base+ontime) + 30 mission reward

	// Query mission progress directly rather than through GetProfile,
	// which anchors "today" to the real wall clock — this test simulates
	// completions on a fixed date instead.
	mission, err := repo.FindMissionProgress(context.Background(), userID, MissionDaily, dailyPeriodKey(now))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mission.Rewarded {
		t.Fatal("expected daily mission to be rewarded after 3 tasks")
	}
	if expAfterThree != 3*70+DailyMissionReward {
		t.Errorf("expected EXP %d after 3 completions + mission reward, got %d", 3*70+DailyMissionReward, expAfterThree)
	}

	// A 4th completion the same day must not grant the reward again.
	svc.OnTaskCompleted(context.Background(), userID, now, nil, task.PriorityLow)
	char, _ = repo.FindCharacter(context.Background(), userID)
	if char.EXP != expAfterThree+70 {
		t.Errorf("expected only +70 for the 4th completion (no repeat mission reward), got total %d", char.EXP)
	}
}

func TestOnTaskCompleted_PerfectWeekBadgeRequiresNoLateCompletions(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()

	// checkPerfectWeek reads entirely from taskSvc.ListCompletedByAssigneeSince
	// (a fake here, so it won't reflect the OnTaskCompleted call below on
	// its own) — seed all 3 on-time completions the badge needs directly.
	dueDate := "2026-08-05"
	completedOnTime := "2026-08-04"
	taskSvc := &fakeTaskService{completedByUser: map[string][]task.Response{
		userID.String(): {
			{ID: "1", DueDate: &dueDate, CompletedAt: &completedOnTime},
			{ID: "2", DueDate: &dueDate, CompletedAt: &completedOnTime},
			{ID: "3", DueDate: &dueDate, CompletedAt: &completedOnTime},
		},
	}}
	svc := NewService(repo, taskSvc, &fakeTeamService{directory: &team.DirectoryResponse{}})

	svc.OnTaskCompleted(context.Background(), userID, time.Now(), nil, task.PriorityLow)

	has, _ := repo.HasBadge(context.Background(), userID, BadgePerfectWeek)
	if !has {
		t.Error("expected perfect_week badge with 3 on-time completions and none late")
	}
}

func TestOnTaskCompleted_PerfectWeekBadgeNotAwardedWithALateCompletion(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()

	dueDate := "2026-08-01"
	completedLate := "2026-08-05" // after due date
	taskSvc := &fakeTaskService{completedByUser: map[string][]task.Response{
		userID.String(): {
			{ID: "1", DueDate: &dueDate, CompletedAt: &completedLate},
		},
	}}
	svc := NewService(repo, taskSvc, &fakeTeamService{directory: &team.DirectoryResponse{}})

	svc.OnTaskCompleted(context.Background(), userID, time.Now(), nil, task.PriorityLow)

	has, _ := repo.HasBadge(context.Background(), userID, BadgePerfectWeek)
	if has {
		t.Error("did not expect perfect_week badge when a late completion exists in the window")
	}
}

func TestGetProfile_UnknownUserReturnsZeroValueWithoutError(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: &team.DirectoryResponse{}})

	profile, err := svc.GetProfile(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Character.Level != 1 || profile.Character.EXP != 0 {
		t.Errorf("expected a fresh level-1, 0-EXP profile, got %+v", profile.Character)
	}
	if len(profile.Badges) != 0 {
		t.Errorf("expected no badges, got %v", profile.Badges)
	}
}

func TestGetLeaderboard_SortsByEXPAndAggregatesDepartments(t *testing.T) {
	repo := newFakeRepository()
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()
	repo.characters[alice] = &Character{UserID: alice, EXP: 100, Level: 2}
	repo.characters[bob] = &Character{UserID: bob, EXP: 300, Level: 4}
	repo.characters[carol] = &Character{UserID: carol, EXP: 50, Level: 1}

	directory := &team.DirectoryResponse{
		Members: []team.DirectoryEntry{
			{UserID: alice.String(), Name: "Alice", DepartmentID: "eng", DepartmentName: "Engineering"},
			{UserID: bob.String(), Name: "Bob", DepartmentID: "eng", DepartmentName: "Engineering"},
			{UserID: carol.String(), Name: "Carol", DepartmentID: "sales", DepartmentName: "Sales"},
		},
	}
	svc := NewService(repo, &fakeTaskService{}, &fakeTeamService{directory: directory})

	board, err := svc.GetLeaderboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(board.Individuals) != 3 || board.Individuals[0].Name != "Bob" {
		t.Fatalf("expected Bob first (highest EXP), got %+v", board.Individuals)
	}

	var eng *DepartmentLeaderboardEntry
	for i := range board.Departments {
		if board.Departments[i].DepartmentID == "eng" {
			eng = &board.Departments[i]
		}
	}
	if eng == nil || eng.TotalEXP != 400 || eng.MemberCount != 2 {
		t.Errorf("expected Engineering total 400 EXP across 2 members, got %+v", eng)
	}
	if board.Departments[0].DepartmentID != "eng" {
		t.Errorf("expected Engineering ranked first (highest total EXP), got %+v", board.Departments)
	}
}
