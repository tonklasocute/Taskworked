package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/khomkrittk/taskworked/backend/internal/config"
	appmiddleware "github.com/khomkrittk/taskworked/backend/internal/middleware"
	"github.com/khomkrittk/taskworked/backend/internal/modules/actionplan"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/report"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	"github.com/khomkrittk/taskworked/backend/internal/modules/team"
	"github.com/khomkrittk/taskworked/backend/internal/platform/cache"
	"github.com/khomkrittk/taskworked/backend/internal/platform/database"
	"github.com/khomkrittk/taskworked/backend/internal/realtime"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(
		&auth.User{},
		&project.Project{}, &project.Member{},
		&task.Task{}, &task.ChecklistItem{}, &task.Tag{}, &task.Dependency{},
		&actionplan.Goal{}, &actionplan.Milestone{},
		&team.Department{},
	); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	redisClient := cache.Connect(cfg.RedisAddr, cfg.RedisPassword)

	tokens := auth.NewTokenService(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, redisClient)
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, tokens)
	authHandler := auth.NewHandler(authService)

	projectRepo := project.NewRepository(db)
	projectService := project.NewService(projectRepo, authService)
	projectHandler := project.NewHandler(projectService)

	hub := realtime.NewHub()
	publisher := realtime.NewPublisher(redisClient)

	subscriberCtx, stopSubscriber := context.WithCancel(context.Background())
	defer stopSubscriber()
	go realtime.RunSubscriber(subscriberCtx, redisClient, hub)

	taskRepo := task.NewRepository(db)
	taskService := task.NewService(taskRepo, projectService, publisher)
	taskHandler := task.NewHandler(taskService)

	actionPlanRepo := actionplan.NewRepository(db)
	actionPlanService := actionplan.NewService(actionPlanRepo, projectService, taskService)
	actionPlanHandler := actionplan.NewHandler(actionPlanService)

	reportService := report.NewService(projectService, taskService)
	reportHandler := report.NewHandler(reportService)

	teamRepo := team.NewRepository(db)
	teamService := team.NewService(teamRepo, authService, taskService, &team.RedisPresence{Client: redisClient})
	teamHandler := team.NewHandler(teamService)

	requireAuth := appmiddleware.RequireAuth(tokens)
	trackPresence := appmiddleware.TrackPresence(redisClient)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "internal server error"})
		},
	})
	app.Use(logger.New())
	app.Use(cors.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/v1")
	authHandler.RegisterRoutes(api, requireAuth)
	authHandler.RegisterUserRoutes(api.Group("", requireAuth, trackPresence))
	projectHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	taskHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	actionPlanHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	reportHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	teamHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	realtime.RegisterRoute(app, tokens, projectService, hub)

	log.Fatal(app.Listen(":" + cfg.Port))
}
