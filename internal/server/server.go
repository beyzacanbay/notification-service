package server

import (
	"github.com/beyzacanbay/notification-service/internal/handler"

	"github.com/gofiber/fiber/v2"
)

func NewServer() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Notification System",
		ServerHeader: "Fiber",
	})

	healthHandler := handler.NewHealthHandler()
	app.Get("/health", healthHandler.Liveness)
	app.Get("/ready", healthHandler.Readiness)

	notificationHandler := handler.NewNotificationHandler()

	api := app.Group("/api/v1")
	notificationHandler.RegisterRoutes(api.Group("/notifications"))

	return app
}
