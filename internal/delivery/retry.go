package delivery

import (
	"math"
	"math/rand"
	"time"
)

type RetryConfig struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

// CalculateBackoff returns exponential backoff with jitter.
// delay = min(BaseDelay * 2^attempt + random(0, BaseDelay), MaxDelay)
func CalculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	delay := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt))
	jitter := rand.Float64() * float64(cfg.BaseDelay)
	delay += jitter
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	return time.Duration(delay)
}
