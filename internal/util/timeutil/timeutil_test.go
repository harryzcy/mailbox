package timeutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSleep(t *testing.T) {
	assert.True(t, Sleep(context.TODO(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.TODO())
	cancel()
	assert.False(t, Sleep(ctx, time.Hour))
}

func TestConflictWait(t *testing.T) {
	for range 100 {
		wait := ConflictWait()
		assert.GreaterOrEqual(t, wait, 50*time.Millisecond)
		assert.Less(t, wait, 250*time.Millisecond)
	}
}
