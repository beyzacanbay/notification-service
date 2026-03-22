package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
)

type postgresNotificationRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationRepo(pool *pgxpool.Pool) NotificationRepository {
	return &postgresNotificationRepo{pool: pool}
}

const insertSQL = `
	INSERT INTO notifications (id, batch_id, channel, recipient, content, priority, status, scheduled_at, attempt_count, max_attempts, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

func (r *postgresNotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, insertSQL,
		n.ID, n.BatchID, n.Channel, n.Recipient, n.Content, n.Priority, n.Status, n.ScheduledAt, n.AttemptCount, n.MaxAttempts, n.CreatedAt, n.UpdatedAt)
	return err
}

func (r *postgresNotificationRepo) CreateBatch(ctx context.Context, notifications []*model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}
	now := time.Now()

	for _, n := range notifications {
		if n.ID == uuid.Nil {
			n.ID = uuid.New()
		}
		n.CreatedAt = now
		n.UpdatedAt = now

		batch.Queue(insertSQL,
			n.ID, n.BatchID, n.Channel, n.Recipient, n.Content, n.Priority, n.Status, n.ScheduledAt, n.AttemptCount, n.MaxAttempts, n.CreatedAt, n.UpdatedAt)
	}

	br := tx.SendBatch(ctx, batch)
	for range notifications {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("batch insert: %w", err)
		}
	}
	br.Close()

	return tx.Commit(ctx)
}

const selectColumns = `id, batch_id, channel, recipient, content, priority, status, scheduled_at, attempt_count, max_attempts, last_error, sent_at, created_at, updated_at`

func scanNotification(row pgx.Row) (*model.Notification, error) {
	n := &model.Notification{}
	err := row.Scan(&n.ID, &n.BatchID, &n.Channel, &n.Recipient, &n.Content, &n.Priority, &n.Status, &n.ScheduledAt, &n.AttemptCount, &n.MaxAttempts, &n.LastError, &n.SentAt, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func scanNotifications(rows pgx.Rows) ([]*model.Notification, error) {
	var notifications []*model.Notification
	for rows.Next() {
		n := &model.Notification{}
		if err := rows.Scan(&n.ID, &n.BatchID, &n.Channel, &n.Recipient, &n.Content, &n.Priority, &n.Status, &n.ScheduledAt, &n.AttemptCount, &n.MaxAttempts, &n.LastError, &n.SentAt, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *postgresNotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	row := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT %s FROM notifications WHERE id = $1", selectColumns), id)
	return scanNotification(row)
}

func (r *postgresNotificationRepo) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*model.Notification, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT %s FROM notifications WHERE batch_id = $1 ORDER BY created_at", selectColumns), batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotifications(rows)
}

func (r *postgresNotificationRepo) List(ctx context.Context, filter *dto.ListNotificationsRequest) ([]*model.Notification, int64, error) {
	filter.SetDefaults()

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Channel != nil {
		conditions = append(conditions, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *filter.Channel)
		argIdx++
	}
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf("SELECT %s FROM notifications %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		selectColumns, whereClause, argIdx, argIdx+1)
	args = append(args, filter.PerPage, filter.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	notifications, err := scanNotifications(rows)
	if err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

func (r *postgresNotificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status, lastError *string) error {
	_, err := r.pool.Exec(ctx, "UPDATE notifications SET status = $1, last_error = $2 WHERE id = $3", status, lastError, id)
	return err
}

func (r *postgresNotificationRepo) IncrementAttempt(ctx context.Context, id uuid.UUID, lastError string) error {
	_, err := r.pool.Exec(ctx, "UPDATE notifications SET attempt_count = attempt_count + 1, last_error = $1 WHERE id = $2", lastError, id)
	return err
}

func (r *postgresNotificationRepo) MarkSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "UPDATE notifications SET status = 'sent', sent_at = NOW() WHERE id = $1", id)
	return err
}

func (r *postgresNotificationRepo) GetMetrics(ctx context.Context) (*dto.MetricsResponse, error) {
	metrics := &dto.MetricsResponse{}

	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE status = 'sent'").Scan(&metrics.TotalSent)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE status = 'failed'").Scan(&metrics.TotalFailed)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE status IN ('pending', 'queued', 'processing')").Scan(&metrics.TotalPending)
	if err != nil {
		return nil, err
	}

	total := metrics.TotalSent + metrics.TotalFailed
	if total > 0 {
		metrics.SuccessRate = float64(metrics.TotalSent) / float64(total) * 100
		metrics.FailureRate = float64(metrics.TotalFailed) / float64(total) * 100
	}

	// Average latency: time between created_at and sent_at for delivered notifications
	var avgMs *float64
	err = r.pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (sent_at - created_at)) * 1000)
		FROM notifications WHERE status = 'sent' AND sent_at IS NOT NULL`).Scan(&avgMs)
	if err == nil && avgMs != nil {
		metrics.AvgLatencyMs = *avgMs
	}

	return metrics, nil
}
