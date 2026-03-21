package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type Consumer struct {
	client *redis.Client
	queues []string
}

func NewConsumer(client *redis.Client, queues []string) *Consumer {
	if len(queues) == 0 {
		queues = model.AllQueueNames()
	}
	return &Consumer{client: client, queues: queues}
}

// Dequeue tries queues in priority order.
// Fetches the lowest-score item that is ready (score <= now), then removes it.
func (c *Consumer) Dequeue(ctx context.Context) (string, error) {
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)

	for _, q := range c.queues {
		// Get the first item with score <= now
		results, err := c.client.ZRangeByScore(ctx, q, &redis.ZRangeBy{
			Min:    "-inf",
			Max:    now,
			Offset: 0,
			Count:  1,
		}).Result()
		if err != nil {
			return "", fmt.Errorf("dequeue from %s: %w", q, err)
		}
		if len(results) == 0 {
			continue
		}

		member := results[0]

		// Remove it — if someone else already took it, ZRem returns 0
		removed, err := c.client.ZRem(ctx, q, member).Result()
		if err != nil {
			return "", fmt.Errorf("remove from %s: %w", q, err)
		}
		if removed == 0 {
			// Another worker grabbed it, try next queue
			continue
		}

		return member, nil
	}

	return "", nil
}
