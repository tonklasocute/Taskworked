package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
)

// --- fakes ---------------------------------------------------------------

type fakeRepository struct {
	notifications map[uuid.UUID]*Notification
	preferences   map[uuid.UUID]*Preference
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		notifications: make(map[uuid.UUID]*Notification),
		preferences:   make(map[uuid.UUID]*Preference),
	}
}

func (f *fakeRepository) Create(_ context.Context, n *Notification) error {
	n.ID = uuid.New()
	n.CreatedAt = time.Now()
	f.notifications[n.ID] = n
	return nil
}

func (f *fakeRepository) ListForUser(_ context.Context, userID uuid.UUID, limit int) ([]Notification, error) {
	var result []Notification
	for _, n := range f.notifications {
		if n.UserID == userID {
			result = append(result, *n)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (f *fakeRepository) CountUnread(_ context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	for _, n := range f.notifications {
		if n.UserID == userID && !n.Read {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepository) FindByID(_ context.Context, id uuid.UUID) (*Notification, error) {
	n, ok := f.notifications[id]
	if !ok {
		return nil, ErrNotFound
	}
	return n, nil
}

func (f *fakeRepository) MarkRead(_ context.Context, id uuid.UUID) error {
	if n, ok := f.notifications[id]; ok {
		n.Read = true
	}
	return nil
}

func (f *fakeRepository) MarkAllRead(_ context.Context, userID uuid.UUID) error {
	for _, n := range f.notifications {
		if n.UserID == userID {
			n.Read = true
		}
	}
	return nil
}

func (f *fakeRepository) FindPreference(_ context.Context, userID uuid.UUID) (*Preference, error) {
	p, ok := f.preferences[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (f *fakeRepository) UpsertPreference(_ context.Context, p *Preference) error {
	f.preferences[p.UserID] = p
	return nil
}

type fakeAuthService struct {
	users []auth.UserResponse
}

func (f *fakeAuthService) ListUsers(context.Context) ([]auth.UserResponse, error) {
	return f.users, nil
}

func (f *fakeAuthService) GetUsersByIDs(_ context.Context, ids []uuid.UUID) ([]auth.UserResponse, error) {
	var result []auth.UserResponse
	for _, id := range ids {
		for _, u := range f.users {
			if u.ID == id.String() {
				result = append(result, u)
			}
		}
	}
	return result, nil
}

type fakeTaskService struct {
	activeByUser    map[string][]task.Response
	completedByUser map[string][]task.Response
}

func newFakeTaskService() *fakeTaskService {
	return &fakeTaskService{activeByUser: map[string][]task.Response{}, completedByUser: map[string][]task.Response{}}
}

func (f *fakeTaskService) ListActiveByAssignee(_ context.Context, userID uuid.UUID) ([]task.Response, error) {
	return f.activeByUser[userID.String()], nil
}

func (f *fakeTaskService) ListCompletedByAssigneeSince(_ context.Context, userID uuid.UUID, _ time.Time) ([]task.Response, error) {
	return f.completedByUser[userID.String()], nil
}

type fakeBroadcaster struct {
	published []Event
}

func (f *fakeBroadcaster) PublishToUser(_ context.Context, _ string, event any) error {
	f.published = append(f.published, event.(Event))
	return nil
}

type fakeEmailSender struct {
	sentTo []string
}

func (f *fakeEmailSender) Send(to, _, _ string) error {
	f.sentTo = append(f.sentTo, to)
	return nil
}

type fakeLineSender struct {
	sentTokens []string
}

func (f *fakeLineSender) Send(token, _ string) error {
	f.sentTokens = append(f.sentTokens, token)
	return nil
}

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *apperrors.AppError, got %T (%v)", err, err)
	}
	return appErr.Status
}

// newTestService takes interface-typed sender params (not the concrete
// fake types) so that passing literal nil produces a truly nil interface
// — passing nil through a concrete *fakeEmailSender param would instead
// wrap a nil pointer in a non-nil interface value, and `!= nil` checks in
// the service would wrongly pass.
func newTestService(repo *fakeRepository, authSvc AuthService, taskSvc TaskService, broadcaster Broadcaster, email EmailSender, line LineSender) Service {
	return NewService(repo, authSvc, taskSvc, broadcaster, email, line)
}

// --- tests -----------------------------------------------------------------

func TestNotifyAssignment_PersistsAndBroadcasts(t *testing.T) {
	repo := newFakeRepository()
	broadcaster := &fakeBroadcaster{}
	svc := newTestService(repo, &fakeAuthService{}, newFakeTaskService(), broadcaster, nil, nil)

	userID := uuid.New()
	if err := svc.NotifyAssignment(context.Background(), userID, "task-1", "Set up CI"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, err := svc.ListNotifications(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 || list.UnreadCount != 1 {
		t.Fatalf("expected 1 unread notification, got %+v", list)
	}
	if len(broadcaster.published) != 1 {
		t.Errorf("expected 1 broadcast, got %d", len(broadcaster.published))
	}
}

func TestNotify_SendsEmailWhenEnabled(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	authSvc := &fakeAuthService{users: []auth.UserResponse{{ID: userID.String(), Email: "alice@example.com"}}}
	email := &fakeEmailSender{}
	svc := newTestService(repo, authSvc, newFakeTaskService(), &fakeBroadcaster{}, email, nil)

	svc.NotifyAssignment(context.Background(), userID, "task-1", "Set up CI")

	if len(email.sentTo) != 1 || email.sentTo[0] != "alice@example.com" {
		t.Errorf("expected email sent to alice@example.com, got %v", email.sentTo)
	}
}

func TestNotify_SkipsEmailWhenDisabled(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	repo.preferences[userID] = &Preference{UserID: userID, EmailEnabled: false}
	authSvc := &fakeAuthService{users: []auth.UserResponse{{ID: userID.String(), Email: "alice@example.com"}}}
	email := &fakeEmailSender{}
	svc := newTestService(repo, authSvc, newFakeTaskService(), &fakeBroadcaster{}, email, nil)

	svc.NotifyAssignment(context.Background(), userID, "task-1", "Set up CI")

	if len(email.sentTo) != 0 {
		t.Errorf("expected no email sent, got %v", email.sentTo)
	}
}

func TestNotify_SendsLineWhenEnabledWithToken(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	repo.preferences[userID] = &Preference{UserID: userID, LineEnabled: true, LineNotifyToken: "tok-123"}
	line := &fakeLineSender{}
	svc := newTestService(repo, &fakeAuthService{}, newFakeTaskService(), &fakeBroadcaster{}, nil, line)

	svc.NotifyAssignment(context.Background(), userID, "task-1", "Set up CI")

	if len(line.sentTokens) != 1 || line.sentTokens[0] != "tok-123" {
		t.Errorf("expected LINE sent with token tok-123, got %v", line.sentTokens)
	}
}

func TestMarkRead_ByOwnerSucceeds(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(repo, &fakeAuthService{}, newFakeTaskService(), &fakeBroadcaster{}, nil, nil)

	userID := uuid.New()
	svc.NotifyAssignment(context.Background(), userID, "task-1", "Set up CI")
	list, _ := svc.ListNotifications(context.Background(), userID)
	notifID := mustParse(t, list.Items[0].ID)

	if err := svc.MarkRead(context.Background(), userID, notifID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, _ = svc.ListNotifications(context.Background(), userID)
	if list.UnreadCount != 0 {
		t.Errorf("expected 0 unread after mark read, got %d", list.UnreadCount)
	}
}

func TestMarkRead_ByNonOwnerForbidden(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(repo, &fakeAuthService{}, newFakeTaskService(), &fakeBroadcaster{}, nil, nil)

	owner := uuid.New()
	intruder := uuid.New()
	svc.NotifyAssignment(context.Background(), owner, "task-1", "Set up CI")
	list, _ := svc.ListNotifications(context.Background(), owner)
	notifID := mustParse(t, list.Items[0].ID)

	err := svc.MarkRead(context.Background(), intruder, notifID)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if status := appErrStatus(t, err); status != 403 {
		t.Errorf("expected 403, got %d", status)
	}
}

func TestUpdatePreference_PersistsChanges(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(repo, &fakeAuthService{}, newFakeTaskService(), &fakeBroadcaster{}, nil, nil)

	userID := uuid.New()
	token := "my-line-token"
	enabled := true
	updated, err := svc.UpdatePreference(context.Background(), userID, UpdatePreferenceRequest{LineEnabled: &enabled, LineNotifyToken: &token})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.LineEnabled || updated.LineNotifyToken != token {
		t.Errorf("expected LINE enabled with token %q, got %+v", token, updated)
	}

	fetched, err := svc.GetPreference(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fetched.LineEnabled || fetched.LineNotifyToken != token {
		t.Errorf("expected persisted preference to stick, got %+v", fetched)
	}
}

func TestSendDailyDigests_SkipsUsersWithNothingDue(t *testing.T) {
	repo := newFakeRepository()
	userWithNothing := uuid.New()
	userWithOverdue := uuid.New()

	authSvc := &fakeAuthService{users: []auth.UserResponse{
		{ID: userWithNothing.String()},
		{ID: userWithOverdue.String()},
	}}
	taskSvc := newFakeTaskService()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	taskSvc.activeByUser[userWithOverdue.String()] = []task.Response{{ID: "t1", Title: "Overdue thing", Status: task.StatusTodo, DueDate: &yesterday}}

	svc := newTestService(repo, authSvc, taskSvc, &fakeBroadcaster{}, nil, nil)
	svc.SendDailyDigests(context.Background())

	nothingList, _ := svc.ListNotifications(context.Background(), userWithNothing)
	if len(nothingList.Items) != 0 {
		t.Errorf("expected no digest for user with nothing due, got %v", nothingList.Items)
	}
	overdueList, _ := svc.ListNotifications(context.Background(), userWithOverdue)
	if len(overdueList.Items) != 1 {
		t.Errorf("expected 1 digest for user with an overdue task, got %v", overdueList.Items)
	}
}

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse uuid %q: %v", s, err)
	}
	return id
}
