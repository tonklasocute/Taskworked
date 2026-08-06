package gamification

import "testing"

func TestLevelForEXP(t *testing.T) {
	cases := []struct {
		exp   int
		level int
	}{
		{0, 1},
		{99, 1},
		{100, 2},
		{250, 3},
		{-50, 1}, // negative EXP floors to level 1
	}
	for _, c := range cases {
		if got := levelForEXP(c.exp); got != c.level {
			t.Errorf("levelForEXP(%d) = %d, want %d", c.exp, got, c.level)
		}
	}
}

func TestExpIntoLevel(t *testing.T) {
	cases := []struct {
		exp  int
		want int
	}{
		{0, 0},
		{50, 50},
		{100, 0},
		{250, 50},
	}
	for _, c := range cases {
		if got := expIntoLevel(c.exp); got != c.want {
			t.Errorf("expIntoLevel(%d) = %d, want %d", c.exp, got, c.want)
		}
	}
}

func TestLevelTitle(t *testing.T) {
	cases := []struct {
		level int
		title string
	}{
		{1, "Novice"},
		{4, "Novice"},
		{5, "Adept"},
		{9, "Adept"},
		{10, "Master"},
		{19, "Master"},
		{20, "Legend"},
		{100, "Legend"},
	}
	for _, c := range cases {
		if got := levelTitle(c.level); got != c.title {
			t.Errorf("levelTitle(%d) = %q, want %q", c.level, got, c.title)
		}
	}
}
