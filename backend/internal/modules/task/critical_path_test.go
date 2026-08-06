package task

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func date(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func taskWithDates(id uuid.UUID, start, due *time.Time) Task {
	return Task{ID: id, StartDate: start, DueDate: due}
}

func TestCriticalPath_LinearChain(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	tasks := []Task{
		taskWithDates(a, date(2026, 8, 1), date(2026, 8, 2)), // 1 day
		taskWithDates(b, date(2026, 8, 2), date(2026, 8, 4)), // 2 days
		taskWithDates(c, date(2026, 8, 4), date(2026, 8, 5)), // 1 day
	}
	deps := []Dependency{
		{TaskID: b, DependsOnID: a},
		{TaskID: c, DependsOnID: b},
	}

	path, totalDays, err := computeCriticalPath(tasks, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPath(t, path, []uuid.UUID{a, b, c})
	if totalDays != 4 {
		t.Errorf("expected total 4 days, got %d", totalDays)
	}
}

func TestCriticalPath_DiamondPicksLongerBranch(t *testing.T) {
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	tasks := []Task{
		taskWithDates(a, date(2026, 8, 1), date(2026, 8, 2)), // 1 day
		taskWithDates(b, date(2026, 8, 2), date(2026, 8, 7)), // 5 days (long branch)
		taskWithDates(c, date(2026, 8, 2), date(2026, 8, 3)), // 1 day (short branch)
		taskWithDates(d, date(2026, 8, 7), date(2026, 8, 8)), // 1 day
	}
	deps := []Dependency{
		{TaskID: b, DependsOnID: a},
		{TaskID: c, DependsOnID: a},
		{TaskID: d, DependsOnID: b},
		{TaskID: d, DependsOnID: c},
	}

	path, totalDays, err := computeCriticalPath(tasks, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPath(t, path, []uuid.UUID{a, b, d})
	if totalDays != 7 {
		t.Errorf("expected total 7 days, got %d", totalDays)
	}
}

func TestCriticalPath_DisconnectedShorterTaskExcluded(t *testing.T) {
	a, b, solo := uuid.New(), uuid.New(), uuid.New()
	tasks := []Task{
		taskWithDates(a, date(2026, 8, 1), date(2026, 8, 4)),    // 3 days
		taskWithDates(b, date(2026, 8, 4), date(2026, 8, 8)),    // 4 days — main chain totals 7
		taskWithDates(solo, date(2026, 8, 1), date(2026, 8, 2)), // 1 day, no deps
	}
	deps := []Dependency{
		{TaskID: b, DependsOnID: a},
	}

	path, totalDays, err := computeCriticalPath(tasks, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPath(t, path, []uuid.UUID{a, b})
	if totalDays != 7 {
		t.Errorf("expected total 7 days, got %d", totalDays)
	}
}

func TestCriticalPath_SingleTaskNoDependencies(t *testing.T) {
	solo := uuid.New()
	tasks := []Task{taskWithDates(solo, date(2026, 8, 1), date(2026, 8, 4))}

	path, totalDays, err := computeCriticalPath(tasks, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPath(t, path, []uuid.UUID{solo})
	if totalDays != 3 {
		t.Errorf("expected total 3 days, got %d", totalDays)
	}
}

func TestCriticalPath_NoSchedulableTasksReturnsEmpty(t *testing.T) {
	tasks := []Task{{ID: uuid.New()}} // no DueDate

	path, totalDays, err := computeCriticalPath(tasks, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) != 0 {
		t.Errorf("expected empty path, got %v", path)
	}
	if totalDays != 0 {
		t.Errorf("expected 0 total days, got %d", totalDays)
	}
}

func TestCriticalPath_CycleIsDetected(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	tasks := []Task{
		taskWithDates(a, date(2026, 8, 1), date(2026, 8, 2)),
		taskWithDates(b, date(2026, 8, 2), date(2026, 8, 3)),
	}
	deps := []Dependency{
		{TaskID: a, DependsOnID: b},
		{TaskID: b, DependsOnID: a},
	}

	_, _, err := computeCriticalPath(tasks, deps)
	if !errors.Is(err, ErrDependencyCycle) {
		t.Fatalf("expected ErrDependencyCycle, got %v", err)
	}
}

func TestCriticalPath_MilestoneHasZeroDurationButStaysOnPath(t *testing.T) {
	a, milestone, b := uuid.New(), uuid.New(), uuid.New()
	tasks := []Task{
		taskWithDates(a, date(2026, 8, 1), date(2026, 8, 3)),         // 2 days
		taskWithDates(milestone, date(2026, 8, 3), date(2026, 8, 3)), // 0 days
		taskWithDates(b, date(2026, 8, 3), date(2026, 8, 5)),         // 2 days
	}
	deps := []Dependency{
		{TaskID: milestone, DependsOnID: a},
		{TaskID: b, DependsOnID: milestone},
	}

	path, totalDays, err := computeCriticalPath(tasks, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPath(t, path, []uuid.UUID{a, milestone, b})
	if totalDays != 4 {
		t.Errorf("expected total 4 days, got %d", totalDays)
	}
}

func TestCriticalPath_DependencyOnUnknownTaskIsIgnored(t *testing.T) {
	a := uuid.New()
	tasks := []Task{taskWithDates(a, date(2026, 8, 1), date(2026, 8, 2))}
	deps := []Dependency{
		{TaskID: a, DependsOnID: uuid.New()}, // references a task not in the set
	}

	path, _, err := computeCriticalPath(tasks, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPath(t, path, []uuid.UUID{a})
}

func assertPath(t *testing.T, got, want []uuid.UUID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("path length mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path mismatch at index %d: got %v, want %v", i, got, want)
		}
	}
}
