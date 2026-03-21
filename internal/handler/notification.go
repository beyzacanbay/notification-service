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
	"github.com/beyzacanbay/notification-service/internal/validator"
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

// Create godoc
// @Summary Create a notification
// @Description Create a new notification request
// @Tags Notifications
// @Accept json
// @Produce json
// @Param notification body dto.CreateNotificationRequest true "Notification request"
// @Success 202 {object} dto.NotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/notifications [post]
func (h *NotificationHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
	}

	if errs := validator.ValidateCreateRequest(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation failed",
			Details: errs,
		})
	}

	priority := model.PriorityNormal
	if req.Priority != nil {
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

// CreateBatch godoc
// @Summary Create a batch of notifications
// @Description Create up to 1000 notifications in a single request
// @Tags Notifications
// @Accept json
// @Produce json
// @Param batch body dto.BatchCreateRequest true "Batch request"
// @Success 202 {object} dto.BatchCreateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/notifications/batch [post]
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
		if errs := validator.ValidateCreateRequest(&r); len(errs) > 0 {
			continue
		}

		priority := model.PriorityNormal
		if r.Priority != nil {
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

// GetByID godoc
// @Summary Get notification by ID
// @Tags Notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} dto.NotificationResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/notifications/{id} [get]
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

// GetStatus godoc
// @Summary Query notification status by ID
// @Tags Notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/notifications/{id}/status [get]
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

// Cancel godoc
// @Summary Cancel a pending or queued notification
// @Tags Notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/notifications/{id}/cancel [patch]
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

// GetBatchStatus godoc
// @Summary Get batch status summary
// @Tags Notifications
// @Produce json
// @Param batchId path string true "Batch ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/notifications/batch/{batchId}/status [get]
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

// List godoc
// @Summary List notifications with filtering and pagination
// @Tags Notifications
// @Produce json
// @Param status query string false "Filter by status"
// @Param channel query string false "Filter by channel"
// @Param start_date query string false "Start date (RFC3339)"
// @Param end_date query string false "End date (RFC3339)"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} dto.PaginatedResponse
// @Router /api/v1/notifications [get]
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
