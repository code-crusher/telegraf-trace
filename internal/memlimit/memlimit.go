// Package memlimit provides helpers to detect memory pressure of the Go
// runtime relative to the soft memory limit configured via GOMEMLIMIT (or
// runtime/debug.SetMemoryLimit).
//
// The Go runtime does not provide a notification mechanism for memory
// pressure, but the limit it tries to respect is well defined: it is the
// value of runtime.MemStats.Sys - runtime.MemStats.HeapReleased. This package
// exposes helpers to compute the current usage and to report whether it is
// approaching the configured limit. A Watcher can additionally check the
// usage after every garbage collection cycle using the gcnotifier package.
package memlimit

import (
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/CAFxX/gcnotifier"
)

// Usage returns the current memory usage of the Go runtime in bytes as
// defined by the GOMEMLIMIT documentation, i.e.
// runtime.MemStats.Sys - runtime.MemStats.HeapReleased.
func Usage() int64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.Sys) - int64(ms.HeapReleased)
}

// Limit returns the currently configured soft memory limit in bytes. If no
// limit is configured the value math.MaxInt64 is returned, matching the
// behavior of debug.SetMemoryLimit(-1).
func Limit() int64 {
	return debug.SetMemoryLimit(-1)
}

// Pressed reports whether the current memory usage is at or above
// fraction*Limit(). A fraction of 0 or less disables the check and always
// reports false. If no memory limit is configured (i.e. the limit is
// math.MaxInt64) it always reports false, as there is no limit to be close
// to.
func Pressed(fraction float64) bool {
	if fraction <= 0 || math.IsNaN(fraction) {
		return false
	}
	limit := Limit()
	if limit >= math.MaxInt64 {
		return false
	}
	return float64(Usage()) >= fraction*float64(limit)
}

// Watcher watches the memory usage of the process and reports whether it is
// under memory pressure. The check is performed after every garbage
// collection cycle using the gcnotifier package, as suggested in
// https://github.com/golang/go/issues/29696.
//
// When the process is not under memory pressure the zero value is reported.
// The onPressure callback, if any, is called on every GC cycle that observes
// the process under memory pressure.
type Watcher struct {
	threshold  float64
	onPressure func()

	notifier *gcnotifier.GCNotifier
	pressed  atomic.Bool
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewWatcher starts a new Watcher that calls onPressure (if not nil) after
// every GC cycle that observes the process to be under memory pressure. The
// threshold is the fraction of the GOMEMLIMIT at which the process is
// considered to be under memory pressure, i.e. a value of 0.9 means the
// process is under pressure when usage reaches 90% of the limit. A threshold
// of 0 or less disables the detection and onPressure is never called.
func NewWatcher(threshold float64, onPressure func()) *Watcher {
	w := &Watcher{
		threshold:  threshold,
		onPressure: onPressure,
		notifier:   gcnotifier.New(),
		done:       make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// Pressure reports whether the last GC cycle observed the process to be
// under memory pressure.
func (w *Watcher) Pressure() bool {
	return w.pressed.Load()
}

// Close stops the watcher and releases its resources.
func (w *Watcher) Close() error {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
	w.wg.Wait()
	w.notifier.Close()
	return nil
}

func (w *Watcher) run() {
	defer w.wg.Done()
	for {
		select {
		case <-w.done:
			return
		case _, ok := <-w.notifier.AfterGC():
			if !ok {
				return
			}
			pressed := Pressed(w.threshold)
			w.pressed.Store(pressed)
			if pressed && w.onPressure != nil {
				w.onPressure()
			}
		}
	}
}
