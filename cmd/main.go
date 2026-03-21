package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"notification_service/internal/handler"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Notification Service is running",
		})
	})

	app.Get("/health", handler.HealthCheck)

	log.Fatal(app.Listen(":8081"))
}
