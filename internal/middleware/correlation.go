package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type contextKey string

const CorrelationIDKey contextKey = "correlation_id"

func CorrelationID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		correlationID := c.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		c.Locals(string(CorrelationIDKey), correlationID)
		c.Set("X-Correlation-ID", correlationID)

		ctx := context.WithValue(c.UserContext(), CorrelationIDKey, correlationID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}
