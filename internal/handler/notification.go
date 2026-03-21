package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/queue"
)

type NotificationHandler struct {
	producer *queue.Producer
}

func NewNotificationHandler(producer *queue.Producer) *NotificationHandler {
	return &NotificationHandler{producer: producer}
}

func (h *NotificationHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.Create)
}

func (h *NotificationHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
	}

	if req.Channel == "" || req.Recipient == "" || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "channel, recipient and content are required",
		})
	}

	if !req.Channel.IsValid() {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "invalid channel, must be one of: sms, email, push",
		})
	}

	priority := model.PriorityNormal
	if req.Priority != nil {
		if !req.Priority.IsValid() {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error: "invalid priority, must be 0 (high), 1 (normal) or 2 (low)",
			})
		}
		priority = *req.Priority
	}

	n := model.Notification{
		ID:        uuid.New(),
		Channel:   req.Channel,
		Recipient: req.Recipient,
		Content:   req.Content,
		Priority:  priority,
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := h.producer.Enqueue(c.UserContext(), &n); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to enqueue notification",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(dto.NotificationResponse{Notification: n})
}
