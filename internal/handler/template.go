package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/service"
)

type TemplateHandler struct {
	svc *service.TemplateService
}

func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
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

	tmpl, err := h.svc.Create(c.UserContext(), &req)
	if err != nil {
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
	templates, err := h.svc.List(c.UserContext())
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

	tmpl, err := h.svc.GetByID(c.UserContext(), id)
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

	tmpl, err := h.svc.Update(c.UserContext(), id, &req)
	if err != nil {
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

	if err := h.svc.Delete(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "failed to delete template",
			Details: err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "template deleted"})
}
