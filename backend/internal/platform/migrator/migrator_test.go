package migrator

import (
	"net/url"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// requireDSN skips the test if no real Postgres is reachable — same
// pattern backend/e2e uses, so this test is a no-op in the plain `go test
// ./...` CI job (no DB service there) and only runs where DATABASE_URL
// actually points at something, e.g. `make e2e`-style local runs.
func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping migrator test that needs a real Postgres")
	}
	return dsn
}

// freshDB drops and recreates a throwaway database on the same server
// DATABASE_URL points at, returning a DSN for it. Mirrors
// backend/e2e/e2e_test.go's resetDatabase/serverAndTestDSN helpers.
func freshDB(t *testing.T, baseDSN, name string) string {
	t.Helper()
	u, err := url.Parse(baseDSN)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	admin := *u
	admin.Path = "/postgres"

	db, err := gorm.Open(postgres.Open(admin.String()), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to postgres server: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	db.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`, name)
	if err := db.Exec("DROP DATABASE IF EXISTS " + name).Error; err != nil {
		t.Fatalf("drop database %s: %v", name, err)
	}
	if err := db.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatalf("create database %s: %v", name, err)
	}

	target := *u
	target.Path = "/" + name
	return target.String()
}

func TestVersion_NoMigrationsAppliedYet(t *testing.T) {
	baseDSN := requireDSN(t)
	dsn := freshDB(t, baseDSN, "migrator_test_unmigrated")

	_, _, ok, err := Version(dsn)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for a database with no migrations applied yet")
	}
}

func TestUp_AppliesAllMigrations_MatchesLatestVersion(t *testing.T) {
	baseDSN := requireDSN(t)
	dsn := freshDB(t, baseDSN, "migrator_test_up")

	if err := Up(dsn); err != nil {
		t.Fatalf("Up: %v", err)
	}

	version, dirty, ok, err := Version(dsn)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true after Up")
	}
	if dirty {
		t.Fatal("expected dirty=false after a clean Up")
	}

	latest, err := LatestVersion()
	if err != nil {
		t.Fatalf("LatestVersion: %v", err)
	}
	if version != latest {
		t.Fatalf("expected version %d (latest embedded) after Up, got %d", latest, version)
	}
}

func TestUp_IsIdempotent(t *testing.T) {
	baseDSN := requireDSN(t)
	dsn := freshDB(t, baseDSN, "migrator_test_idempotent")

	if err := Up(dsn); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	if err := Up(dsn); err != nil {
		t.Fatalf("second Up (should be a no-op, not an error): %v", err)
	}
}
