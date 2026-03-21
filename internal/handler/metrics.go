package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

type MetricsHandler struct {
	repo     repository.NotificationRepository
	producer *queue.Producer
}

func NewMetricsHandler(repo repository.NotificationRepository, producer *queue.Producer) *MetricsHandler {
	return &MetricsHandler{repo: repo, producer: producer}
}

func (h *MetricsHandler) Metrics(c *fiber.Ctx) error {
	metrics, err := h.repo.GetMetrics(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to get metrics",
			Details: err.Error(),
		})
	}

	depths, err := h.producer.GetQueueDepth(c.UserContext())
	if err == nil {
		metrics.QueueDepth = depths
	}

	return c.JSON(metrics)
}
