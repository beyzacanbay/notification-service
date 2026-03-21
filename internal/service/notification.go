package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/validator"
)

type Enqueuer interface {
	Enqueue(ctx context.Context, id uuid.UUID, priority model.Priority, channel model.Channel) error
}

type NotificationService struct {
	repo     repository.NotificationRepository
	producer Enqueuer
}

func NewNotificationService(repo repository.NotificationRepository, producer Enqueuer) *NotificationService {
	return &NotificationService{repo: repo, producer: producer}
}

func (s *NotificationService) Create(ctx context.Context, req *dto.CreateNotificationRequest, idempotencyKey string) (*model.Notification, error) {
	// Idempotency check
	if idempotencyKey != "" {
		existing, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	priority := model.PriorityNormal
	if req.Priority != nil {
		priority = *req.Priority
	}

	n := &model.Notification{
		ID:             uuid.New(),
		IdempotencyKey: idempotencyKey,
		Channel:        req.Channel,
		Recipient:      req.Recipient,
		Content:        req.Content,
		Priority:       priority,
		Status:         model.StatusPending,
		MaxAttempts:    3,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("save notification: %w", err)
	}

	if err := s.producer.Enqueue(ctx, n.ID, n.Priority, n.Channel); err != nil {
		return nil, fmt.Errorf("enqueue notification: %w", err)
	}

	return n, nil
}

func (s *NotificationService) CreateBatch(ctx context.Context, req *dto.BatchCreateRequest) (*dto.BatchCreateResponse, error) {
	batchID := uuid.New()
	now := time.Now()
	var notifications []*model.Notification
	var batchErrors []dto.BatchItemError

	for i, r := range req.Notifications {
		if errs := validator.ValidateCreateRequest(&r); len(errs) > 0 {
			batchErrors = append(batchErrors, dto.BatchItemError{
				Index:   i,
				Details: errs,
			})
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
		return nil, fmt.Errorf("no valid notifications in batch")
	}

	if err := s.repo.CreateBatch(ctx, notifications); err != nil {
		return nil, fmt.Errorf("save batch: %w", err)
	}

	for _, n := range notifications {
		s.producer.Enqueue(ctx, n.ID, n.Priority, n.Channel)
	}

	resp := &dto.BatchCreateResponse{
		BatchID:      batchID.String(),
		TotalCreated: len(notifications),
		TotalFailed:  len(batchErrors),
		Errors:       batchErrors,
	}
	for _, n := range notifications {
		resp.Notifications = append(resp.Notifications, dto.NotificationResponse{Notification: *n})
	}

	return resp, nil
}

func (s *NotificationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *NotificationService) Cancel(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("notification not found")
	}

	if n.Status != model.StatusPending && n.Status != model.StatusQueued {
		return fmt.Errorf("only pending or queued notifications can be cancelled")
	}

	return s.repo.UpdateStatus(ctx, id, model.StatusCancelled, nil)
}

func (s *NotificationService) GetBatchStatus(ctx context.Context, batchID uuid.UUID) (map[string]int, int, error) {
	notifications, err := s.repo.GetByBatchID(ctx, batchID)
	if err != nil {
		return nil, 0, err
	}

	counts := make(map[string]int)
	for _, n := range notifications {
		counts[string(n.Status)]++
	}

	return counts, len(notifications), nil
}

func (s *NotificationService) List(ctx context.Context, filter *dto.ListNotificationsRequest) ([]dto.NotificationResponse, int64, int, error) {
	filter.SetDefaults()

	notifications, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	var data []dto.NotificationResponse
	for _, n := range notifications {
		data = append(data, dto.NotificationResponse{Notification: *n})
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PerPage)))
	return data, total, totalPages, nil
}
