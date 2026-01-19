package retry

import (
	"context"
	"strings"
	"time"
)

func Do(ctx context.Context, maxRetries int, fn func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = fn(); err == nil {
			return nil
		}

		if IsRateLimitError(err) {
			backoff := time.Duration(1<<i) * 15 * time.Second
			if backoff > 120*time.Second {
				backoff = 120 * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		backoff := time.Duration(1<<i) * 100 * time.Millisecond
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return err
}

func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "resource_exhausted") ||
		strings.Contains(errStr, "too many requests")
}
