package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/model"
)

const QueueName = "notifications"

type Producer struct {
	client *redis.Client
}

func NewProducer(client *redis.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Enqueue(ctx context.Context, n *model.Notification) error {
	data, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal notification %s: %w", n.ID, err)
	}

	score := float64(time.Now().UnixMilli())

	err = p.client.ZAdd(ctx, QueueName, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
	if err != nil {
		return fmt.Errorf("enqueue notification %s: %w", n.ID, err)
	}

	return nil
}
