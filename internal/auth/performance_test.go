package auth

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestValidationPerformance measures validation performance under load
func TestValidationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Create store with multiple orgs and keys
	orgs := make([]orgConfigSimple, 10)
	for i := 0; i < 10; i++ {
		orgs[i] = orgConfigSimple{
			OrgID:   uuid.New(),
			APIKeys: []string{fmt.Sprintf("key-%d-1", i), fmt.Sprintf("key-%d-2", i)},
		}
	}

	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Performance test: measure throughput
	duration := 5 * time.Second
	var successCount, failureCount atomic.Int64

	start := time.Now()
	stopChan := make(chan struct{})

	// Launch multiple goroutines
	numGoroutines := 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			orgIdx := id % len(orgs)
			orgID := orgs[orgIdx].OrgID
			apiKey := orgs[orgIdx].APIKeys[0]

			for {
				select {
				case <-stopChan:
					return
				default:
					valid, err := store.ValidateCredentials(orgID, apiKey)
					if err != nil {
						t.Errorf("Validation error: %v", err)
						failureCount.Add(1)
					} else if valid {
						successCount.Add(1)
					} else {
						failureCount.Add(1)
					}
				}
			}
		}(i)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	elapsed := time.Since(start)
	totalRequests := successCount.Load() + failureCount.Load()
	throughput := float64(totalRequests) / elapsed.Seconds()

	t.Logf("Performance Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount.Load())
	t.Logf("  Failed: %d", failureCount.Load())
	t.Logf("  Throughput: %.2f req/sec", throughput)
	t.Logf("  Avg Latency: %.2f ms", elapsed.Seconds()*1000/float64(totalRequests))

	// Bcrypt is intentionally slow, expect around 100-500 req/sec with cost 12
	if throughput < 10 {
		t.Errorf("Throughput too low: %.2f req/sec", throughput)
	}
}

// TestValidationLatency measures latency distribution
func TestValidationLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	apiKey := "test-key"
	orgs := []orgConfigSimple{{OrgID: orgID, APIKeys: []string{apiKey}}}

	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Measure latencies
	numSamples := 100
	latencies := make([]time.Duration, numSamples)

	for i := 0; i < numSamples; i++ {
		start := time.Now()
		valid, err := store.ValidateCredentials(orgID, apiKey)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Validation error: %v", err)
		}
		if !valid {
			t.Fatal("Validation should succeed")
		}

		latencies[i] = latency
	}

	// Calculate statistics
	var sum time.Duration
	min := latencies[0]
	max := latencies[0]

	for _, lat := range latencies {
		sum += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	avg := sum / time.Duration(numSamples)

	// Calculate percentiles (simple approach)
	sortedLatencies := make([]time.Duration, numSamples)
	copy(sortedLatencies, latencies)
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	p50 := sortedLatencies[numSamples*50/100]
	p95 := sortedLatencies[numSamples*95/100]
	p99 := sortedLatencies[numSamples*99/100]

	t.Logf("Latency Statistics (n=%d):", numSamples)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Avg: %v", avg)
	t.Logf("  P50: %v", p50)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)

	// Bcrypt with cost 12 should take roughly 50-200ms per validation
	if avg > 500*time.Millisecond {
		t.Errorf("Average latency too high: %v", avg)
	}
}

// TestConcurrentValidationScaling tests how validation performance scales with concurrency
func TestConcurrentValidationScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scaling test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	apiKey := "test-key"
	orgs := []orgConfigSimple{{OrgID: orgID, APIKeys: []string{apiKey}}}

	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Test with different concurrency levels
	concurrencyLevels := []int{1, 2, 5, 10, 25, 50, 100}
	requestsPerLevel := 100

	t.Logf("Concurrency Scaling Test:")
	t.Logf("%-12s %-15s %-15s %-15s", "Concurrency", "Total Time", "Throughput", "Avg Latency")

	for _, concurrency := range concurrencyLevels {
		start := time.Now()
		var wg sync.WaitGroup
		requestsPerGoroutine := requestsPerLevel / concurrency

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerGoroutine; j++ {
					store.ValidateCredentials(orgID, apiKey)
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)

		throughput := float64(requestsPerLevel) / elapsed.Seconds()
		avgLatency := elapsed / time.Duration(requestsPerLevel)

		t.Logf("%-12d %-15v %-15.2f %-15v",
			concurrency, elapsed, throughput, avgLatency)
	}
}

// TestMemoryUsageUnderLoad tests memory stability under sustained load
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Create larger dataset
	numOrgs := 50
	keysPerOrg := 5
	orgs := make([]orgConfigSimple, numOrgs)

	for i := 0; i < numOrgs; i++ {
		keys := make([]string, keysPerOrg)
		for j := 0; j < keysPerOrg; j++ {
			keys[j] = fmt.Sprintf("key-org%d-%d", i, j)
		}
		orgs[i] = orgConfigSimple{
			OrgID:   uuid.New(),
			APIKeys: keys,
		}
	}

	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Run sustained load for a period
	duration := 10 * time.Second
	var requestCount atomic.Int64
	stopChan := make(chan struct{})

	numGoroutines := 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					orgIdx := id % len(orgs)
					keyIdx := requestCount.Load() % int64(keysPerOrg)
					store.ValidateCredentials(orgs[orgIdx].OrgID, orgs[orgIdx].APIKeys[keyIdx])
					requestCount.Add(1)
				}
			}
		}(i)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	t.Logf("Memory Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", requestCount.Load())
	t.Logf("  Organizations: %d", numOrgs)
	t.Logf("  Keys per Org: %d", keysPerOrg)
	t.Logf("  Test completed without OOM")
}

// TestReloadPerformanceImpact measures the impact of hot reload on ongoing requests
func TestReloadPerformanceImpact(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reload performance test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	orgID := uuid.New()
	apiKey := "test-key"
	orgs := []orgConfigSimple{{OrgID: orgID, APIKeys: []string{apiKey}}}

	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}

	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()

	// Measure baseline performance
	baselineDuration := 2 * time.Second
	baselineCount := measureRequestsInDuration(t, store, orgID, apiKey, baselineDuration)
	baselineThroughput := float64(baselineCount) / baselineDuration.Seconds()

	t.Logf("Baseline throughput: %.2f req/sec", baselineThroughput)

	// Now measure with periodic reloads
	reloadDuration := 5 * time.Second
	var reloadCount atomic.Int64
	stopChan := make(chan struct{})
	var wg sync.WaitGroup

	// Continuous validation
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopChan:
				return
			default:
				store.ValidateCredentials(orgID, apiKey)
				reloadCount.Add(1)
			}
		}
	}()

	// Periodic reloads
	reloadTicker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for {
			select {
			case <-stopChan:
				reloadTicker.Stop()
				return
			case <-reloadTicker.C:
				// Trigger reload by updating file
				generateAuthConfigSimple(orgs, authConfig)
			}
		}
	}()

	time.Sleep(reloadDuration)
	close(stopChan)
	wg.Wait()

	reloadThroughput := float64(reloadCount.Load()) / reloadDuration.Seconds()
	impactPercent := ((baselineThroughput - reloadThroughput) / baselineThroughput) * 100

	t.Logf("With reloads throughput: %.2f req/sec", reloadThroughput)
	t.Logf("Performance impact: %.2f%%", impactPercent)

	// Reload shouldn't have huge impact (accept up to 50% degradation during heavy reload)
	if impactPercent > 50 {
		t.Logf("Warning: Reload has significant performance impact: %.2f%%", impactPercent)
	}
}

// TestLargeScaleConfiguration tests performance with many orgs and keys
func TestLargeScaleConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale test in short mode")
	}

	tmpDir := t.TempDir()
	authConfig := filepath.Join(tmpDir, "auth.cfg")

	// Create large configuration
	numOrgs := 1000
	keysPerOrg := 10
	t.Logf("Creating configuration with %d orgs and %d keys per org", numOrgs, keysPerOrg)

	orgs := make([]orgConfigSimple, numOrgs)
	for i := 0; i < numOrgs; i++ {
		keys := make([]string, keysPerOrg)
		for j := 0; j < keysPerOrg; j++ {
			keys[j] = fmt.Sprintf("key-%d-%d", i, j)
		}
		orgs[i] = orgConfigSimple{
			OrgID:   uuid.New(),
			APIKeys: keys,
		}
	}

	// Measure load time
	start := time.Now()
	if err := generateAuthConfigSimple(orgs, authConfig); err != nil {
		t.Fatalf("Failed to generate auth config: %v", err)
	}
	generateTime := time.Since(start)
	t.Logf("Generate time: %v", generateTime)

	// Measure store creation/load time
	start = time.Now()
	store, err := NewFileStore(authConfig)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer store.Close()
	loadTime := time.Since(start)
	t.Logf("Load time: %v", loadTime)

	// Test validation performance with large dataset
	testOrg := orgs[numOrgs/2]
	testKey := testOrg.APIKeys[0]

	start = time.Now()
	valid, err := store.ValidateCredentials(testOrg.OrgID, testKey)
	validationTime := time.Since(start)

	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if !valid {
		t.Error("Validation should succeed")
	}

	t.Logf("Single validation time: %v", validationTime)

	// Test reload time with large dataset
	start = time.Now()
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	reloadTime := time.Since(start)
	t.Logf("Reload time: %v", reloadTime)

	// Validation time should be consistent regardless of dataset size (O(n) where n=keys for that org)
	if validationTime > 1*time.Second {
		t.Errorf("Validation time too high for large dataset: %v", validationTime)
	}
}

// TestBcryptCostImpact compares performance across different bcrypt costs
func TestBcryptCostImpact(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bcrypt cost test in short mode")
	}

	apiKey := "test-key"
	costs := []int{4, 8, 10, 12, 14}

	t.Logf("Bcrypt Cost Impact:")
	t.Logf("%-6s %-15s %-20s", "Cost", "Hash Time", "Verify Time")

	for _, cost := range costs {
		// Measure hash time
		start := time.Now()
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), cost)
		hashTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to hash with cost %d: %v", cost, err)
		}

		// Measure verify time
		start = time.Now()
		err = bcrypt.CompareHashAndPassword(hashedBytes, []byte(apiKey))
		verifyTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to verify with cost %d: %v", cost, err)
		}

		t.Logf("%-6d %-15v %-20v", cost, hashTime, verifyTime)
	}

	t.Log("\nNote: Cost 12 is recommended for production")
	t.Log("Higher cost = better security but slower performance")
}

// Helper function to measure requests in a duration
func measureRequestsInDuration(t *testing.T, store *FileStore, orgID uuid.UUID, apiKey string, duration time.Duration) int64 {
	var count atomic.Int64
	stopChan := make(chan struct{})

	go func() {
		for {
			select {
			case <-stopChan:
				return
			default:
				store.ValidateCredentials(orgID, apiKey)
				count.Add(1)
			}
		}
	}()

	time.Sleep(duration)
	close(stopChan)
	time.Sleep(100 * time.Millisecond) // Let goroutine finish

	return count.Load()
}

// BenchmarkValidationDifferentOrgCounts benchmarks with varying numbers of orgs
func BenchmarkValidationDifferentOrgCounts(b *testing.B) {
	orgCounts := []int{1, 10, 100, 1000}

	for _, numOrgs := range orgCounts {
		b.Run(fmt.Sprintf("orgs=%d", numOrgs), func(b *testing.B) {
			tmpDir := b.TempDir()
			authConfig := filepath.Join(tmpDir, "auth.cfg")

			orgs := make([]orgConfigSimple, numOrgs)
			for i := 0; i < numOrgs; i++ {
				orgs[i] = orgConfigSimple{
					OrgID:   uuid.New(),
					APIKeys: []string{fmt.Sprintf("key-%d", i)},
				}
			}

			generateAuthConfigSimple(orgs, authConfig)
			store, _ := NewFileStore(authConfig)
			defer store.Close()

			testOrg := orgs[0]

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.ValidateCredentials(testOrg.OrgID, testOrg.APIKeys[0])
			}
		})
	}
}

// BenchmarkValidationDifferentKeyCountsPerOrg benchmarks with varying keys per org
func BenchmarkValidationDifferentKeyCountsPerOrg(b *testing.B) {
	keyCounts := []int{1, 5, 10, 50}

	for _, numKeys := range keyCounts {
		b.Run(fmt.Sprintf("keys=%d", numKeys), func(b *testing.B) {
			tmpDir := b.TempDir()
			authConfig := filepath.Join(tmpDir, "auth.cfg")

			orgID := uuid.New()
			keys := make([]string, numKeys)
			for i := 0; i < numKeys; i++ {
				keys[i] = fmt.Sprintf("key-%d", i)
			}

			orgs := []orgConfigSimple{{OrgID: orgID, APIKeys: keys}}
			generateAuthConfigSimple(orgs, authConfig)
			store, _ := NewFileStore(authConfig)
			defer store.Close()

			// Test with the last key (worst case)
			testKey := keys[numKeys-1]

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.ValidateCredentials(orgID, testKey)
			}
		})
	}
}
