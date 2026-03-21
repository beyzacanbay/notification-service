package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

type Enqueuer interface {
	Enqueue(ctx context.Context, id uuid.UUID) error
}

type NotificationHandler struct {
	repo     repository.NotificationRepository
	producer Enqueuer
}

func NewNotificationHandler(repo repository.NotificationRepository, producer Enqueuer) *NotificationHandler {
	return &NotificationHandler{repo: repo, producer: producer}
}

func (h *NotificationHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.Create)
	r.Get("/:id", h.GetByID)
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

	n := &model.Notification{
		ID:        uuid.New(),
		Channel:   req.Channel,
		Recipient: req.Recipient,
		Content:   req.Content,
		Priority:  priority,
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.repo.Create(c.UserContext(), n); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to save notification",
			Details: err.Error(),
		})
	}

	if err := h.producer.Enqueue(c.UserContext(), n.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to enqueue notification",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(dto.NotificationResponse{Notification: *n})
}

func (h *NotificationHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid notification ID"})
	}

	n, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "notification not found"})
	}

	return c.JSON(dto.NotificationResponse{Notification: *n})
}
