package notification

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// AuthService is the slice of auth.Service the notifier needs.
type AuthService interface {
	ListUsers(ctx context.Context) ([]auth.UserResponse, error)
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]auth.UserResponse, error)
}

// TaskService is the slice of task.Service the digest builders need.
type TaskService interface {
	ListActiveByAssignee(ctx context.Context, userID uuid.UUID) ([]task.Response, error)
	ListCompletedByAssigneeSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]task.Response, error)
}

// Broadcaster delivers the in-app event live. *realtime.Publisher
// satisfies this structurally via its PublishToUser method.
type Broadcaster interface {
	PublishToUser(ctx context.Context, userID string, event any) error
}

// EmailSender and LineSender are pluggable so tests don't need real SMTP
// or a real LINE Notify token.
type EmailSender interface {
	Send(to, subject, body string) error
}

type LineSender interface {
	Send(token, message string) error
}

// Event is the payload pushed over a user's personal WebSocket channel.
type Event struct {
	Type         string   `json:"type"` // "notification.created"
	Notification Response `json:"notification"`
}

type Service interface {
	ListNotifications(ctx context.Context, actorID uuid.UUID) (*ListResponse, error)
	MarkRead(ctx context.Context, actorID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, actorID uuid.UUID) error

	GetPreference(ctx context.Context, actorID uuid.UUID) (*PreferenceResponse, error)
	UpdatePreference(ctx context.Context, actorID uuid.UUID, req UpdatePreferenceRequest) (*PreferenceResponse, error)

	// NotifyAssignment and NotifyComment satisfy task.Notifier
	// structurally — task.Service calls these through that local
	// interface, never importing this package directly (see the package
	// doc comment in model.go).
	NotifyAssignment(ctx context.Context, assigneeID uuid.UUID, taskID, taskTitle string) error
	NotifyComment(ctx context.Context, userID uuid.UUID, taskID, taskTitle, commentBody string) error

	// The four digests are invoked by cron (see cmd/api/main.go). Each
	// silently skips a user who has nothing to report — no empty digests.
	SendDailyDigests(ctx context.Context)
	SendEndOfDayDigests(ctx context.Context)
	SendWeeklyDigests(ctx context.Context)
	SendMonthlyDigests(ctx context.Context)
}

type service struct {
	repo        Repository
	authSvc     AuthService
	taskSvc     TaskService
	broadcaster Broadcaster
	emailSender EmailSender
	lineSender  LineSender
}

func NewService(repo Repository, authSvc AuthService, taskSvc TaskService, broadcaster Broadcaster, emailSender EmailSender, lineSender LineSender) Service {
	return &service{
		repo: repo, authSvc: authSvc, taskSvc: taskSvc,
		broadcaster: broadcaster, emailSender: emailSender, lineSender: lineSender,
	}
}

func (s *service) ListNotifications(ctx context.Context, actorID uuid.UUID) (*ListResponse, error) {
	notifications, err := s.repo.ListForUser(ctx, actorID, 50)
	if err != nil {
		return nil, apperrors.Internal("failed to list notifications")
	}
	unread, err := s.repo.CountUnread(ctx, actorID)
	if err != nil {
		return nil, apperrors.Internal("failed to count unread notifications")
	}

	items := make([]Response, len(notifications))
	for i := range notifications {
		items[i] = toResponse(&notifications[i])
	}
	return &ListResponse{Items: items, UnreadCount: unread}, nil
}

func (s *service) MarkRead(ctx context.Context, actorID, id uuid.UUID) error {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperrors.NotFound("notification not found")
	}
	if n.UserID != actorID {
		return apperrors.Forbidden("not your notification")
	}
	if err := s.repo.MarkRead(ctx, id); err != nil {
		return apperrors.Internal("failed to mark notification read")
	}
	return nil
}

func (s *service) MarkAllRead(ctx context.Context, actorID uuid.UUID) error {
	if err := s.repo.MarkAllRead(ctx, actorID); err != nil {
		return apperrors.Internal("failed to mark notifications read")
	}
	return nil
}

func (s *service) GetPreference(ctx context.Context, actorID uuid.UUID) (*PreferenceResponse, error) {
	pref, err := s.getOrDefaultPreference(ctx, actorID)
	if err != nil {
		return nil, err
	}
	resp := toPreferenceResponse(pref)
	return &resp, nil
}

func (s *service) UpdatePreference(ctx context.Context, actorID uuid.UUID, req UpdatePreferenceRequest) (*PreferenceResponse, error) {
	pref, err := s.getOrDefaultPreference(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if req.EmailEnabled != nil {
		pref.EmailEnabled = *req.EmailEnabled
	}
	if req.LineEnabled != nil {
		pref.LineEnabled = *req.LineEnabled
	}
	if req.LineNotifyToken != nil {
		pref.LineNotifyToken = *req.LineNotifyToken
	}

	if err := s.repo.UpsertPreference(ctx, pref); err != nil {
		return nil, apperrors.Internal("failed to update preferences")
	}
	resp := toPreferenceResponse(pref)
	return &resp, nil
}

func (s *service) getOrDefaultPreference(ctx context.Context, userID uuid.UUID) (*Preference, error) {
	pref, err := s.repo.FindPreference(ctx, userID)
	if err == nil {
		return pref, nil
	}
	if err != ErrNotFound {
		return nil, apperrors.Internal("failed to load preferences")
	}
	return &Preference{UserID: userID, EmailEnabled: true, LineEnabled: false}, nil
}

func (s *service) NotifyAssignment(ctx context.Context, assigneeID uuid.UUID, taskID, taskTitle string) error {
	return s.notify(ctx, assigneeID, TypeAssignment, "You were assigned a task", taskTitle, "/tasks/"+taskID)
}

func (s *service) NotifyComment(ctx context.Context, userID uuid.UUID, taskID, taskTitle, commentBody string) error {
	preview := commentBody
	const maxPreview = 200
	if len(preview) > maxPreview {
		preview = preview[:maxPreview] + "…"
	}
	return s.notify(ctx, userID, TypeComment, "New activity on \""+taskTitle+"\"", preview, "/tasks/"+taskID)
}

// notify persists, broadcasts, and (per preference) emails/LINEs a single
// notification. Delivery-channel failures are logged, not returned — a
// down SMTP server shouldn't take out in-app notifications too.
func (s *service) notify(ctx context.Context, userID uuid.UUID, typ Type, title, body, link string) error {
	n := &Notification{UserID: userID, Type: typ, Title: title, Body: body, Link: link}
	if err := s.repo.Create(ctx, n); err != nil {
		return apperrors.Internal("failed to create notification")
	}

	resp := toResponse(n)
	if s.broadcaster != nil {
		if err := s.broadcaster.PublishToUser(ctx, userID.String(), Event{Type: "notification.created", Notification: resp}); err != nil {
			log.Printf("notification: broadcast failed: %v", err)
		}
	}

	pref, err := s.getOrDefaultPreference(ctx, userID)
	if err != nil {
		log.Printf("notification: failed to load preference for %s: %v", userID, err)
		return nil
	}

	if pref.EmailEnabled && s.emailSender != nil {
		if users, err := s.authSvc.GetUsersByIDs(ctx, []uuid.UUID{userID}); err == nil && len(users) == 1 {
			if err := s.emailSender.Send(users[0].Email, title, body); err != nil {
				log.Printf("notification: email send failed: %v", err)
			}
		}
	}
	if pref.LineEnabled && pref.LineNotifyToken != "" && s.lineSender != nil {
		if err := s.lineSender.Send(pref.LineNotifyToken, title+"\n"+body); err != nil {
			log.Printf("notification: LINE send failed: %v", err)
		}
	}

	return nil
}

func (s *service) SendDailyDigests(ctx context.Context) {
	s.forEachUser(ctx, func(userID uuid.UUID) {
		active, err := s.taskSvc.ListActiveByAssignee(ctx, userID)
		if err != nil || len(active) == 0 {
			return
		}

		overdue, nearDue, dueToday := 0, 0, 0
		today := truncateDay(time.Now())
		for _, t := range active {
			if t.DueDate == nil {
				continue
			}
			due, err := time.Parse("2006-01-02", *t.DueDate)
			if err != nil {
				continue
			}
			switch {
			case due.Before(today):
				overdue++
			case due.Equal(today):
				dueToday++
			case !due.After(today.AddDate(0, 0, 3)):
				nearDue++
			}
		}
		if overdue == 0 && nearDue == 0 && dueToday == 0 {
			return
		}

		body := fmt.Sprintf("Due today: %d · Near due: %d · Overdue: %d · Total active: %d", dueToday, nearDue, overdue, len(active))
		_ = s.notify(ctx, userID, TypeDailyDigest, "Daily summary", body, "")
	})
}

func (s *service) SendEndOfDayDigests(ctx context.Context) {
	since := truncateDay(time.Now())
	s.sendCompletionDigest(ctx, TypeEODDigest, "End of day summary", since)
}

func (s *service) SendWeeklyDigests(ctx context.Context) {
	since := truncateDay(time.Now()).AddDate(0, 0, -7)
	s.sendCompletionDigest(ctx, TypeWeeklyDigest, "Weekly summary", since)
}

func (s *service) SendMonthlyDigests(ctx context.Context) {
	since := truncateDay(time.Now()).AddDate(0, -1, 0)
	s.sendCompletionDigest(ctx, TypeMonthlyDigest, "Monthly summary", since)
}

func (s *service) sendCompletionDigest(ctx context.Context, typ Type, title string, since time.Time) {
	s.forEachUser(ctx, func(userID uuid.UUID) {
		completed, err := s.taskSvc.ListCompletedByAssigneeSince(ctx, userID, since)
		if err != nil {
			return
		}
		active, err := s.taskSvc.ListActiveByAssignee(ctx, userID)
		if err != nil {
			return
		}
		if len(completed) == 0 && len(active) == 0 {
			return
		}

		body := fmt.Sprintf("Completed: %d · Still active: %d", len(completed), len(active))
		_ = s.notify(ctx, userID, typ, title, body, "")
	})
}

func (s *service) forEachUser(ctx context.Context, fn func(userID uuid.UUID)) {
	users, err := s.authSvc.ListUsers(ctx)
	if err != nil {
		log.Printf("notification: failed to list users for digest: %v", err)
		return
	}
	for _, u := range users {
		userID, err := uuid.Parse(u.ID)
		if err != nil {
			continue
		}
		fn(userID)
	}
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
