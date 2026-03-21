package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DLQName = "dlq:notifications"

type DLQ struct {
	client *redis.Client
}

func NewDLQ(client *redis.Client) *DLQ {
	return &DLQ{client: client}
}

func (d *DLQ) Push(ctx context.Context, id uuid.UUID, reason string) {
	d.client.HSet(ctx, DLQName, id.String(), reason)
}

func (d *DLQ) List(ctx context.Context) (map[string]string, error) {
	result, err := d.client.HGetAll(ctx, DLQName).Result()
	if err != nil {
		return nil, fmt.Errorf("list DLQ: %w", err)
	}
	return result, nil
}
