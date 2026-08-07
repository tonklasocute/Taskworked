//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

// TestBootstrap_FirstUserIsSuperAdmin needs to be the very first thing
// that ever touches the users table, so it gets its own dedicated
// database rather than the shared one the other scenarios use.
func TestBootstrap_FirstUserIsSuperAdmin(t *testing.T) {
	app := newTestApp(t, "taskworked_e2e_bootstrap")

	first := registerUser(t, app, "First Admin", "first@e2e.test")
	if first.Role != "super_admin" {
		t.Errorf("expected the first registered account to be super_admin, got %q", first.Role)
	}

	second := registerUser(t, app, "Second User", "second@e2e.test")
	if second.Role != "employee" {
		t.Errorf("expected the second registered account to be a plain employee, got %q", second.Role)
	}
}

// TestProjectAndTaskLifecycle_AwardsGamification walks a task from
// creation through completion and checks that gamification — a separate
// module wired in only via the local Gamifier interface — actually
// picked up the credit. This is exactly the kind of cross-module wiring
// bug unit tests (which fake out the other module) can't catch.
func TestProjectAndTaskLifecycle_AwardsGamification(t *testing.T) {
	app := newTestApp(t, "taskworked_e2e")
	user := registerUser(t, app, "Lifecycle User", uniqueEmail("lifecycle"))

	projResp := request(t, app, http.MethodPost, "/api/v1/projects", user.AccessToken, map[string]string{
		"name": "E2E Project",
	})
	requireStatus(t, projResp, http.StatusCreated, "create project")
	var project struct {
		ID string `json:"id"`
	}
	decode(t, projResp, &project)

	taskResp := request(t, app, http.MethodPost, "/api/v1/tasks", user.AccessToken, map[string]any{
		"project_id":  project.ID,
		"title":       "Ship the E2E suite",
		"assignee_id": user.UserID,
	})
	requireStatus(t, taskResp, http.StatusCreated, "create task")
	var createdTask struct {
		ID string `json:"id"`
	}
	decode(t, taskResp, &createdTask)

	updateResp := request(t, app, http.MethodPatch, "/api/v1/tasks/"+createdTask.ID, user.AccessToken, map[string]string{
		"status": "done",
	})
	requireStatus(t, updateResp, http.StatusOK, "complete task")

	profileResp := request(t, app, http.MethodGet, "/api/v1/gamification/profile", user.AccessToken, nil)
	requireStatus(t, profileResp, http.StatusOK, "get gamification profile")
	var profile struct {
		Character struct {
			TotalCompleted int `json:"total_completed"`
			EXP            int `json:"exp"`
		} `json:"character"`
	}
	decode(t, profileResp, &profile)

	if profile.Character.TotalCompleted != 1 {
		t.Errorf("expected 1 completed task credited to gamification, got %d", profile.Character.TotalCompleted)
	}
	if profile.Character.EXP <= 0 {
		t.Errorf("expected positive EXP after completing a task, got %d", profile.Character.EXP)
	}
}

// TestPermissionBoundary_NonMemberCannotListProjectTasks checks a
// permission check through the full HTTP path (middleware -> handler ->
// service -> project.Service.CheckMembership), not just the service unit
// test that already covers the same logic in isolation.
func TestPermissionBoundary_NonMemberCannotListProjectTasks(t *testing.T) {
	app := newTestApp(t, "taskworked_e2e")
	owner := registerUser(t, app, "Owner", uniqueEmail("owner"))
	outsider := registerUser(t, app, "Outsider", uniqueEmail("outsider"))

	projResp := request(t, app, http.MethodPost, "/api/v1/projects", owner.AccessToken, map[string]string{
		"name": "Private Project",
	})
	requireStatus(t, projResp, http.StatusCreated, "create project")
	var project struct {
		ID string `json:"id"`
	}
	decode(t, projResp, &project)

	listResp := request(t, app, http.MethodGet, "/api/v1/tasks?project_id="+project.ID, outsider.AccessToken, nil)
	if listResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for a non-member listing tasks, got %d (%s)", listResp.StatusCode, listResp.Error)
	}
}

// TestCommentNotifiesWatcher exercises task -> notification wiring: the
// task module raises the notification through the local Notifier
// interface, and the notification module has to actually persist and
// serve it back for a different user than the one who commented.
func TestCommentNotifiesWatcher(t *testing.T) {
	app := newTestApp(t, "taskworked_e2e")
	owner := registerUser(t, app, "Owner", uniqueEmail("commentowner"))
	watcher := registerUser(t, app, "Watcher", uniqueEmail("watcher"))

	projResp := request(t, app, http.MethodPost, "/api/v1/projects", owner.AccessToken, map[string]string{
		"name": "Comment Project",
	})
	requireStatus(t, projResp, http.StatusCreated, "create project")
	var project struct {
		ID string `json:"id"`
	}
	decode(t, projResp, &project)

	addMemberResp := request(t, app, http.MethodPost, "/api/v1/projects/"+project.ID+"/members", owner.AccessToken, map[string]string{
		"user_id": watcher.UserID, "role": "member",
	})
	requireStatus(t, addMemberResp, http.StatusCreated, "add member")

	taskResp := request(t, app, http.MethodPost, "/api/v1/tasks", owner.AccessToken, map[string]any{
		"project_id": project.ID, "title": "Discuss this",
	})
	requireStatus(t, taskResp, http.StatusCreated, "create task")
	var createdTask struct {
		ID string `json:"id"`
	}
	decode(t, taskResp, &createdTask)

	watchResp := request(t, app, http.MethodPost, "/api/v1/tasks/"+createdTask.ID+"/watch", watcher.AccessToken, nil)
	requireStatus(t, watchResp, http.StatusOK, "watch task")

	commentResp := request(t, app, http.MethodPost, "/api/v1/tasks/"+createdTask.ID+"/comments", owner.AccessToken, map[string]string{
		"body": "Please review",
	})
	requireStatus(t, commentResp, http.StatusCreated, "add comment")

	notifResp := request(t, app, http.MethodGet, "/api/v1/notifications", watcher.AccessToken, nil)
	requireStatus(t, notifResp, http.StatusOK, "list notifications")
	var notifs struct {
		Items []struct {
			Type string `json:"type"`
			Body string `json:"body"`
		} `json:"items"`
	}
	decode(t, notifResp, &notifs)

	found := false
	for _, n := range notifs.Items {
		if n.Type == "comment" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the watcher to receive a comment notification, got %+v", notifs.Items)
	}
}
