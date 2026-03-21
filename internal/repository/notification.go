package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
)

var ErrNotFound = errors.New("notification not found")

type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	CreateBatch(ctx context.Context, notifications []*model.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.Notification, error)
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*model.Notification, error)
	List(ctx context.Context, filter *dto.ListNotificationsRequest) ([]*model.Notification, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status, lastError *string) error
	IncrementAttempt(ctx context.Context, id uuid.UUID, lastError string) error
	MarkSent(ctx context.Context, id uuid.UUID) error
	GetMetrics(ctx context.Context) (*dto.MetricsResponse, error)
}
