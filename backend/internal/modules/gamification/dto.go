package gamification

// expPerLevel is a flat 100 EXP per level. A progressive curve (harder at
// higher levels) is a reasonable future refinement, but the spec doesn't
// mandate one and a flat rate is trivial to reason about and test.
const expPerLevel = 100

func levelForEXP(exp int) int {
	if exp < 0 {
		exp = 0
	}
	return exp/expPerLevel + 1
}

func expIntoLevel(exp int) int {
	if exp < 0 {
		exp = 0
	}
	return exp % expPerLevel
}

// levelTitle is the "Evolution Character" display tier — a computed
// label/stage from Level, not a separate stored entity.
func levelTitle(level int) string {
	switch {
	case level >= 20:
		return "Legend"
	case level >= 10:
		return "Master"
	case level >= 5:
		return "Adept"
	default:
		return "Novice"
	}
}

type CharacterResponse struct {
	Level          int    `json:"level"`
	LevelTitle     string `json:"level_title"`
	EXP            int    `json:"exp"`
	EXPIntoLevel   int    `json:"exp_into_level"`
	EXPPerLevel    int    `json:"exp_per_level"`
	TotalCompleted int    `json:"total_completed"`
	CurrentStreak  int    `json:"current_streak"`
	LongestStreak  int    `json:"longest_streak"`
}

func toCharacterResponse(c *Character) CharacterResponse {
	return CharacterResponse{
		Level:          c.Level,
		LevelTitle:     levelTitle(c.Level),
		EXP:            c.EXP,
		EXPIntoLevel:   expIntoLevel(c.EXP),
		EXPPerLevel:    expPerLevel,
		TotalCompleted: c.TotalCompleted,
		CurrentStreak:  c.CurrentStreak,
		LongestStreak:  c.LongestStreak,
	}
}

type BadgeResponse struct {
	Code      BadgeCode `json:"code"`
	AwardedAt string    `json:"awarded_at"`
}

type MissionResponse struct {
	Type      MissionType `json:"type"`
	Count     int         `json:"count"`
	Threshold int         `json:"threshold"`
	Reward    int         `json:"reward"`
	Completed bool        `json:"completed"`
}

type ProfileResponse struct {
	Character CharacterResponse `json:"character"`
	Badges    []BadgeResponse   `json:"badges"`
	Missions  []MissionResponse `json:"missions"`
}

type LeaderboardEntry struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Level  int    `json:"level"`
	EXP    int    `json:"exp"`
}

type DepartmentLeaderboardEntry struct {
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	TotalEXP       int    `json:"total_exp"`
	MemberCount    int    `json:"member_count"`
}

type LeaderboardResponse struct {
	Individuals []LeaderboardEntry           `json:"individuals"`
	Departments []DepartmentLeaderboardEntry `json:"departments"`
}
