package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type TemplateRepository interface {
	Create(ctx context.Context, t *model.Template) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error)
	List(ctx context.Context) ([]*model.Template, error)
	Update(ctx context.Context, t *model.Template) error
	Delete(ctx context.Context, id uuid.UUID) error
}
