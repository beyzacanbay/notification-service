package queue

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/beyzacanbay/notification-service/internal/model"
)

// PriorityWeight defines how many dequeues each priority gets per cycle.
type PriorityWeight struct {
	Priority model.Priority
	Weight   int
}

// DefaultWeights: high=6, normal=3, low=1 out of every 10 dequeues.
var DefaultWeights = []PriorityWeight{
	{Priority: model.PriorityHigh, Weight: 6},
	{Priority: model.PriorityNormal, Weight: 3},
	{Priority: model.PriorityLow, Weight: 1},
}

type Consumer struct {
	client   *redis.Client
	channels []model.Channel
	weights  []PriorityWeight

	mu       sync.Mutex
	schedule []model.Priority // precomputed dequeue order
	cursor   int
}

func NewConsumer(client *redis.Client, queues []string) *Consumer {
	_ = queues // kept for backward compat, weights replace queue ordering

	c := &Consumer{
		client:   client,
		channels: []model.Channel{model.ChannelSMS, model.ChannelEmail, model.ChannelPush},
		weights:  DefaultWeights,
	}
	c.schedule = c.buildSchedule()
	return c
}

// buildSchedule creates a repeating sequence based on weights.
// e.g. [high,high,high,high,high,high,normal,normal,normal,low]
func (c *Consumer) buildSchedule() []model.Priority {
	var schedule []model.Priority
	for _, w := range c.weights {
		for i := 0; i < w.Weight; i++ {
			schedule = append(schedule, w.Priority)
		}
	}
	return schedule
}

// nextPriority returns the next priority to dequeue from, advancing the cursor.
func (c *Consumer) nextPriority() model.Priority {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := c.schedule[c.cursor]
	c.cursor = (c.cursor + 1) % len(c.schedule)
	return p
}

// Dequeue picks a priority based on weighted fair schedule,
// then tries all channels for that priority.
// If the chosen priority is empty, falls back to other priorities.
func (c *Consumer) Dequeue(ctx context.Context) (string, error) {
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)
	primary := c.nextPriority()

	// Try primary priority first (all channels)
	if id, err := c.tryPriority(ctx, primary, now); err != nil {
		return "", err
	} else if id != "" {
		return id, nil
	}

	// Fallback: try other priorities in weight order
	for _, w := range c.weights {
		if w.Priority == primary {
			continue
		}
		if id, err := c.tryPriority(ctx, w.Priority, now); err != nil {
			return "", err
		} else if id != "" {
			return id, nil
		}
	}

	return "", nil
}

func (c *Consumer) tryPriority(ctx context.Context, priority model.Priority, now string) (string, error) {
	for _, ch := range c.channels {
		q := model.QueueName(priority, ch)

		results, err := c.client.ZRangeArgs(ctx, redis.ZRangeArgs{
			Key:     q,
			Start:   "-inf",
			Stop:    now,
			ByScore: true,
			Offset:  0,
			Count:   1,
		}).Result()
		if err != nil {
			return "", fmt.Errorf("dequeue from %s: %w", q, err)
		}
		if len(results) == 0 {
			continue
		}

		member := results[0]

		removed, err := c.client.ZRem(ctx, q, member).Result()
		if err != nil {
			return "", fmt.Errorf("remove from %s: %w", q, err)
		}
		if removed == 0 {
			continue
		}

		return member, nil
	}
	return "", nil
}
