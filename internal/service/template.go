package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
)

type TemplateService struct {
	repo repository.TemplateRepository
}

func NewTemplateService(repo repository.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) Create(ctx context.Context, req *dto.CreateTemplateRequest) (*model.Template, error) {
	tmpl := &model.Template{
		Name:            req.Name,
		Channel:         req.Channel,
		ContentTemplate: req.ContentTemplate,
	}

	if err := s.repo.Create(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return tmpl, nil
}

func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TemplateService) List(ctx context.Context) ([]*model.Template, error) {
	return s.repo.List(ctx)
}

func (s *TemplateService) Update(ctx context.Context, id uuid.UUID, req *dto.CreateTemplateRequest) (*model.Template, error) {
	tmpl := &model.Template{
		ID:              id,
		Name:            req.Name,
		Channel:         req.Channel,
		ContentTemplate: req.ContentTemplate,
	}

	if err := s.repo.Update(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return tmpl, nil
}

func (s *TemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *TemplateService) RenderContent(tmpl *model.Template, params map[string]interface{}) (string, error) {
	t, err := template.New("content").Parse(tmpl.ContentTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
