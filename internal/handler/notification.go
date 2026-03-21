package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
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

	n := model.Notification{
		ID:        uuid.New(),
		Channel:   req.Channel,
		Recipient: req.Recipient,
		Content:   req.Content,
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
	}

	return c.Status(fiber.StatusAccepted).JSON(dto.NotificationResponse{Notification: n})
}
