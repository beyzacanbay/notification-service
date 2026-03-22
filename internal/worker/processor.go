package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/beyzacanbay/notification-service/internal/delivery"
	"github.com/beyzacanbay/notification-service/internal/model"
	"github.com/beyzacanbay/notification-service/internal/queue"
	"github.com/beyzacanbay/notification-service/internal/ratelimiter"
	"github.com/beyzacanbay/notification-service/internal/repository"
	ws "github.com/beyzacanbay/notification-service/internal/websocket"
)

type Processor struct {
	repo        repository.NotificationRepository
	providers   *delivery.ProviderRegistry
	rateLimiter *ratelimiter.RateLimiter
	producer    *queue.Producer
	dlq         *queue.DLQ
	retryCfg    delivery.RetryConfig
	hub         *ws.Hub
	logger      *slog.Logger
}

func NewProcessor(
	repo repository.NotificationRepository,
	providers *delivery.ProviderRegistry,
	rl *ratelimiter.RateLimiter,
	producer *queue.Producer,
	dlq *queue.DLQ,
	retryCfg delivery.RetryConfig,
	hub *ws.Hub,
	logger *slog.Logger,
) *Processor {
	return &Processor{
		repo:        repo,
		providers:   providers,
		rateLimiter: rl,
		producer:    producer,
		dlq:         dlq,
		retryCfg:    retryCfg,
		hub:         hub,
		logger:      logger,
	}
}

func (p *Processor) Process(ctx context.Context, notificationID string) error {
	id, err := uuid.Parse(notificationID)
	if err != nil {
		return fmt.Errorf("parse notification ID: %w", err)
	}

	log := p.logger.With("notification_id", id.String())

	n, err := p.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get notification: %w", err)
	}

	// Skip terminal states
	if n.Status == model.StatusSent || n.Status == model.StatusCancelled || n.Status == model.StatusFailed {
		log.Info("skipping notification in terminal state", "status", n.Status)
		return nil
	}

	// Resolve provider for channel
	provider, err := p.providers.Get(n.Channel)
	if err != nil {
		return fmt.Errorf("resolve provider: %w", err)
	}

	// Rate limit check
	allowed, err := p.rateLimiter.Allow(ctx, string(n.Channel))
	if err != nil {
		log.Error("rate limiter error", "error", err)
		return p.requeue(ctx, n, "rate limiter error")
	}
	if !allowed {
		log.Debug("rate limited, re-enqueueing")
		return p.requeue(ctx, n, "rate limited")
	}

	// Mark as processing
	if err := p.repo.UpdateStatus(ctx, id, model.StatusProcessing, nil); err != nil {
		return fmt.Errorf("update status to processing: %w", err)
	}

	// Attempt delivery
	result, err := provider.Send(ctx, n)
	if err != nil {
		log.Warn("delivery failed", "error", err, "channel", n.Channel, "attempt", n.AttemptCount+1)
		return p.handleFailure(ctx, n, err)
	}

	// Success
	log.Info("notification delivered", "message_id", result.MessageID, "channel", n.Channel)
	if err := p.repo.MarkSent(ctx, id); err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	p.hub.BroadcastStatus(id.String(), string(model.StatusSent))
	return nil
}

func (p *Processor) requeue(ctx context.Context, n *model.Notification, reason string) error {
	if err := p.repo.UpdateStatus(ctx, n.ID, model.StatusQueued, &reason); err != nil {
		return err
	}
	delay := delivery.CalculateBackoff(0, delivery.RetryConfig{
		BaseDelay: p.retryCfg.BaseDelay / 10,
		MaxDelay:  p.retryCfg.BaseDelay,
	})
	return p.producer.EnqueueWithDelay(ctx, n.ID, n.Priority, n.Channel, delay)
}

func (p *Processor) handleFailure(ctx context.Context, n *model.Notification, deliveryErr error) error {
	errMsg := deliveryErr.Error()

	// Permanent error — don't retry
	if !delivery.IsRetryable(deliveryErr) {
		p.logger.Warn("permanent delivery error, sending to DLQ", "notification_id", n.ID, "error", errMsg)
		if err := p.repo.UpdateStatus(ctx, n.ID, model.StatusFailed, &errMsg); err != nil {
			return err
		}
		p.dlq.Push(ctx, n.ID, errMsg)
		p.hub.BroadcastStatus(n.ID.String(), string(model.StatusFailed))
		return nil
	}

	if err := p.repo.IncrementAttempt(ctx, n.ID, errMsg); err != nil {
		return fmt.Errorf("increment attempt: %w", err)
	}

	nextAttempt := n.AttemptCount + 1
	if nextAttempt >= n.MaxAttempts {
		p.logger.Warn("max retries reached, sending to DLQ", "notification_id", n.ID, "attempts", nextAttempt)
		if err := p.repo.UpdateStatus(ctx, n.ID, model.StatusFailed, &errMsg); err != nil {
			return err
		}
		p.dlq.Push(ctx, n.ID, errMsg)
		p.hub.BroadcastStatus(n.ID.String(), string(model.StatusFailed))
		return nil
	}

	// Retryable — re-enqueue with exponential backoff
	delay := delivery.CalculateBackoff(nextAttempt, p.retryCfg)
	p.logger.Info("retrying notification", "notification_id", n.ID, "attempt", nextAttempt, "delay", delay)

	if err := p.repo.UpdateStatus(ctx, n.ID, model.StatusQueued, &errMsg); err != nil {
		return err
	}
	return p.producer.EnqueueWithDelay(ctx, n.ID, n.Priority, n.Channel, delay)
}
