package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/service"
	"github.com/beyzacanbay/notification-service/internal/validator"
)

type NotificationHandler struct {
	svc         *service.NotificationService
	templateSvc *service.TemplateService
}

func NewNotificationHandler(svc *service.NotificationService, templateSvc *service.TemplateService) *NotificationHandler {
	return &NotificationHandler{svc: svc, templateSvc: templateSvc}
}

func (h *NotificationHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.Create)
	r.Post("/batch", h.CreateBatch)
	r.Post("/from-template", h.SendFromTemplate)
	r.Get("/", h.List)
	r.Get("/batch/:batchId/status", h.GetBatchStatus)
	r.Get("/:id", h.GetByID)
	r.Get("/:id/status", h.GetStatus)
	r.Patch("/:id/cancel", h.Cancel)
}

// Create godoc
// @Summary Create a notification
// @Tags Notifications
// @Accept json
// @Produce json
// @Param notification body dto.CreateNotificationRequest true "Notification request"
// @Success 202 {object} dto.NotificationResponse
// @Failure 400 {object} dto.ErrorResponse
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

	idempotencyKey := c.Get("Idempotency-Key")

	n, err := h.svc.Create(c.UserContext(), &req, idempotencyKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to create notification",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(dto.NotificationResponse{Notification: *n})
}

// CreateBatch godoc
// @Summary Create a batch of notifications
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

	resp, err := h.svc.CreateBatch(c.UserContext(), &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "failed to create batch",
			Details: err.Error(),
		})
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

	n, err := h.svc.GetByID(c.UserContext(), id)
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

	n, err := h.svc.GetByID(c.UserContext(), id)
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
// @Router /api/v1/notifications/{id}/cancel [patch]
func (h *NotificationHandler) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid notification ID"})
	}

	if err := h.svc.Cancel(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: err.Error()})
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

	counts, total, err := h.svc.GetBatchStatus(c.UserContext(), batchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to get batch",
			Details: err.Error(),
		})
	}

	if total == 0 {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "batch not found"})
	}

	return c.JSON(fiber.Map{
		"batch_id": batchID,
		"total":    total,
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

	data, total, totalPages, err := h.svc.List(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to list notifications",
			Details: err.Error(),
		})
	}

	return c.JSON(dto.PaginatedResponse{
		Data:       data,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalItems: total,
		TotalPages: totalPages,
	})
}

// SendFromTemplate godoc
// @Summary Send notification from template
// @Description Render a template with variables and create a notification
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body dto.SendFromTemplateRequest true "Template send request"
// @Success 202 {object} dto.NotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/notifications/from-template [post]
func (h *NotificationHandler) SendFromTemplate(c *fiber.Ctx) error {
	var req dto.SendFromTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
	}

	if req.TemplateID == "" || req.Recipient == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "template_id and recipient are required",
		})
	}

	templateID, err := uuid.Parse(req.TemplateID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid template_id"})
	}

	tmpl, err := h.templateSvc.GetByID(c.UserContext(), templateID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "template not found"})
	}

	content, err := h.templateSvc.RenderContent(tmpl, req.Params)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "failed to render template",
			Details: err.Error(),
		})
	}

	notifReq := &dto.CreateNotificationRequest{
		Channel:   tmpl.Channel,
		Recipient: req.Recipient,
		Content:   content,
		Priority:  req.Priority,
	}

	n, err := h.svc.Create(c.UserContext(), notifReq, c.Get("Idempotency-Key"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to create notification",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(dto.NotificationResponse{Notification: *n})
}
