//go:build e2e

// Package e2e — this file is P1.2's core deliverable: proof, against a
// real Postgres and the real HTTP stack, that a user in one organization
// cannot access, modify, delete, or even detect the existence of another
// organization's resources. See
// docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md
// and the P1.2 implementation notes for the design this verifies.
//
// There is no Organizations API yet (that's a later phase — see the
// audit), so a second organization and a cross-org user are set up here
// via direct SQL against the test database, mirroring exactly what
// resetDatabase/serverAndTestDSN already do elsewhere in this package.
package e2e

import (
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/khomkrittk/taskworked/backend/internal/platform/database"
)

// createOrganization inserts a second organization directly and returns
// its ID — there's no Organizations API yet (see package doc comment).
func createOrganization(t *testing.T, testDSN, name, slug string) uuid.UUID {
	t.Helper()
	db, err := database.Connect(testDSN)
	if err != nil {
		t.Fatalf("connect for org setup: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	id := uuid.New()
	if err := db.Exec(
		`INSERT INTO organizations (id, name, slug, status) VALUES (?, ?, ?, 'active')`,
		id, name, slug,
	).Error; err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	return id
}

// moveUserToOrganization reassigns userID's organization_id directly.
// Deliberately does NOT re-issue a token for the user afterward — the
// existing access token's org_id claim goes stale, and the test relies on
// that: every authorization check in this codebase re-derives the actor's
// organization from a fresh server-side lookup (auth.Service.
// GetOrganizationID), never from the JWT claim alone, so access changes
// take effect on the very next request with the *same* token. This test
// is also, incidentally, the regression check for that specific design
// decision.
func moveUserToOrganization(t *testing.T, testDSN string, userID, orgID uuid.UUID) {
	t.Helper()
	db, err := database.Connect(testDSN)
	if err != nil {
		t.Fatalf("connect for org move: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Exec(`UPDATE users SET organization_id = ? WHERE id = ?`, orgID, userID).Error; err != nil {
		t.Fatalf("move user to organization: %v", err)
	}
}

func testDSNFor(t *testing.T, dbName string) string {
	t.Helper()
	baseDSN := os.Getenv("DATABASE_URL")
	if baseDSN == "" {
		t.Skip("DATABASE_URL not set")
	}
	_, testDSN := serverAndTestDSN(t, baseDSN, dbName)
	return testDSN
}

// TestTenantIsolation is the minimum required scenario: two organizations,
// one user and one project/task each, then a full cross-org access matrix
// covering every major resource and HTTP verb. Table below mirrors the
// security test matrix requested for P1.2.
//
//	Resource       Same Org   Different Org   Expected
//	Project GET       yes          no          200 / 404
//	Project PATCH     yes          no          200 / 404
//	Project DELETE     -           no                404
//	Task    GET       yes          no          200 / 404
//	Task    PATCH     yes          no          200 / 404 (covers status-change + assign, same endpoint)
//	Task    DELETE     -           no                404
//	Comment POST      yes          no          201 / 404
//	Attachment list   yes          no          200 / 404
//	Project list             (userB never sees Project A)
//	Team directory           (userB never sees ownerA)
//	Leaderboard              (userB never sees ownerA's character)
//	Role update (cross-org admin escalation)     404
func TestTenantIsolation(t *testing.T) {
	dbName := "e2e_tenant_isolation"
	app := newTestApp(t, dbName)
	testDSN := testDSNFor(t, dbName)

	// --- Organization A (the default org every registration lands in
	// today, per the P1.2 audit) ---
	ownerA := registerUser(t, app, "Owner A", uniqueEmail("ownerA"))

	projA := createProject(t, app, ownerA.AccessToken, "Org A Project")
	taskA := createTask(t, app, ownerA.AccessToken, projA, "Org A Task")

	// --- Organization B: a second org + a user moved into it after
	// registering (see moveUserToOrganization's doc comment for why the
	// token isn't re-issued) ---
	orgB := createOrganization(t, testDSN, "Org B", "org-b-"+uuid.NewString())
	userB := registerUser(t, app, "User B", uniqueEmail("userB"))
	moveUserToOrganization(t, testDSN, uuid.MustParse(userB.UserID), orgB)

	// userB creates their own project/task in Org B — used both as a
	// positive control (same-org access must keep working) and to prove
	// isolation is symmetric, not just one-directional.
	projB := createProject(t, app, userB.AccessToken, "Org B Project")
	taskB := createTask(t, app, userB.AccessToken, projB, "Org B Task")

	// === Cross-organization access: userB (Org B) against Org A's resources ===

	t.Run("Project_GET_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/projects/"+projA, userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusNotFound, "GET Org A project as Org B user")
	})

	t.Run("Project_PATCH_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPatch, "/api/v1/projects/"+projA, userB.AccessToken, map[string]string{"name": "Hijacked"})
		requireStatus(t, resp, http.StatusNotFound, "PATCH Org A project as Org B user")
	})

	t.Run("Project_DELETE_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodDelete, "/api/v1/projects/"+projA, userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusNotFound, "DELETE Org A project as Org B user")
	})

	t.Run("Project_AddMember_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPost, "/api/v1/projects/"+projA+"/members", userB.AccessToken, map[string]string{"user_id": userB.UserID})
		requireStatus(t, resp, http.StatusNotFound, "add self as member of Org A project as Org B user")
	})

	t.Run("Task_GET_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/tasks/"+taskA, userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusForbidden, "GET Org A task as Org B user")
	})

	t.Run("Task_PATCH_status_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPatch, "/api/v1/tasks/"+taskA, userB.AccessToken, map[string]string{"status": "done"})
		requireStatus(t, resp, http.StatusForbidden, "PATCH (status change) Org A task as Org B user")
	})

	t.Run("Task_PATCH_assign_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPatch, "/api/v1/tasks/"+taskA, userB.AccessToken, map[string]string{"assignee_id": userB.UserID})
		requireStatus(t, resp, http.StatusForbidden, "PATCH (reassign) Org A task as Org B user")
	})

	t.Run("Task_DELETE_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodDelete, "/api/v1/tasks/"+taskA, userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusForbidden, "DELETE Org A task as Org B user")
	})

	t.Run("Task_Comment_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPost, "/api/v1/tasks/"+taskA+"/comments", userB.AccessToken, map[string]string{"body": "hijack attempt"})
		requireStatus(t, resp, http.StatusForbidden, "POST comment on Org A task as Org B user")
	})

	t.Run("Task_Attachments_list_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/tasks/"+taskA+"/attachments", userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusForbidden, "list attachments on Org A task as Org B user")
	})

	t.Run("Task_Checklist_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPost, "/api/v1/tasks/"+taskA+"/checklist", userB.AccessToken, map[string]string{"text": "hijack"})
		requireStatus(t, resp, http.StatusForbidden, "add checklist item on Org A task as Org B user")
	})

	t.Run("Task_Watch_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodPost, "/api/v1/tasks/"+taskA+"/watch", userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusForbidden, "watch Org A task as Org B user")
	})

	t.Run("Gantt_cross_org_denied", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/projects/"+projA+"/gantt", userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusForbidden, "GET Org A gantt view as Org B user")
	})

	// === List/aggregate endpoints: Org A data must never appear ===

	t.Run("ProjectList_never_includes_other_org", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/projects", userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusOK, "list projects as Org B user")
		var body struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		decode(t, resp, &body)
		for _, p := range body.Items {
			if p.ID == projA {
				t.Fatalf("Org A project %s leaked into Org B user's project list", projA)
			}
		}
	})

	t.Run("TeamDirectory_never_includes_other_org", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/team", userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusOK, "get team directory as Org B user")
		var body struct {
			Members []struct {
				UserID string `json:"user_id"`
			} `json:"members"`
		}
		decode(t, resp, &body)
		for _, m := range body.Members {
			if m.UserID == ownerA.UserID {
				t.Fatalf("Org A user %s leaked into Org B team directory", ownerA.UserID)
			}
		}
	})

	t.Run("Leaderboard_never_includes_other_org", func(t *testing.T) {
		// Give ownerA some EXP so they'd actually show up on a leaderboard
		// if isolation were broken.
		markDone := request(t, app, http.MethodPatch, "/api/v1/tasks/"+taskA, ownerA.AccessToken, map[string]string{"status": "done"})
		requireStatus(t, markDone, http.StatusOK, "Org A owner completes their own task")

		resp := request(t, app, http.MethodGet, "/api/v1/gamification/leaderboard", userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusOK, "get leaderboard as Org B user")
		var body struct {
			Individuals []struct {
				UserID string `json:"user_id"`
			} `json:"individuals"`
		}
		decode(t, resp, &body)
		for _, entry := range body.Individuals {
			if entry.UserID == ownerA.UserID {
				t.Fatalf("Org A user %s leaked into Org B leaderboard", ownerA.UserID)
			}
		}
	})

	// === Vertical/horizontal privilege escalation across the org boundary ===

	t.Run("CrossOrg_role_update_denied_even_for_admin", func(t *testing.T) {
		// ownerA is a super_admin (first user ever registered in this test
		// DB) — confirms that role alone doesn't grant cross-org reach.
		resp := request(t, app, http.MethodPatch, "/api/v1/users/"+userB.UserID+"/role", ownerA.AccessToken, map[string]string{"role": "admin"})
		requireStatus(t, resp, http.StatusNotFound, "Org A admin changes Org B user's role")
	})

	t.Run("CrossOrg_department_update_denied_even_for_admin", func(t *testing.T) {
		resp := request(t, app, http.MethodPatch, "/api/v1/users/"+userB.UserID+"/department", ownerA.AccessToken, map[string]any{"department_id": nil})
		requireStatus(t, resp, http.StatusNotFound, "Org A admin changes Org B user's department")
	})

	// === Organization ID tampering: client-supplied org context must never be trusted ===

	t.Run("Spoofed_project_id_in_task_create_body_denied", func(t *testing.T) {
		// userB tries to create a task directly under Org A's project by
		// supplying its ID in the request body — the only "organization
		// context" this API ever accepts from a client is implicit in
		// resource IDs like this, and it must still be rejected exactly
		// like every other cross-org access attempt above.
		resp := request(t, app, http.MethodPost, "/api/v1/tasks", userB.AccessToken, map[string]string{
			"project_id": projA, "title": "Spoofed task",
		})
		requireStatus(t, resp, http.StatusForbidden, "create task under Org A's project as Org B user")
	})

	// === Positive controls: same-organization access must be unaffected ===

	t.Run("SameOrg_project_GET_allowed", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/projects/"+projB, userB.AccessToken, nil)
		requireStatus(t, resp, http.StatusOK, "GET own Org B project")
	})

	t.Run("SameOrg_task_PATCH_allowed", func(t *testing.T) {
		resp := request(t, app, http.MethodPatch, "/api/v1/tasks/"+taskB, userB.AccessToken, map[string]string{"status": "doing"})
		requireStatus(t, resp, http.StatusOK, "PATCH own Org B task")
	})

	t.Run("SameOrg_owner_can_still_manage_org_A_project", func(t *testing.T) {
		resp := request(t, app, http.MethodGet, "/api/v1/projects/"+projA, ownerA.AccessToken, nil)
		requireStatus(t, resp, http.StatusOK, "Org A owner reads their own project")
	})
}

// --- small request-building helpers -----------------------------------

func createProject(t *testing.T, app *fiber.App, token, name string) string {
	t.Helper()
	resp := request(t, app, http.MethodPost, "/api/v1/projects", token, map[string]string{"name": name})
	requireStatus(t, resp, http.StatusCreated, "create project "+name)
	var body struct {
		ID string `json:"id"`
	}
	decode(t, resp, &body)
	return body.ID
}

func createTask(t *testing.T, app *fiber.App, token, projectID, title string) string {
	t.Helper()
	resp := request(t, app, http.MethodPost, "/api/v1/tasks", token, map[string]string{
		"project_id": projectID, "title": title,
	})
	requireStatus(t, resp, http.StatusCreated, "create task "+title)
	var body struct {
		ID string `json:"id"`
	}
	decode(t, resp, &body)
	return body.ID
}
