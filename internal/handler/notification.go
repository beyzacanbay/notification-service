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
	r.Post("/batch", h.CreateBatch)
	r.Get("/:id", h.GetByID)
	r.Get("/:id/status", h.GetStatus)
	r.Patch("/:id/cancel", h.Cancel)
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

func (h *NotificationHandler) CreateBatch(c *fiber.Ctx) error {
	var req dto.BatchCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
	}

	if len(req.Notifications) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "at least one notification is required",
		})
	}
	if len(req.Notifications) > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "batch size must not exceed 1000",
		})
	}

	batchID := uuid.New()
	now := time.Now()
	var notifications []*model.Notification

	for _, r := range req.Notifications {
		if r.Channel == "" || r.Recipient == "" || r.Content == "" {
			continue
		}
		if !r.Channel.IsValid() {
			continue
		}

		priority := model.PriorityNormal
		if r.Priority != nil && r.Priority.IsValid() {
			priority = *r.Priority
		}

		notifications = append(notifications, &model.Notification{
			ID:        uuid.New(),
			BatchID:   &batchID,
			Channel:   r.Channel,
			Recipient: r.Recipient,
			Content:   r.Content,
			Priority:  priority,
			Status:    model.StatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if len(notifications) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "no valid notifications in batch",
		})
	}

	if err := h.repo.CreateBatch(c.UserContext(), notifications); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to save batch",
			Details: err.Error(),
		})
	}

	for _, n := range notifications {
		h.producer.Enqueue(c.UserContext(), n.ID)
	}

	resp := dto.BatchCreateResponse{
		BatchID:      batchID.String(),
		TotalCreated: len(notifications),
	}
	for _, n := range notifications {
		resp.Notifications = append(resp.Notifications, dto.NotificationResponse{Notification: *n})
	}

	return c.Status(fiber.StatusAccepted).JSON(resp)
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

func (h *NotificationHandler) GetStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid notification ID"})
	}

	n, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "notification not found"})
	}

	return c.JSON(fiber.Map{
		"id":     n.ID,
		"status": n.Status,
	})
}

func (h *NotificationHandler) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid notification ID"})
	}

	n, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "notification not found"})
	}

	if n.Status != model.StatusPending && n.Status != model.StatusQueued {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "only pending or queued notifications can be cancelled",
		})
	}

	if err := h.repo.UpdateStatus(c.UserContext(), id, model.StatusCancelled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to cancel notification",
			Details: err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "notification cancelled"})
}
