package delivery

import (
	"math"
	"time"
)

type RetryConfig struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

// CalculateBackoff returns exponential backoff delay for the given attempt.
func CalculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	delay := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	return time.Duration(delay)
}
