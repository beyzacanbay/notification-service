package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type postgresNotificationRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationRepo(pool *pgxpool.Pool) NotificationRepository {
	return &postgresNotificationRepo{pool: pool}
}

func (r *postgresNotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, channel, recipient, content, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		n.ID, n.Channel, n.Recipient, n.Content, n.Priority, n.Status, n.CreatedAt, n.UpdatedAt)

	return err
}

func (r *postgresNotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	n := &model.Notification{}

	err := r.pool.QueryRow(ctx, `
		SELECT id, channel, recipient, content, priority, status, created_at, updated_at
		FROM notifications WHERE id = $1`, id).Scan(
		&n.ID, &n.Channel, &n.Recipient, &n.Content, &n.Priority, &n.Status, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return n, nil
}

func (r *postgresNotificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE notifications SET status = $1 WHERE id = $2",
		status, id)
	return err
}
