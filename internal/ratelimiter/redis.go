package ratelimiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding window rate limiter using Redis.
// Limits to maxPerSecond requests per channel per second.
type RateLimiter struct {
	client       *redis.Client
	maxPerSecond int
}

func New(client *redis.Client, maxPerSecond int) *RateLimiter {
	return &RateLimiter{client: client, maxPerSecond: maxPerSecond}
}

var rateLimitScript = redis.NewScript(`
	local key = KEYS[1]
	local limit = tonumber(ARGV[1])
	local now = tonumber(ARGV[2])
	local window = 1000

	redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
	local count = redis.call('ZCARD', key)

	if count < limit then
		redis.call('ZADD', key, now, now .. '-' .. math.random(1000000))
		redis.call('PEXPIRE', key, window)
		return 1
	end

	return 0
`)

func (r *RateLimiter) Allow(ctx context.Context, channel string) (bool, error) {
	key := "ratelimit:" + channel
	now := time.Now().UnixMilli()

	result, err := rateLimitScript.Run(ctx, r.client, []string{key}, r.maxPerSecond, now).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
