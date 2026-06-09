package backoff

import (
	"math"
	"math/rand/v2"
	"time"
)

type Config struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64
}

// returns sensible defaults:
// InitialDelay: 1 second
// MaxDelay:     30 seconds
// Multiplier:   2.0  (doubles each retry)
// Jitter:       0.2  (±20% random variation)
func DefaultConfig() Config {
	return Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.2,
	}
}

// calculates the delay for a given attempt number
// formula: min(InitialDelay * Multiplier^attempt, MaxDelay)
// then adds jitter: delay ± (delay * Jitter * random)
// attempt 0 → ~1s, attempt 1 → ~2s, attempt 2 → ~4s, attempt 3 → ~8s
func Calculate(attempt int, cfg Config) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	result := time.Duration(delay * (1 + cfg.Jitter*(rand.Float64()*2-1)))
	if result > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return result
}
