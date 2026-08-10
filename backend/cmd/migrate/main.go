// Command migrate applies every pending versioned SQL migration
// (backend/migrations) to DATABASE_URL and exits. Run this as a one-shot
// step before starting the api binary — see docker-compose.yml's migrate
// service. Deliberately doesn't share config.Load() with cmd/api: this
// only needs DATABASE_URL, not the JWT/SMTP/MinIO secrets the full app
// config requires, so a misconfigured unrelated secret can't block a
// migration run.
//
// -force-baseline is a one-time cutover flag for a database that already
// has the exact schema migration 0001 would create from scratch (e.g. one
// previously managed by GORM AutoMigrate) — it records "version 1 applied"
// without running 0001's SQL, which would otherwise fail with "relation
// already exists". See docs/operations/database-migrations.md. Never use
// this on a fresh or already-migrated database.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/khomkrittk/taskworked/backend/internal/platform/migrator"
)

func main() {
	_ = godotenv.Load() // best-effort, same as config.Load() — picks up backend/.env for local runs

	forceBaseline := flag.Bool("force-baseline", false, "mark migration 1 as applied without running it (one-time cutover for a pre-existing AutoMigrate-managed database)")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if *forceBaseline {
		if err := migrator.Force(dsn, 1); err != nil {
			log.Fatalf("force-baseline failed: %v", err)
		}
		log.Print("database marked as migration version 1 (baseline) without running it")
	}

	if err := migrator.Up(dsn); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	version, dirty, ok, err := migrator.Version(dsn)
	if err != nil {
		log.Fatalf("migration applied but version check failed: %v", err)
	}
	if !ok {
		log.Fatal("migration ran but no version is recorded — this should not happen")
	}
	log.Printf("database is now at migration version %d (dirty=%v)", version, dirty)
}
