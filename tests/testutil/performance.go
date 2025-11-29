// Performance measurement and timing helpers for performance tests.
//
// This file provides utilities for measuring execution time, simulating load,
// and validating performance targets in benchmark and timing tests.
//
// Usage:
//
//	import "github.com/eterrain/tf-backend-service/tests/testutil"
//
//	func TestPerformance(t *testing.T) {
//	    timer := testutil.NewTimer()
//	    timer.Start()
//
//	    // ... code to measure ...
//
//	    duration := timer.Stop()
//	    testutil.AssertPerformanceTarget(t, duration, 100*time.Millisecond, "Operation")
//	}
package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Timer provides simple timing measurement utilities
type Timer struct {
	startTime time.Time
	endTime   time.Time
	running   bool
}

// NewTimer creates a new timer instance
func NewTimer() *Timer {
	return &Timer{}
}

// Start begins timing measurement
func (t *Timer) Start() {
	t.startTime = time.Now()
	t.running = true
}

// Stop ends timing measurement and returns duration
func (t *Timer) Stop() time.Duration {
	if !t.running {
		return 0
	}
	t.endTime = time.Now()
	t.running = false
	return t.Duration()
}

// Duration returns the measured duration
func (t *Timer) Duration() time.Duration {
	if t.running {
		return time.Since(t.startTime)
	}
	return t.endTime.Sub(t.startTime)
}

// Reset clears the timer for reuse
func (t *Timer) Reset() {
	t.startTime = time.Time{}
	t.endTime = time.Time{}
	t.running = false
}

// AssertPerformanceTarget verifies that operation completed within target duration
func AssertPerformanceTarget(t *testing.T, actual time.Duration, target time.Duration, operation string) bool {
	t.Helper()
	return assert.LessOrEqual(t, actual, target,
		"%s should complete within %v (actual: %v)",
		operation, target, actual)
}

// MeasureOperation executes a function and returns its duration
func MeasureOperation(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// MeasureOperationN executes a function N times and returns average duration
func MeasureOperationN(fn func(), iterations int) time.Duration {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		fn()
	}
	total := time.Since(start)
	return total / time.Duration(iterations)
}

// SimulateLoad executes a function concurrently with specified goroutines and iterations
// Returns total duration and average per operation
func SimulateLoad(fn func(), goroutines int, iterationsPerGoroutine int) (total time.Duration, avgPerOp time.Duration) {
	done := make(chan bool, goroutines)
	start := time.Now()

	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < iterationsPerGoroutine; i++ {
				fn()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for g := 0; g < goroutines; g++ {
		<-done
	}

	total = time.Since(start)
	totalOps := goroutines * iterationsPerGoroutine
	avgPerOp = total / time.Duration(totalOps)

	return total, avgPerOp
}

// WarmupCache runs a function several times to warm up caches before benchmarking
func WarmupCache(fn func(), iterations int) {
	for i := 0; i < iterations; i++ {
		fn()
	}
}

// PerformanceReport contains performance measurement results
type PerformanceReport struct {
	Operations      int
	TotalDuration   time.Duration
	AvgDuration     time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	OpsPerSecond    float64
}

// MeasureDetailedPerformance runs a function multiple times and collects detailed metrics
func MeasureDetailedPerformance(fn func(), iterations int) PerformanceReport {
	durations := make([]time.Duration, iterations)

	start := time.Now()
	for i := 0; i < iterations; i++ {
		opStart := time.Now()
		fn()
		durations[i] = time.Since(opStart)
	}
	total := time.Since(start)

	// Calculate statistics
	var min, max time.Duration
	min = durations[0]
	max = durations[0]

	for _, d := range durations {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(iterations)
	opsPerSec := float64(iterations) / total.Seconds()

	return PerformanceReport{
		Operations:      iterations,
		TotalDuration:   total,
		AvgDuration:     avg,
		MinDuration:     min,
		MaxDuration:     max,
		OpsPerSecond:    opsPerSec,
	}
}
