package timeutil

import (
	"context"
	"math/rand/v2"
	"time"
)

// ConflictWait returns a random wait before retrying a write that conflicted, so writers don't retry in lockstep
func ConflictWait() time.Duration {
	return time.Duration(50+rand.IntN(200)) * time.Millisecond //nolint:gosec // jitter, not security
}

// Sleep waits for d, returning false if ctx is done first
func Sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
