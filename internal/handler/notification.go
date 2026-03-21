package handler

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

type Enqueuer interface {
	Enqueue(ctx context.Context, id uuid.UUID, priority model.Priority, channel model.Channel) error
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
	r.Get("/", h.List)
	r.Get("/batch/:batchId/status", h.GetBatchStatus)
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
		ID:          uuid.New(),
		Channel:     req.Channel,
		Recipient:   req.Recipient,
		Content:     req.Content,
		Priority:    priority,
		Status:      model.StatusPending,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repo.Create(c.UserContext(), n); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to save notification",
			Details: err.Error(),
		})
	}

	if err := h.producer.Enqueue(c.UserContext(), n.ID, n.Priority, n.Channel); err != nil {
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
			ID:          uuid.New(),
			BatchID:     &batchID,
			Channel:     r.Channel,
			Recipient:   r.Recipient,
			Content:     r.Content,
			Priority:    priority,
			Status:      model.StatusPending,
			MaxAttempts: 3,
			CreatedAt:   now,
			UpdatedAt:   now,
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
		h.producer.Enqueue(c.UserContext(), n.ID, n.Priority, n.Channel)
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

	if err := h.repo.UpdateStatus(c.UserContext(), id, model.StatusCancelled, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to cancel notification",
			Details: err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "notification cancelled"})
}

func (h *NotificationHandler) GetBatchStatus(c *fiber.Ctx) error {
	batchID, err := uuid.Parse(c.Params("batchId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid batch ID"})
	}

	notifications, err := h.repo.GetByBatchID(c.UserContext(), batchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to get batch",
			Details: err.Error(),
		})
	}

	if len(notifications) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "batch not found"})
	}

	counts := make(map[string]int)
	for _, n := range notifications {
		counts[string(n.Status)]++
	}

	return c.JSON(fiber.Map{
		"batch_id": batchID,
		"total":    len(notifications),
		"statuses": counts,
	})
}

func (h *NotificationHandler) List(c *fiber.Ctx) error {
	filter := &dto.ListNotificationsRequest{}

	if s := c.Query("status"); s != "" {
		status := model.Status(s)
		filter.Status = &status
	}
	if ch := c.Query("channel"); ch != "" {
		channel := model.Channel(ch)
		filter.Channel = &channel
	}
	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse(time.RFC3339, sd); err == nil {
			filter.StartDate = &t
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse(time.RFC3339, ed); err == nil {
			filter.EndDate = &t
		}
	}
	if p := c.Query("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if pp := c.Query("per_page"); pp != "" {
		filter.PerPage, _ = strconv.Atoi(pp)
	}

	notifications, total, err := h.repo.List(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to list notifications",
			Details: err.Error(),
		})
	}

	var data []dto.NotificationResponse
	for _, n := range notifications {
		data = append(data, dto.NotificationResponse{Notification: *n})
	}

	return c.JSON(dto.PaginatedResponse{
		Data:       data,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(filter.PerPage))),
	})
}
