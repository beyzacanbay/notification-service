package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const QueueName = "notifications"

type Producer struct {
	client *redis.Client
}

func NewProducer(client *redis.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Enqueue(ctx context.Context, id uuid.UUID) error {
	score := float64(time.Now().UnixMilli())

	err := p.client.ZAdd(ctx, QueueName, redis.Z{
		Score:  score,
		Member: id.String(),
	}).Err()
	if err != nil {
		return fmt.Errorf("enqueue notification %s: %w", id, err)
	}

	return nil
}

func (p *Producer) GetQueueDepth(ctx context.Context) (int64, error) {
	return p.client.ZCard(ctx, QueueName).Result()
}
