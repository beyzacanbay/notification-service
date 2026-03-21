package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type Producer struct {
	client *redis.Client
}

func NewProducer(client *redis.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Enqueue(ctx context.Context, id uuid.UUID, priority model.Priority, channel model.Channel) error {
	queueName := model.QueueName(priority, channel)
	score := float64(time.Now().UnixMilli())

	err := p.client.ZAdd(ctx, queueName, redis.Z{
		Score:  score,
		Member: id.String(),
	}).Err()
	if err != nil {
		return fmt.Errorf("enqueue notification %s to %s: %w", id, queueName, err)
	}
	return nil
}

func (p *Producer) EnqueueWithDelay(ctx context.Context, id uuid.UUID, priority model.Priority, channel model.Channel, delay time.Duration) error {
	queueName := model.QueueName(priority, channel)
	score := float64(time.Now().Add(delay).UnixMilli())

	err := p.client.ZAdd(ctx, queueName, redis.Z{
		Score:  score,
		Member: id.String(),
	}).Err()
	if err != nil {
		return fmt.Errorf("enqueue with delay notification %s to %s: %w", id, queueName, err)
	}
	return nil
}

func (p *Producer) GetQueueDepth(ctx context.Context) (map[string]int64, error) {
	depths := make(map[string]int64)
	for _, name := range model.AllQueueNames() {
		count, err := p.client.ZCard(ctx, name).Result()
		if err != nil {
			return nil, err
		}
		if count > 0 {
			depths[name] = count
		}
	}
	return depths, nil
}
