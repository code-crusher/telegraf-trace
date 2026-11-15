package models

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/testutil"
)

// newValueMetric creates a metric whose field value is the given number so
// tests can tell metrics apart.
func valueMetric(v int64) telegraf.Metric {
	return metric.New("pressure_test", map[string]string{}, map[string]any{"value": v}, time.Unix(v, 0))
}

func newBuffer(t *testing.T, capacity int) *MemoryBuffer {
	t.Helper()
	buf, err := NewMemoryBuffer(capacity, NewBufferStats(map[string]string{}, capacity))
	require.NoError(t, err)
	buf.Stats().MetricsAdded.Set(0)
	buf.Stats().MetricsWritten.Set(0)
	buf.Stats().MetricsDropped.Set(0)
	return buf
}

func TestMemoryBufferPressureDropsOldest(t *testing.T) {
	buf := newBuffer(t, 10)

	var pressure atomic.Bool
	buf.SetPressureMonitor(pressure.Load, true)

	// Fill the buffer with 3 metrics while there is no memory pressure
	for i := range 3 {
		buf.Add(valueMetric(int64(i)))
	}
	require.Equal(t, 3, buf.Len())

	// Enable pressure and add more metrics; the oldest metrics should be
	// dropped instead of growing the buffer unbounded
	pressure.Store(true)
	for i := 3; i < 8; i++ {
		buf.Add(valueMetric(int64(i)))
	}

	require.Equal(t, 3, buf.Len())
	require.Equal(t, int64(5), buf.Stats().MetricsDropped.Get())

	// The buffer should hold the most recent metrics
	tx := buf.BeginTransaction(10)
	defer buf.EndTransaction(tx)
	var values []int64
	for _, m := range tx.Batch {
		v, ok := m.GetField("value")
		require.True(t, ok)
		values = append(values, v.(int64))
	}
	require.Equal(t, []int64{5, 6, 7}, values)
}

func TestMemoryBufferPressureDisabledDoesNotDrop(t *testing.T) {
	buf := newBuffer(t, 10)

	var pressure atomic.Bool
	pressure.Store(true)
	buf.SetPressureMonitor(pressure.Load, false)

	for i := range 5 {
		buf.Add(valueMetric(int64(i)))
	}
	require.Equal(t, 5, buf.Len())
	require.Equal(t, int64(0), buf.Stats().MetricsDropped.Get())
}

func TestMemoryBufferPressureOnEmptyBuffer(t *testing.T) {
	buf := newBuffer(t, 10)

	buf.SetPressureMonitor(func() bool { return true }, true)

	// Adding to an empty buffer under pressure must not drop anything
	buf.Add(valueMetric(1))
	require.Equal(t, 1, buf.Len())
	require.Equal(t, int64(0), buf.Stats().MetricsDropped.Get())
}

func TestMemoryBufferPressureFreezesOccupancy(t *testing.T) {
	buf := newBuffer(t, 5)

	// Fill the buffer to 4 out of 5 slots before pressure kicks in
	for i := range 4 {
		buf.Add(valueMetric(int64(i)))
	}
	require.Equal(t, 4, buf.Len())

	buf.SetPressureMonitor(func() bool { return true }, true)

	// Adding under memory pressure must not grow the buffer, instead every
	// new metric replaces the oldest buffered metric, as if the buffer had
	// reached its capacity
	for i := 4; i < 9; i++ {
		buf.Add(valueMetric(int64(i)))
	}
	require.Equal(t, 4, buf.Len())
	require.Equal(t, int64(5), buf.Stats().MetricsDropped.Get())

	// The buffer should hold the most recent metrics
	tx := buf.BeginTransaction(10)
	defer buf.EndTransaction(tx)
	var values []int64
	for _, m := range tx.Batch {
		v, ok := m.GetField("value")
		require.True(t, ok)
		values = append(values, v.(int64))
	}
	require.Equal(t, []int64{5, 6, 7, 8}, values)
}

func TestMemoryBufferPressureWithBatchInFlight(t *testing.T) {
	buf := newBuffer(t, 10)

	pressure := &atomic.Bool{}
	buf.SetPressureMonitor(pressure.Load, true)

	for i := range 4 {
		buf.Add(valueMetric(int64(i)))
	}

	// Take 2 metrics out of the buffer into a batch, then add under pressure
	tx := buf.BeginTransaction(2)
	require.Len(t, tx.Batch, 2)

	pressure.Store(true)
	buf.Add(valueMetric(10))

	// The evicted metric must be the oldest buffered one and the in-flight
	// batch must be unaffected
	require.Len(t, tx.Batch, 2)
	keep := tx.InferKeep()
	require.Len(t, keep, 2)

	buf.EndTransaction(tx)

	// Buffer should still hold the in-flight slots consistent metrics
	require.Positive(t, buf.Len())
}

func TestRunningOutputMemoryPressureTriggersFlush(t *testing.T) {
	conf := &OutputConfig{
		Filter:                  Filter{},
		MemoryPressureThreshold: 0.9,
	}

	m := &mockOutput{}
	ro, err := NewRunningOutput(m, conf, 1000, 10000)
	require.NoError(t, err)
	defer ro.Close()

	// The watcher should be created when the threshold is set
	time.Sleep(time.Millisecond) // give the watcher a chance to start
	require.NotNil(t, ro.pressureWatch)

	// Add a single metric; this is below the batch size so no flush is triggered
	ro.AddMetric(testutil.TestMetric(1, "pressure_test"))
	select {
	case <-ro.BatchReady:
		t.Fatal("unexpected batch ready signal before pressure flush")
	default:
	}

	// Simulate the memory pressure callback, as invoked by the watcher after
	// a GC cycle
	ro.onMemoryPressure()

	select {
	case <-ro.BatchReady:
	case <-time.After(time.Second):
		t.Fatal("expected a batch ready signal after memory pressure was detected")
	}
}

func TestRunningOutputMemoryPressureNoTriggerWhenFailing(t *testing.T) {
	conf := &OutputConfig{
		Filter:                  Filter{},
		MemoryPressureThreshold: 0.9,
	}

	m := &mockOutput{batchAcceptSize: -1}
	ro, err := NewRunningOutput(m, conf, 1000, 10000)
	require.NoError(t, err)
	defer ro.Close()

	ro.AddMetric(testutil.TestMetric(1, "pressure_test"))

	// Simulate a failed write so lastWriteFailed is set
	require.Error(t, ro.Write())

	// Drain any pending signal from the failed write path
	select {
	case <-ro.BatchReady:
	default:
	}

	ro.onMemoryPressure()
	select {
	case <-ro.BatchReady:
		t.Fatal("unexpected batch ready signal while the output is failing")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRunningOutputMemoryPressureDrop(t *testing.T) {
	conf := &OutputConfig{
		Filter:                  Filter{},
		MemoryPressureThreshold: 0.9,
		MemoryPressureDrop:      true,
	}

	m := &mockOutput{}
	ro, err := NewRunningOutput(m, conf, 1000, 5)
	require.NoError(t, err)
	defer ro.Close()
	ro.buffer.Stats().MetricsDropped.Set(0)

	for i := range 3 {
		ro.AddMetric(testutil.TestMetric(int64(i), "pressure_test"))
	}
	require.Equal(t, 3, ro.BufferLength())
	ro.buffer.Stats().MetricsDropped.Set(0)

	// Simulate memory pressure by replacing the monitor function with one
	// that reports pressure
	buf, ok := ro.buffer.(*MemoryBuffer)
	require.True(t, ok)
	buf.SetPressureMonitor(func() bool { return true }, true)

	ro.AddMetric(testutil.TestMetric(3, "pressure_test"))
	require.Equal(t, 3, ro.BufferLength())
	require.Equal(t, int64(1), ro.buffer.Stats().MetricsDropped.Get())
}
