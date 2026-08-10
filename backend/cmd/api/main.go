package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/khomkrittk/taskworked/backend/internal/bootstrap"
	"github.com/khomkrittk/taskworked/backend/internal/config"
	"github.com/khomkrittk/taskworked/backend/internal/platform/cache"
	"github.com/khomkrittk/taskworked/backend/internal/platform/database"
	"github.com/khomkrittk/taskworked/backend/internal/platform/migrator"
)

func main() {
	cfg := config.Load()

	// Migrations are a separate one-shot step (see cmd/migrate and
	// docker-compose.yml's migrate service), not run here — this only
	// verifies the schema is actually caught up before serving traffic,
	// so a skipped/failed migration step fails loudly at boot instead of
	// confusingly on the first query that touches a missing column.
	if err := verifyMigrationsApplied(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration check failed: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	redisClient := cache.Connect(cfg.RedisAddr, cfg.RedisPassword)

	app := bootstrap.New(cfg, db, redisClient)
	defer app.StopSubscriber()
	defer app.CronScheduler.Stop()

	log.Fatal(app.Fiber.Listen(":" + cfg.Port))
}

// verifyMigrationsApplied fails fast if the database hasn't had migrations
// run against it at all, is mid-way through a failed migration ("dirty"),
// or is behind the version this binary's embedded migrations expect. Any
// of these would otherwise surface later as a confusing runtime error on
// whatever query happens to touch the missing schema first.
func verifyMigrationsApplied(dsn string) error {
	version, dirty, ok, err := migrator.Version(dsn)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no migrations have been applied to this database — run the migrate step first (see cmd/migrate)")
	}
	if dirty {
		return fmt.Errorf("database schema is dirty at migration version %d — a previous migration failed partway and needs manual fixing before the API can start safely", version)
	}

	latest, err := migrator.LatestVersion()
	if err != nil {
		return err
	}
	if version < latest {
		return fmt.Errorf("database is at migration version %d but this binary expects version %d — run the migrate step first", version, latest)
	}
	return nil
}
