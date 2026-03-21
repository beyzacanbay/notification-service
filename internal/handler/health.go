package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/dto"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Liveness godoc
// @Summary Health check
// @Tags Health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Router /health [get]
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(dto.HealthResponse{Status: "ok"})
}

// Readiness godoc
// @Summary Readiness check
// @Description Check if the service and its dependencies are ready
// @Tags Health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Failure 503 {object} dto.HealthResponse
// @Router /ready [get]
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
	defer cancel()

	services := make(map[string]string)
	healthy := true

	if err := h.db.Ping(ctx); err != nil {
		services["postgres"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		services["postgres"] = "healthy"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
		healthy = false
	} else {
		services["redis"] = "healthy"
	}

	status := fiber.StatusOK
	statusText := "ok"
	if !healthy {
		status = fiber.StatusServiceUnavailable
		statusText = "degraded"
	}

	return c.Status(status).JSON(dto.HealthResponse{
		Status:   statusText,
		Services: services,
	})
}
