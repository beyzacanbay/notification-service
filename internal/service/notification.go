package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/dto"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/repository"
	"github.com/beyzacanbay/notification-service/internal/validator"
)

const idempotencyTTL = 24 * time.Hour

type Enqueuer interface {
	Enqueue(ctx context.Context, id uuid.UUID, priority model.Priority, channel model.Channel) error
}

type NotificationService struct {
	repo     repository.NotificationRepository
	producer Enqueuer
	redis    *redis.Client
}

func NewNotificationService(repo repository.NotificationRepository, producer Enqueuer, redisClient *redis.Client) *NotificationService {
	return &NotificationService{repo: repo, producer: producer, redis: redisClient}
}

func (s *NotificationService) Create(ctx context.Context, req *dto.CreateNotificationRequest, idempotencyKey string) (*model.Notification, error) {
	// Idempotency: explicit key from header, fallback to auto-hash
	idemKey := idempotencyKey
	if idemKey == "" {
		idemKey = idempotencyHash(req.Channel, req.Recipient, req.Content)
	}
	idemKey = "idempotency:" + idemKey
	if s.redis != nil {
		if existingID, err := s.redis.Get(ctx, idemKey).Result(); err == nil {
			id, _ := uuid.Parse(existingID)
			if existing, err := s.repo.GetByID(ctx, id); err == nil {
				return existing, nil
			}
		}
	}

	priority := model.PriorityNormal
	if req.Priority != nil {
		priority = *req.Priority
	}

	n := &model.Notification{
		ID:          uuid.New(),
		Channel:     req.Channel,
		Recipient:   req.Recipient,
		Content:     req.Content,
		Priority:    priority,
		Status:      model.StatusPending,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("save notification: %w", err)
	}

	// Cache idempotency key with 24h TTL
	if s.redis != nil {
		s.redis.Set(ctx, idemKey, n.ID.String(), idempotencyTTL)
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
		_ = s.producer.Enqueue(ctx, n.ID, n.Priority, n.Channel)
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

func idempotencyHash(channel model.Channel, recipient, content string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", channel, recipient, content)))
	return fmt.Sprintf("%x", hash[:16])
}
