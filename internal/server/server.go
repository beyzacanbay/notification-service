package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/handler"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

func NewServer(db *pgxpool.Pool, redisClient *redis.Client, repo repository.NotificationRepository, producer *queue.Producer) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Notification System",
		ServerHeader: "Fiber",
	})

	healthHandler := handler.NewHealthHandler(db, redisClient)
	app.Get("/health", healthHandler.Liveness)
	app.Get("/ready", healthHandler.Readiness)

	notificationHandler := handler.NewNotificationHandler(repo, producer)

	api := app.Group("/api/v1")
	notificationHandler.RegisterRoutes(api.Group("/notifications"))

	return app
}
