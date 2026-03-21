package handler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"text/template"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

type TemplateHandler struct {
	templateRepo repository.TemplateRepository
	notifRepo    repository.NotificationRepository
	producer     Enqueuer
}

func NewTemplateHandler(templateRepo repository.TemplateRepository, notifRepo repository.NotificationRepository, producer Enqueuer) *TemplateHandler {
	return &TemplateHandler{templateRepo: templateRepo, notifRepo: notifRepo, producer: producer}
}

func (h *TemplateHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/:id", h.GetByID)
	r.Put("/:id", h.Update)
	r.Delete("/:id", h.Delete)
}

// Create godoc
// @Summary Create a template
// @Tags Templates
// @Accept json
// @Produce json
// @Param template body dto.CreateTemplateRequest true "Template"
// @Success 201 {object} model.Template
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/templates [post]
func (h *TemplateHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
	}

	if req.Name == "" || req.ContentTemplate == "" || !req.Channel.IsValid() {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "name, channel and content_template are required",
		})
	}

	tmpl := &model.Template{
		Name:            req.Name,
		Channel:         req.Channel,
		ContentTemplate: req.ContentTemplate,
	}

	if err := h.templateRepo.Create(c.UserContext(), tmpl); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to create template",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(tmpl)
}

// List godoc
// @Summary List all templates
// @Tags Templates
// @Produce json
// @Success 200 {array} model.Template
// @Router /api/v1/templates [get]
func (h *TemplateHandler) List(c *fiber.Ctx) error {
	templates, err := h.templateRepo.List(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to list templates",
			Details: err.Error(),
		})
	}
	return c.JSON(templates)
}

// GetByID godoc
// @Summary Get template by ID
// @Tags Templates
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} model.Template
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/templates/{id} [get]
func (h *TemplateHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid template ID"})
	}

	tmpl, err := h.templateRepo.GetByID(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "template not found"})
	}

	return c.JSON(tmpl)
}

// Update godoc
// @Summary Update a template
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param template body dto.CreateTemplateRequest true "Template"
// @Success 200 {object} model.Template
// @Router /api/v1/templates/{id} [put]
func (h *TemplateHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid template ID"})
	}

	var req dto.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid request body",
			Details: err.Error(),
		})
	}

	tmpl := &model.Template{
		ID:              id,
		Name:            req.Name,
		Channel:         req.Channel,
		ContentTemplate: req.ContentTemplate,
	}

	if err := h.templateRepo.Update(c.UserContext(), tmpl); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to update template",
			Details: err.Error(),
		})
	}

	return c.JSON(tmpl)
}

// Delete godoc
// @Summary Delete a template
// @Tags Templates
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/templates/{id} [delete]
func (h *TemplateHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "invalid template ID"})
	}

	if err := h.templateRepo.Delete(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to delete template",
			Details: err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "template deleted"})
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
func (h *TemplateHandler) SendFromTemplate(c *fiber.Ctx) error {
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

	tmpl, err := h.templateRepo.GetByID(c.UserContext(), templateID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Error: "template not found"})
	}

	// Render content template with variables
	content, err := renderTemplate(tmpl.ContentTemplate, req.Params)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "failed to render template",
			Details: err.Error(),
		})
	}

	priority := model.PriorityNormal
	if req.Priority != nil {
		priority = *req.Priority
	}

	// Idempotency
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", tmpl.Channel, req.Recipient, content)))
	idempotencyKey := fmt.Sprintf("%x", hash[:16])

	existing, err := h.notifRepo.GetByIdempotencyKey(c.UserContext(), idempotencyKey)
	if err == nil && existing != nil {
		return c.Status(fiber.StatusAccepted).JSON(dto.NotificationResponse{Notification: *existing})
	}

	n := &model.Notification{
		ID:             uuid.New(),
		IdempotencyKey: idempotencyKey,
		Channel:        tmpl.Channel,
		Recipient:      req.Recipient,
		Content:        content,
		Priority:       priority,
		Status:         model.StatusPending,
		MaxAttempts:    3,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.notifRepo.Create(c.UserContext(), n); err != nil {
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

func renderTemplate(tmplStr string, params map[string]interface{}) (string, error) {
	t, err := template.New("tmpl").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}
	return buf.String(), nil
}
