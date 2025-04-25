package benchmark

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"uptime-go/internal/monitor"
	"uptime-go/internal/net/config"
)

// memStats holds memory statistics
type memStats struct {
	HeapAlloc    uint64
	TotalAlloc   uint64
	Mallocs      uint64
	NumGC        uint32
	PauseTotalNs uint64
}

// getMemStats returns current memory statistics
func getMemStats() memStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return memStats{
		HeapAlloc:    stats.HeapAlloc,
		TotalAlloc:   stats.TotalAlloc,
		Mallocs:      stats.Mallocs,
		NumGC:        stats.NumGC,
		PauseTotalNs: stats.PauseTotalNs,
	}
}

// printMemUsage prints memory usage statistics
func printMemStats(before, after memStats) {
	fmt.Printf("Memory Usage:\n")
	fmt.Printf("  Heap Alloc: %v -> %v (%+v)\n",
		byteSize(before.HeapAlloc),
		byteSize(after.HeapAlloc),
		byteSize(after.HeapAlloc-before.HeapAlloc))
	fmt.Printf("  Total Alloc: %v -> %v (%+v)\n",
		byteSize(before.TotalAlloc),
		byteSize(after.TotalAlloc),
		byteSize(after.TotalAlloc-before.TotalAlloc))
	fmt.Printf("  Mallocs: %v -> %v (%+v)\n",
		before.Mallocs,
		after.Mallocs,
		after.Mallocs-before.Mallocs)
	fmt.Printf("  GC Runs: %v -> %v (%+v)\n",
		before.NumGC,
		after.NumGC,
		after.NumGC-before.NumGC)
	fmt.Printf("  GC Pause Total: %v -> %v (%+v)\n",
		time.Duration(before.PauseTotalNs),
		time.Duration(after.PauseTotalNs),
		time.Duration(after.PauseTotalNs-before.PauseTotalNs))
}

// byteSize formats byte size to human-readable format
func byteSize(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// createTestServer creates a test HTTP server that responds with specified status code
func createTestServer(statusCode int, responseDelay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(responseDelay)
		w.WriteHeader(statusCode)
		w.Write([]byte("OK"))
	}))
}

// createTestConfigs creates test NetworkConfig entries
func createTestConfigs(count int, server *httptest.Server) []*config.NetworkConfig {
	configs := make([]*config.NetworkConfig, count)
	for i := 0; i < count; i++ {
		configs[i] = &config.NetworkConfig{
			URL:             server.URL,
			RefreshInterval: 1 * time.Second,
			Timeout:         500 * time.Millisecond,
			FollowRedirects: true,
			SkipSSL:         false,
		}
	}
	return configs
}

// benchmarkMonitor tests the performance of monitoring a specific number of websites
func benchmarkMonitor(b *testing.B, websiteCount int) {
	// Create a test HTTP server
	server := createTestServer(200, 20*time.Millisecond)
	defer server.Close()

	// Create configs for the specified website count
	configs := createTestConfigs(websiteCount, server)

	// Record memory before
	beforeStats := getMemStats()
	startTime := time.Now()

	// Create monitor
	uptimeMonitor, err := monitor.NewUptimeMonitor(configs)
	if err != nil {
		b.Fatalf("Failed to create monitor: %v", err)
	}

	// Run the benchmark
	b.ResetTimer()

	// Start monitoring for a short period
	monitoringDuration := 5 * time.Second
	go func() {
		time.Sleep(monitoringDuration)
		uptimeMonitor.Stop()
	}()

	// Start the monitor
	uptimeMonitor.Start()

	b.StopTimer()

	// Record memory after
	afterStats := getMemStats()
	elapsedTime := time.Since(startTime)

	// Print results
	b.Logf("Benchmark for %d websites:", websiteCount)
	b.Logf("Monitoring duration: %v", monitoringDuration)
	b.Logf("Total elapsed time: %v", elapsedTime)
	b.Logf("Average time per website check: %v", elapsedTime/time.Duration(websiteCount))

	// Log memory usage
	b.Logf("Memory usage:")
	printMemStats(beforeStats, afterStats)
}

// BenchmarkMonitor1Site benchmarks monitoring 1 website
func BenchmarkMonitor1Site(b *testing.B) {
	benchmarkMonitor(b, 1)
}

// BenchmarkMonitor10Sites benchmarks monitoring 10 websites
func BenchmarkMonitor10Sites(b *testing.B) {
	benchmarkMonitor(b, 10)
}

// BenchmarkMonitor50Sites benchmarks monitoring 50 websites
func BenchmarkMonitor50Sites(b *testing.B) {
	benchmarkMonitor(b, 50)
}

// BenchmarkMonitor100Sites benchmarks monitoring 100 websites
func BenchmarkMonitor100Sites(b *testing.B) {
	benchmarkMonitor(b, 100)
}

// BenchmarkMonitor500Sites benchmarks monitoring 500 websites
func BenchmarkMonitor500Sites(b *testing.B) {
	benchmarkMonitor(b, 500)
}
