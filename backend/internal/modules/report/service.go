package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

const dateLayout = "2006-01-02"

type Service interface {
	GetPeriodReport(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID, from, to time.Time) (*PeriodReport, error)
	// ExportCSV dumps every task currently in the project (not scoped to
	// a period — the on-screen report is where period scoping matters;
	// the export is a full reference table to open in a spreadsheet).
	ExportCSV(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) ([]byte, error)
}

type service struct {
	projectSvc project.Service
	taskSvc    task.Service
}

func NewService(projectSvc project.Service, taskSvc task.Service) Service {
	return &service{projectSvc: projectSvc, taskSvc: taskSvc}
}

func (s *service) GetPeriodReport(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID, from, to time.Time) (*PeriodReport, error) {
	isMember, _, err := s.projectSvc.CheckMembership(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, apperrors.Internal("failed to check project membership")
	}
	if !isMember {
		return nil, apperrors.Forbidden("not a member of this project")
	}

	tasks, err := s.taskSvc.ListAllResponses(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := truncateDay(now)
	nearDueCutoff := today.AddDate(0, 0, 3)

	summary := Summary{TotalTasks: len(tasks)}
	var lateTasks []task.Response
	performanceByAssignee := make(map[string]*AssigneePerformance)
	completedByDay := make(map[string]int)

	for _, t := range tasks {
		if !t.CreatedAt.Before(from) && !t.CreatedAt.After(to) {
			summary.CreatedInPeriod++
		}

		completedAt := parseDate(t.CompletedAt)
		dueDate := parseDate(t.DueDate)

		if completedAt != nil && !completedAt.Before(from) && !completedAt.After(to) {
			summary.CompletedInPeriod++
			completedByDay[completedAt.Format(dateLayout)]++

			if t.AssigneeID != nil {
				perf := performanceByAssignee[*t.AssigneeID]
				if perf == nil {
					perf = &AssigneePerformance{AssigneeID: *t.AssigneeID}
					performanceByAssignee[*t.AssigneeID] = perf
				}
				perf.CompletedCount++
				if dueDate != nil && completedAt.After(*dueDate) {
					perf.LateCount++
				} else {
					perf.OnTimeCount++
				}
			}
		}

		if t.Status != task.StatusDone && dueDate != nil {
			if dueDate.Before(today) {
				summary.OverdueCount++
				lateTasks = append(lateTasks, t)
			} else if !dueDate.After(nearDueCutoff) {
				summary.NearDueCount++
			}
		}
	}

	if summary.CreatedInPeriod > 0 {
		summary.CompletionRate = float64(summary.CompletedInPeriod) / float64(summary.CreatedInPeriod)
	}

	sort.Slice(lateTasks, func(i, j int) bool {
		return *lateTasks[i].DueDate < *lateTasks[j].DueDate
	})

	performance := make([]AssigneePerformance, 0, len(performanceByAssignee))
	for _, perf := range performanceByAssignee {
		if perf.CompletedCount > 0 {
			perf.OnTimeRate = float64(perf.OnTimeCount) / float64(perf.CompletedCount)
		}
		performance = append(performance, *perf)
	}
	sort.Slice(performance, func(i, j int) bool { return performance[i].AssigneeID < performance[j].AssigneeID })

	productivity := make([]ProductivityPoint, 0)
	for d := truncateDay(from); !d.After(truncateDay(to)); d = d.AddDate(0, 0, 1) {
		key := d.Format(dateLayout)
		productivity = append(productivity, ProductivityPoint{Date: key, CompletedCount: completedByDay[key]})
	}

	return &PeriodReport{
		From:         from.Format(dateLayout),
		To:           to.Format(dateLayout),
		Summary:      summary,
		LateTasks:    lateTasks,
		Performance:  performance,
		Productivity: productivity,
	}, nil
}

func (s *service) ExportCSV(ctx context.Context, actorID uuid.UUID, actorRole auth.Role, projectID uuid.UUID) ([]byte, error) {
	tasks, err := s.taskSvc.ListAllResponses(ctx, actorID, actorRole, projectID)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Title", "Status", "Priority", "Assignee ID", "Start Date", "Due Date", "Completed At", "Created At"})
	for _, t := range tasks {
		_ = w.Write([]string{
			t.Title,
			string(t.Status),
			string(t.Priority),
			deref(t.AssigneeID),
			deref(t.StartDate),
			deref(t.DueDate),
			deref(t.CompletedAt),
			t.CreatedAt.Format(dateLayout),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, apperrors.Internal("failed to generate CSV")
	}
	return buf.Bytes(), nil
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func parseDate(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil
	}
	return &t
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
