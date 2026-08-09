//go:build e2e

// Package e2e exercises the real HTTP stack — full dependency-injection
// wiring (internal/bootstrap), real Postgres, real Redis — instead of the
// per-module fakes the rest of the test suite uses. Unit tests catch
// service-logic bugs; this suite catches the bugs that only show up when
// the modules are actually wired together and hitting a real database
// (migrations, GORM query correctness, cross-module permission checks
// through the full HTTP path).
//
// Requires a build tag so `go test ./...` doesn't try to run it without
// Postgres/Redis available:
//
//	go test -tags e2e ./e2e/...   (or: make e2e)
//
// Configuration is read from the same env vars the app itself uses
// (DATABASE_URL, REDIS_ADDR, ...) — see testConfig below. Point
// DATABASE_URL at any reachable Postgres server; the suite creates its
// own throwaway databases there (dropped and recreated on every run, not
// cleaned up afterward — inspect them after a failure if you need to).
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/khomkrittk/taskworked/backend/internal/bootstrap"
	"github.com/khomkrittk/taskworked/backend/internal/config"
	"github.com/khomkrittk/taskworked/backend/internal/platform/database"
)

func TestMain(m *testing.M) {
	// Best-effort: picks up backend/.env for local runs. CI sets the
	// same env vars directly, so a missing file here is not an error.
	_ = godotenv.Load("../.env")
	os.Exit(m.Run())
}

// --- test app / database setup --------------------------------------------

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// serverAndTestDSN splits a Postgres URL into a maintenance-connection DSN
// (pointed at the "postgres" database, for issuing DROP/CREATE DATABASE)
// and a DSN for dbName on that same server.
func serverAndTestDSN(t *testing.T, baseDSN, dbName string) (serverDSN, testDSN string) {
	t.Helper()
	u, err := url.Parse(baseDSN)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	server := *u
	server.Path = "/postgres"
	test := *u
	test.Path = "/" + dbName
	return server.String(), test.String()
}

// resetDatabase drops and recreates dbName on the server baseDSN points
// at, so every run starts from an empty schema — most importantly for
// TestBootstrap_FirstUserIsSuperAdmin, which needs to be the very first
// thing that ever touches the users table.
func resetDatabase(t *testing.T, baseDSN, dbName string) {
	t.Helper()
	serverDSN, _ := serverAndTestDSN(t, baseDSN, dbName)

	admin, err := database.Connect(serverDSN)
	if err != nil {
		t.Fatalf("connect to postgres server: %v", err)
	}
	sqlDB, err := admin.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Terminate any connections left over from a previous (e.g. crashed)
	// run so DROP DATABASE doesn't block on them.
	admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`, dbName)
	if err := admin.Exec("DROP DATABASE IF EXISTS " + dbName).Error; err != nil {
		t.Fatalf("drop database %s: %v", dbName, err)
	}
	if err := admin.Exec("CREATE DATABASE " + dbName).Error; err != nil {
		t.Fatalf("create database %s: %v", dbName, err)
	}
}

func testConfig(dbDSN string) *config.Config {
	return &config.Config{
		DatabaseURL:         dbDSN,
		RedisAddr:           envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       os.Getenv("REDIS_PASSWORD"),
		JWTAccessSecret:     "e2e-test-access-secret",
		JWTRefreshSecret:    "e2e-test-refresh-secret",
		AccessTokenTTL:      15 * time.Minute,
		RefreshTokenTTL:     7 * 24 * time.Hour,
		MinioEndpoint:       envOr("MINIO_ENDPOINT", "localhost:9000"),
		MinioPublicEndpoint: envOr("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinioAccessKey:      os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:      os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:         "taskworked-e2e",

		// High enough that no existing scenario (a handful of requests per
		// test) ever trips it, but explicit rather than relying on
		// zero-value/library-default behavior — see resilience_test.go for
		// the test that deliberately exercises the auth-specific limit.
		RateLimitGlobalMax:    1000,
		RateLimitGlobalWindow: time.Minute,
		RateLimitAuthMax:      5,
		RateLimitAuthWindow:   time.Minute,
	}
}

// newTestApp resets dbName to an empty schema, migrates it, and wires up
// a real *fiber.App via internal/bootstrap — the same construction path
// cmd/api/main.go uses. dbName lets callers choose between a database
// dedicated to one test (for bootstrap-ordering-sensitive tests) and a
// shared one (for tests that just need a working, member-populated app).
func newTestApp(t *testing.T, dbName string) *fiber.App {
	t.Helper()
	baseDSN := os.Getenv("DATABASE_URL")
	if baseDSN == "" {
		t.Skip("DATABASE_URL not set — run via `make e2e` (see Makefile) or export DATABASE_URL/REDIS_ADDR yourself")
	}

	resetDatabase(t, baseDSN, dbName)
	_, testDSN := serverAndTestDSN(t, baseDSN, dbName)

	db, err := database.Connect(testDSN)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	if err := bootstrap.Migrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	cfg := testConfig(testDSN)
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("connect to redis at %s: %v", cfg.RedisAddr, err)
	}

	app := bootstrap.New(cfg, db, redisClient)
	t.Cleanup(func() {
		app.CronScheduler.Stop()
		app.StopSubscriber()
	})
	return app.Fiber
}

// --- HTTP helpers -----------------------------------------------------

type apiResponse struct {
	StatusCode int
	Success    bool
	Data       json.RawMessage
	Error      string
}

func request(t *testing.T, app *fiber.App, method, path, token string, body any) apiResponse {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("%s %s: request failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read response body: %v", method, path, err)
	}

	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   string          `json:"error"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("%s %s: decode response envelope %q: %v", method, path, raw, err)
		}
	}
	return apiResponse{StatusCode: resp.StatusCode, Success: env.Success, Data: env.Data, Error: env.Error}
}

func decode(t *testing.T, r apiResponse, out any) {
	t.Helper()
	if err := json.Unmarshal(r.Data, out); err != nil {
		t.Fatalf("decode response data: %v (raw: %s)", err, r.Data)
	}
}

func requireStatus(t *testing.T, r apiResponse, want int, action string) {
	t.Helper()
	if r.StatusCode != want {
		t.Fatalf("%s: expected status %d, got %d (%s)", action, want, r.StatusCode, r.Error)
	}
}

var emailCounter int64

// uniqueEmail keeps concurrently-run tests that share a database (see
// newTestApp) from colliding on "email already registered".
func uniqueEmail(prefix string) string {
	n := atomic.AddInt64(&emailCounter, 1)
	return fmt.Sprintf("%s-%d-%d@e2e.test", prefix, time.Now().UnixNano(), n)
}

type registeredUser struct {
	AccessToken string
	UserID      string
	Role        string
}

func registerUser(t *testing.T, app *fiber.App, name, email string) registeredUser {
	t.Helper()
	resp := request(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": name, "email": email, "password": "Password123!",
	})
	requireStatus(t, resp, http.StatusCreated, "register "+email)

	var body struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	decode(t, resp, &body)
	return registeredUser{AccessToken: body.AccessToken, UserID: body.User.ID, Role: body.User.Role}
}
