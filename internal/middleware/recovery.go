package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func Recovery(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"error", r,
					"stack", string(debug.Stack()),
					"correlation_id", GetCorrelationID(c.UserContext()),
				)
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
		}()
		return c.Next()
	}
}
