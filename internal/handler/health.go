package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/beyzacanbay/notification-service/internal/dto"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(dto.HealthResponse{Status: "ok"})
}

func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	return c.JSON(dto.HealthResponse{Status: "ok"})
}
