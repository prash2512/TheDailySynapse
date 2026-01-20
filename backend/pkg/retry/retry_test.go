package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "429 status code",
			err:  errors.New("HTTP 429: rate limit exceeded"),
			want: true,
		},
		{
			name: "rate limit text",
			err:  errors.New("rate limit exceeded"),
			want: true,
		},
		{
			name: "quota exceeded",
			err:  errors.New("quota exceeded"),
			want: true,
		},
		{
			name: "resource_exhausted",
			err:  errors.New("resource_exhausted"),
			want: true,
		},
		{
			name: "too many requests",
			err:  errors.New("too many requests"),
			want: true,
		},
		{
			name: "case insensitive",
			err:  errors.New("RATE LIMIT EXCEEDED"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("something went wrong"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			if got != tt.want {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	calls := 0

	err := Do(ctx, 3, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Errorf("Do() called function %d times, want 1", calls)
	}
}

func TestDo_RetriesOnError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	testErr := errors.New("test error")

	err := Do(ctx, 3, func() error {
		calls++
		if calls < 3 {
			return testErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}
	if calls != 3 {
		t.Errorf("Do() called function %d times, want 3", calls)
	}
}

func TestDo_ExhaustsRetries(t *testing.T) {
	ctx := context.Background()
	calls := 0
	testErr := errors.New("test error")

	err := Do(ctx, 3, func() error {
		calls++
		return testErr
	})

	if err == nil {
		t.Error("Do() error = nil, want error")
	}
	if err != testErr {
		t.Errorf("Do() error = %v, want %v", err, testErr)
	}
	if calls != 3 {
		t.Errorf("Do() called function %d times, want 3", calls)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	calls := 0
	err := Do(ctx, 3, func() error {
		calls++
		return errors.New("test error")
	})

	if err != context.Canceled {
		t.Errorf("Do() error = %v, want %v", err, context.Canceled)
	}
	if calls > 3 {
		t.Errorf("Do() called function %d times, want at most 3", calls)
	}
}

func TestDo_RateLimitBackoff(t *testing.T) {
	ctx := context.Background()
	calls := 0
	rateLimitErr := errors.New("HTTP 429: rate limit exceeded")

	start := time.Now()
	err := Do(ctx, 2, func() error {
		calls++
		if calls < 2 {
			return rateLimitErr
		}
		return nil
	})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}
	// Should have waited at least 15 seconds (first retry backoff)
	if duration < 15*time.Second {
		t.Errorf("Do() duration = %v, want at least 15s", duration)
	}
}

func TestDo_RegularErrorBackoff(t *testing.T) {
	ctx := context.Background()
	calls := 0
	regularErr := errors.New("regular error")

	start := time.Now()
	err := Do(ctx, 2, func() error {
		calls++
		if calls < 2 {
			return regularErr
		}
		return nil
	})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}
	// Should have waited at least 100ms (first retry backoff)
	if duration < 100*time.Millisecond {
		t.Errorf("Do() duration = %v, want at least 100ms", duration)
	}
}

