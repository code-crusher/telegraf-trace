package memlimit

import (
	"math"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setMemoryLimit(t *testing.T, limit int64) {
	t.Helper()
	old := debug.SetMemoryLimit(limit)
	t.Cleanup(func() { debug.SetMemoryLimit(old) })
}

func TestUsageIsPositive(t *testing.T) {
	require.Positive(t, Usage())
}

func TestLimitReflectsSetMemoryLimit(t *testing.T) {
	setMemoryLimit(t, 1<<20)
	require.Equal(t, int64(1<<20), Limit())
}

func TestPressedDisabledWithoutLimit(t *testing.T) {
	setMemoryLimit(t, math.MaxInt64)
	require.False(t, Pressed(0.5))
}

func TestPressedDisabledWithZeroFraction(t *testing.T) {
	setMemoryLimit(t, 1)
	require.False(t, Pressed(0))
	require.False(t, Pressed(-1))
	require.False(t, Pressed(math.NaN()))
}

func TestPressedWithTinyLimit(t *testing.T) {
	// Any usage is above a 1 byte limit
	setMemoryLimit(t, 1)
	require.True(t, Pressed(0.5))
}

func TestPressedWithHugeLimit(t *testing.T) {
	setMemoryLimit(t, math.MaxInt64-1)
	require.False(t, Pressed(0.5))
}

func TestWatcherReportsPressure(t *testing.T) {
	setMemoryLimit(t, 1)
	pressured := make(chan struct{}, 1)
	w := NewWatcher(0.5, func() {
		select {
		case pressured <- struct{}{}:
		default:
		}
	})
	defer func() { require.NoError(t, w.Close()) }()

	// The GC finalizer based notification is asynchronous, so wait for the
	// watcher to observe at least one GC cycle.
	require.Eventually(t, func() bool {
		runtime.GC()
		select {
		case <-pressured:
			return w.Pressure()
		default:
			return false
		}
	}, 10*time.Second, 10*time.Millisecond)
}

func TestWatcherNoPressureWhenLimitDisabled(t *testing.T) {
	setMemoryLimit(t, math.MaxInt64)
	called := make(chan struct{}, 1)
	w := NewWatcher(0.5, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	defer func() { require.NoError(t, w.Close()) }()

	runtime.GC()
	select {
	case <-called:
		t.Fatal("onPressure should not be called when no memory limit is set")
	case <-time.After(100 * time.Millisecond):
	}
	require.False(t, w.Pressure())
}
