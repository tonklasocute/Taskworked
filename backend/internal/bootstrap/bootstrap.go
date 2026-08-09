// Package bootstrap wires every module's repository/service/handler graph
// into a ready-to-serve Fiber app. It exists so cmd/api/main.go and the
// E2E test suite (backend/e2e) build the exact same dependency graph
// instead of the tests maintaining a second, driftable copy of the wiring.
package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/khomkrittk/taskworked/backend/docs"
	"github.com/khomkrittk/taskworked/backend/internal/config"
	appmiddleware "github.com/khomkrittk/taskworked/backend/internal/middleware"
	"github.com/khomkrittk/taskworked/backend/internal/modules/actionplan"
	"github.com/khomkrittk/taskworked/backend/internal/modules/ai"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/gamification"
	"github.com/khomkrittk/taskworked/backend/internal/modules/notification"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
	"github.com/khomkrittk/taskworked/backend/internal/modules/report"
	"github.com/khomkrittk/taskworked/backend/internal/modules/task"
	"github.com/khomkrittk/taskworked/backend/internal/modules/team"
	"github.com/khomkrittk/taskworked/backend/internal/platform/storage"
	"github.com/khomkrittk/taskworked/backend/internal/realtime"
)

// projectEventPublisher bridges realtime.Publisher's named PublishToProject
// method to task.EventPublisher's generic Publish signature, so the task
// package's interface (and its tests) never had to change shape.
type projectEventPublisher struct{ p *realtime.Publisher }

func (a *projectEventPublisher) Publish(ctx context.Context, projectID string, event any) error {
	return a.p.PublishToProject(ctx, projectID, event)
}

// taskNotifierHolder breaks the construction cycle between task.Service
// (needs a Notifier to raise assignment notifications) and
// notification.Service (needs task.Service for its digest queries): the
// holder is handed to task.NewService empty, then populated with the real
// notification.Service once that's constructed further down.
type taskNotifierHolder struct{ notifier task.Notifier }

func (h *taskNotifierHolder) NotifyAssignment(ctx context.Context, assigneeID uuid.UUID, taskID, taskTitle string) error {
	if h.notifier == nil {
		return nil
	}
	return h.notifier.NotifyAssignment(ctx, assigneeID, taskID, taskTitle)
}

func (h *taskNotifierHolder) NotifyComment(ctx context.Context, userID uuid.UUID, taskID, taskTitle, commentBody string) error {
	if h.notifier == nil {
		return nil
	}
	return h.notifier.NotifyComment(ctx, userID, taskID, taskTitle, commentBody)
}

// taskGamifierHolder is the same late-binding fix as taskNotifierHolder,
// for the same reason: gamification.Service needs task.Service (for its
// perfect-week badge check), and task.Service needs a Gamifier.
type taskGamifierHolder struct{ gamifier task.Gamifier }

func (h *taskGamifierHolder) OnTaskCompleted(ctx context.Context, userID uuid.UUID, completedAt time.Time, dueDate *time.Time, priority task.Priority) error {
	if h.gamifier == nil {
		return nil
	}
	return h.gamifier.OnTaskCompleted(ctx, userID, completedAt, dueDate, priority)
}

// swaggerUIPage loads Swagger UI from a CDN (no npm bundle needed on the
// Go side) pointed at the embedded spec served next to it.
const swaggerUIPage = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Taskworked API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: "/docs/openapi.yaml",
        dom_id: "#swagger-ui",
      });
    };
  </script>
</body>
</html>`

// Migrate runs AutoMigrate for every module's models. Callers own the
// database connection (main connects to the configured DATABASE_URL; the
// E2E suite connects to its own throwaway test database).
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&auth.User{},
		&project.Project{}, &project.Member{},
		&task.Task{}, &task.ChecklistItem{}, &task.Tag{}, &task.Dependency{},
		&task.Comment{}, &task.Attachment{}, &task.Watcher{},
		&actionplan.Goal{}, &actionplan.Milestone{},
		&team.Department{},
		&notification.Notification{}, &notification.Preference{},
		&gamification.Character{}, &gamification.Badge{}, &gamification.MissionProgress{},
	)
}

// App bundles the wired Fiber app with the background jobs main() starts
// alongside it (cron digests, the Redis pub/sub subscriber feeding
// WebSocket broadcasts) so callers can start and stop them together.
type App struct {
	Fiber          *fiber.App
	CronScheduler  *cron.Cron
	StopSubscriber context.CancelFunc
}

// New wires every module's repository/service/handler graph into a
// ready-to-serve Fiber app. db and redisClient are connected by the
// caller (not here) so main() and the E2E suite can point at different
// instances; likewise Migrate is the caller's responsibility, run before
// New if the schema isn't already up to date.
func New(cfg *config.Config, db *gorm.DB, redisClient *redis.Client) *App {
	// MinIO is a soft dependency: attachments degrade to a clear
	// per-request error (see task.Service.Storage) rather than the whole
	// API failing to start when object storage is unreachable.
	var taskStorage task.Storage
	if minioClient, err := storage.Connect(context.Background(), cfg.MinioEndpoint, cfg.MinioPublicEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL); err != nil {
		log.Printf("warning: failed to connect to MinIO, attachments will be disabled: %v", err)
	} else {
		taskStorage = minioClient
	}

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
	go realtime.RunSubscriber(subscriberCtx, redisClient, hub)

	taskRepo := task.NewRepository(db)
	notifierHolder := &taskNotifierHolder{}
	gamifierHolder := &taskGamifierHolder{}
	taskService := task.NewService(taskRepo, projectService, authService, &projectEventPublisher{publisher}, notifierHolder, gamifierHolder, taskStorage)
	taskHandler := task.NewHandler(taskService)

	actionPlanRepo := actionplan.NewRepository(db)
	actionPlanService := actionplan.NewService(actionPlanRepo, projectService, taskService)
	actionPlanHandler := actionplan.NewHandler(actionPlanService)

	reportService := report.NewService(projectService, taskService)
	reportHandler := report.NewHandler(reportService)

	teamRepo := team.NewRepository(db)
	teamService := team.NewService(teamRepo, authService, taskService, &team.RedisPresence{Client: redisClient})
	teamHandler := team.NewHandler(teamService)

	gamificationRepo := gamification.NewRepository(db)
	gamificationService := gamification.NewService(gamificationRepo, taskService, teamService)
	gamificationHandler := gamification.NewHandler(gamificationService)
	gamifierHolder.gamifier = gamificationService

	notificationRepo := notification.NewRepository(db)
	emailSender := &notification.SMTPSender{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		Username: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
	}
	// publisher already has a PublishToUser method matching
	// notification.Broadcaster's signature — no adapter needed here,
	// unlike task.EventPublisher (see projectEventPublisher above).
	notificationService := notification.NewService(notificationRepo, authService, taskService, publisher, emailSender, notification.NewLineNotifySender())
	notificationHandler := notification.NewHandler(notificationService)

	aiClient := ai.NewAnthropicClient(cfg.AnthropicAPIKey)
	aiService := ai.NewService(aiClient, projectService, taskService, reportService)
	aiHandler := ai.NewHandler(aiService)
	notifierHolder.notifier = notificationService

	cronScheduler := cron.New()
	digestCtx := context.Background()
	cronScheduler.AddFunc("0 8 * * *", func() { notificationService.SendDailyDigests(digestCtx) })
	cronScheduler.AddFunc("0 18 * * *", func() { notificationService.SendEndOfDayDigests(digestCtx) })
	cronScheduler.AddFunc("30 7 * * 1", func() { notificationService.SendWeeklyDigests(digestCtx) })
	cronScheduler.AddFunc("0 7 1 * *", func() { notificationService.SendMonthlyDigests(digestCtx) })
	cronScheduler.Start()

	requireAuth := appmiddleware.RequireAuth(tokens)
	trackPresence := appmiddleware.TrackPresence(redisClient)

	app := fiber.New(fiber.Config{
		// Above Fiber's 4MB default so task attachment uploads (capped at
		// 20MB in the handler) aren't rejected before they even get there.
		BodyLimit: 25 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "internal server error"})
		},
	})
	// recover must be the outermost middleware: without it, a panic
	// anywhere in a handler (nil deref, bad type assertion, etc.) crashes
	// the whole process and drops every other in-flight request, not just
	// the one that panicked.
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Global rate limit protects every route from gross abuse; the auth
	// group additionally gets a much stricter limit below (see
	// RegisterRoutes call) since register/login/refresh are the
	// specifically credential-guessable endpoints. Both are in-memory
	// (per-process) — correct for today's single-instance deployment; if
	// the API is ever run as multiple replicas, swap in a shared store
	// (e.g. a Redis-backed fiber/storage/redis, since Redis is already a
	// dependency here) so limits are enforced across instances, not reset
	// per-replica.
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitGlobalMax,
		Expiration: cfg.RateLimitGlobalWindow,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"success": false, "error": "too many requests"})
		},
	}))
	authLimiter := limiter.New(limiter.Config{
		Max:        cfg.RateLimitAuthMax,
		Expiration: cfg.RateLimitAuthWindow,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"success": false, "error": "too many requests"})
		},
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API docs are public, same as /health — they describe the API's
	// shape, not its data, matching common practice for a docs endpoint.
	app.Get("/docs/openapi.yaml", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/yaml")
		return c.Send(docs.OpenAPISpec)
	})
	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.SendString(swaggerUIPage)
	})

	api := app.Group("/api/v1")
	authHandler.RegisterRoutes(api, requireAuth, authLimiter)
	authHandler.RegisterUserRoutes(api.Group("", requireAuth, trackPresence))
	projectHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	taskHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	actionPlanHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	reportHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	teamHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	notificationHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	gamificationHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	aiHandler.RegisterRoutes(api.Group("", requireAuth, trackPresence))
	realtime.RegisterRoute(app, tokens, projectService, hub)

	return &App{Fiber: app, CronScheduler: cronScheduler, StopSubscriber: stopSubscriber}
}
