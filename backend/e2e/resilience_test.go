//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestPanicRecovery_ReturnsInternalServerError proves the recover
// middleware registered in bootstrap.New (P0.2) is actually wired into the
// real app, not just present in a unit test that reimplements the pattern:
// a handler that panics must produce a 500 response, and — critically —
// must not take the rest of the process down with it.
func TestPanicRecovery_ReturnsInternalServerError(t *testing.T) {
	app := newTestApp(t, "e2e_panic_recovery")

	app.Get("/__test/panic", func(c *fiber.Ctx) error {
		panic("boom: intentional panic for recover-middleware verification")
	})

	resp := request(t, app, http.MethodGet, "/__test/panic", "", nil)
	requireStatus(t, resp, http.StatusInternalServerError, "GET /__test/panic")

	// The real assertion: the app must still be alive and serving other
	// requests immediately afterward. A crashed process would fail this
	// (and every other test in the same run) rather than return 500 above.
	registerUser(t, app, "Panic Survivor", uniqueEmail("panic-survivor"))
}

// TestAuthRateLimit_BlocksAfterThreshold proves the stricter, auth-specific
// rate limiter (P0.2) actually trips: hammering /auth/login past
// RateLimitAuthMax (5, per testConfig) must eventually return 429, not just
// keep returning 401s for bad credentials forever.
func TestAuthRateLimit_BlocksAfterThreshold(t *testing.T) {
	app := newTestApp(t, "e2e_rate_limit")

	email := uniqueEmail("ratelimit")
	loginAttempt := map[string]string{"email": email, "password": "wrong-password"}

	var sawTooManyRequests bool
	var lastStatus int
	const attempts = 20 // well past the configured limit of 5
	for i := 0; i < attempts; i++ {
		resp := request(t, app, http.MethodPost, "/api/v1/auth/login", "", loginAttempt)
		lastStatus = resp.StatusCode
		if resp.StatusCode == http.StatusTooManyRequests {
			sawTooManyRequests = true
			break
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("login attempt %d: expected 401 (bad credentials) or 429 (rate limited), got %d (%s)", i, resp.StatusCode, resp.Error)
		}
	}

	if !sawTooManyRequests {
		t.Fatalf("expected a 429 within %d login attempts (limit is 5/window), last status was %d", attempts, lastStatus)
	}
}
