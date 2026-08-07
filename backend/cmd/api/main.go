package main

import (
	"log"

	"github.com/khomkrittk/taskworked/backend/internal/bootstrap"
	"github.com/khomkrittk/taskworked/backend/internal/config"
	"github.com/khomkrittk/taskworked/backend/internal/platform/cache"
	"github.com/khomkrittk/taskworked/backend/internal/platform/database"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := bootstrap.Migrate(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	redisClient := cache.Connect(cfg.RedisAddr, cfg.RedisPassword)

	app := bootstrap.New(cfg, db, redisClient)
	defer app.StopSubscriber()
	defer app.CronScheduler.Stop()

	log.Fatal(app.Fiber.Listen(":" + cfg.Port))
}
