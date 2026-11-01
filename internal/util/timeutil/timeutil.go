package timeutil

import (
	"context"
	"time"
)

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
