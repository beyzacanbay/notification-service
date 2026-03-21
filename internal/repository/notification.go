package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/model"
)

var ErrNotFound = errors.New("notification not found")

type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	CreateBatch(ctx context.Context, notifications []*model.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status) error
}
