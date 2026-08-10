// Package migrator runs this project's versioned SQL migrations
// (backend/migrations) against a Postgres database via golang-migrate.
// This replaces the old GORM AutoMigrate call in bootstrap.Migrate — see
// docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md
// for why: AutoMigrate applies additive DDL at every process boot with no
// review step, no down-migration, and no changelog. From here on, schema
// changes ship as new numbered files in backend/migrations, never
// AutoMigrate.
package migrator

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers the "postgres://" driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/khomkrittk/taskworked/backend/migrations"
)

func newMigrate(dsn string) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("init migrate: %w", err)
	}
	return m, nil
}

// Up applies every pending migration, in order. Idempotent: calling it
// against an already-up-to-date database is a no-op.
func Up(dsn string) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Force marks dsn as being at the given migration version without running
// any migration SQL. This is the standard technique for adopting versioned
// migrations against a database whose baseline schema already exists from
// elsewhere (here: one previously managed by GORM AutoMigrate, matching
// exactly what migration 0001 creates from scratch) — running 0001's
// CREATE TABLE statements against such a database would fail with
// "relation already exists". Use once, by hand, when cutting an existing
// pre-migration database over to this system (see
// docs/operations/database-migrations.md); never as part of routine
// deploys, and never on a database that doesn't already match the
// baseline migration's schema exactly.
func Force(dsn string, version int) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Force(version)
}

// Version reports the migration version currently applied to dsn, and
// whether the last migration attempt left the schema "dirty" (a migration
// started but didn't finish — needs manual intervention, never silently
// retried). ok is false if no migrations have ever been applied to this
// database yet.
func Version(dsn string) (version uint, dirty bool, ok bool, err error) {
	m, err := newMigrate(dsn)
	if err != nil {
		return 0, false, false, err
	}
	defer m.Close()

	v, d, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, err
	}
	return v, d, true, nil
}

// LatestVersion returns the highest migration version embedded in this
// binary (regardless of what any particular database has applied) — used
// by cmd/api's startup check to confirm a database is fully caught up
// before serving traffic, rather than failing confusingly on the first
// query that touches a column/table that doesn't exist yet.
func LatestVersion() (uint, error) {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return 0, fmt.Errorf("load embedded migrations: %w", err)
	}
	defer sourceDriver.Close()

	version, err := sourceDriver.First()
	if err != nil {
		return 0, fmt.Errorf("read first migration version: %w", err)
	}
	for {
		next, err := sourceDriver.Next(version)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("walk migrations: %w", err)
		}
		version = next
	}
	return version, nil
}
