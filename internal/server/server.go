package server

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/handler"
	"github.com/beyzacanbay/notification-service/internal/middleware"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/service"
	ws "github.com/beyzacanbay/notification-service/internal/websocket"
)

func NewServer(db *pgxpool.Pool, redisClient *redis.Client, notifRepo repository.NotificationRepository, templateRepo repository.TemplateRepository, producer *queue.Producer, wsHub *ws.Hub, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Notification System",
		ServerHeader: "Fiber",
	})

	// Middleware
	app.Use(middleware.Recovery(logger))
	app.Use(middleware.Tracing("notification-api"))
	app.Use(middleware.CorrelationID())
	app.Use(middleware.RequestLogger(logger))

	// Health & metrics
	healthHandler := handler.NewHealthHandler(db, redisClient)
	app.Get("/health", healthHandler.Liveness)
	app.Get("/ready", healthHandler.Readiness)

	metricsHandler := handler.NewMetricsHandler(notifRepo, producer)
	app.Get("/metrics", metricsHandler.Metrics)

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// WebSocket
	app.Use("/ws", ws.UpgradeMiddleware())
	app.Get("/ws/notifications", wsHub.Handler())

	// Services
	notifSvc := service.NewNotificationService(notifRepo, producer, redisClient)
	templateSvc := service.NewTemplateService(templateRepo)

	// API v1
	api := app.Group("/api/v1")

	templateHandler := handler.NewTemplateHandler(templateSvc)
	templateHandler.RegisterRoutes(api.Group("/templates"))

	notificationHandler := handler.NewNotificationHandler(notifSvc, templateSvc)
	notificationHandler.RegisterRoutes(api.Group("/notifications"))

	return app
}
