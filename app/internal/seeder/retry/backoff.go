package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

func Duration(attempt int, initial time.Duration, maxBackoff time.Duration, jitter float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	backoff := float64(initial) * math.Pow(2, float64(attempt-1))
	if backoff > float64(maxBackoff) {
		backoff = float64(maxBackoff)
	}

	if jitter > 0 {
		multiplier := 1 + ((rand.Float64() * 2 * jitter) - jitter)
		backoff *= multiplier
	}

	if backoff < 0 {
		backoff = 0
	}

	return time.Duration(backoff)
}

func Wait(ctx context.Context, attempt int, initial time.Duration, maxBackoff time.Duration, jitter float64) error {
	timer := time.NewTimer(Duration(attempt, initial, maxBackoff, jitter))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
