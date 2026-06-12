package benchmark_test

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/metrics"
	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/queue"
	"github.com/tomtorh96/task-scheduler/internal/scheduler"
)

func TestMain(m *testing.M) {
	metrics.Init()
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newScheduler creates a started scheduler and returns a cleanup function.
func newScheduler(workers, buffer int) (*scheduler.Scheduler, func()) {
	s := scheduler.NewScheduler(workers, buffer)
	s.Start()
	return s, func() { s.Shutdown(10 * time.Second) }
}

// newPool creates a started pool with its own queue and returns a cleanup function.
func newPool(workers, buffer int) (*pool.Pool, func()) {
	q := queue.NewPriorityQueue()
	p := pool.NewPool(workers, buffer, q)
	p.Start()
	return p, func() { p.Shutdown(10 * time.Second) }
}

// submitAndWait submits n jobs to the pool and blocks until all complete.
func submitAndWait(p *pool.Pool, n int, fn func() error) {
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		p.Submit(&pool.Job{
			Fn: func() error {
				defer wg.Done()
				return fn()
			},
			Priority:   1,
			MaxRetries: 1,
		})
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Throughput benchmarks — jobs/sec at varying worker counts
// ---------------------------------------------------------------------------

// BenchmarkThroughput_1Worker measures how many no-op jobs/sec a single worker
// can process. This is the baseline — any overhead here is pure scheduler cost.
func BenchmarkThroughput_1Worker(b *testing.B) {
	p, cleanup := newPool(1, b.N+1)
	defer cleanup()

	b.ResetTimer()
	submitAndWait(p, b.N, func() error { return nil })
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "jobs/sec")
}

// BenchmarkThroughput_4Workers measures throughput with 4 concurrent workers.
func BenchmarkThroughput_4Workers(b *testing.B) {
	p, cleanup := newPool(4, b.N+1)
	defer cleanup()

	b.ResetTimer()
	submitAndWait(p, b.N, func() error { return nil })
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "jobs/sec")
}

// BenchmarkThroughput_8Workers measures throughput with 8 concurrent workers.
func BenchmarkThroughput_8Workers(b *testing.B) {
	p, cleanup := newPool(8, b.N+1)
	defer cleanup()

	b.ResetTimer()
	submitAndWait(p, b.N, func() error { return nil })
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "jobs/sec")
}

// BenchmarkThroughput_16Workers measures throughput with 16 concurrent workers.
func BenchmarkThroughput_16Workers(b *testing.B) {
	p, cleanup := newPool(16, b.N+1)
	defer cleanup()

	b.ResetTimer()
	submitAndWait(p, b.N, func() error { return nil })
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "jobs/sec")
}

// ---------------------------------------------------------------------------
// Latency benchmarks — scheduling overhead per job
// ---------------------------------------------------------------------------

// BenchmarkSchedulingLatency measures the time from Submit() to job execution
// start. This isolates queue + dispatch overhead from actual job work.
func BenchmarkSchedulingLatency(b *testing.B) {
	p, cleanup := newPool(4, b.N+1)
	defer cleanup()

	var totalLatency int64 // nanoseconds, accumulated atomically
	var wg sync.WaitGroup
	wg.Add(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enqueued := time.Now()
		p.Submit(&pool.Job{
			Fn: func() error {
				latency := time.Since(enqueued).Nanoseconds()
				atomic.AddInt64(&totalLatency, latency)
				wg.Done()
				return nil
			},
			Priority:   1,
			MaxRetries: 1,
		})
	}
	wg.Wait()

	avgLatencyNs := float64(totalLatency) / float64(b.N)
	b.ReportMetric(avgLatencyNs, "ns/dispatch")
	b.ReportMetric(avgLatencyNs/1e6, "ms/dispatch")
}

// ---------------------------------------------------------------------------
// Priority queue benchmarks — heap operations in isolation
// ---------------------------------------------------------------------------

// BenchmarkQueueEnqueue measures raw enqueue throughput on the priority queue.
func BenchmarkQueueEnqueue(b *testing.B) {
	q := queue.NewPriorityQueue()

	// pre-build jobs outside the timed section
	jobs := make([]*pool.Job, b.N)
	for i := 0; i < b.N; i++ {
		jobs[i] = &pool.Job{ID: "job", Priority: i % 10}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(jobs[i], jobs[i].Priority)
	}
}

// BenchmarkQueueDequeue measures raw dequeue throughput on the priority queue.
// Pre-fills the queue so Dequeue never blocks.
func BenchmarkQueueDequeue(b *testing.B) {
	q := queue.NewPriorityQueue()

	// pre-fill outside the timed section
	for i := 0; i < b.N; i++ {
		q.Enqueue(&pool.Job{ID: "job", Priority: i % 10}, i%10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue()
	}
}

// BenchmarkQueueRoundTrip measures enqueue + dequeue together — the full
// round trip a job takes through the priority queue.
func BenchmarkQueueRoundTrip(b *testing.B) {
	q := queue.NewPriorityQueue()
	job := &pool.Job{ID: "job", Priority: 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(job, 1)
		q.Dequeue()
	}
}

// ---------------------------------------------------------------------------
// Concurrency stress benchmark — mixed priorities under load
// ---------------------------------------------------------------------------

// BenchmarkMixedPriorityLoad submits jobs with random priorities from multiple
// goroutines simultaneously. This stresses the mutex + heap under real
// concurrent access patterns.
func BenchmarkMixedPriorityLoad(b *testing.B) {
	const totalJobs = 10000
	p, cleanup := newPool(8, totalJobs+1)
	defer cleanup()

	const producers = 4
	jobsPerProducer := totalJobs / producers

	var wg sync.WaitGroup
	wg.Add(totalJobs)

	b.ResetTimer()
	for g := 0; g < producers; g++ {
		priority := (g % 5) + 1
		go func(pri int) {
			for i := 0; i < jobsPerProducer; i++ {
				p.Submit(&pool.Job{
					Fn: func() error {
						wg.Done()
						return nil
					},
					Priority:   pri,
					MaxRetries: 1,
				})
			}
		}(priority)
	}

	wg.Wait()
	b.ReportMetric(float64(totalJobs)/b.Elapsed().Seconds(), "jobs/sec")
}

// ---------------------------------------------------------------------------
// End-to-end scheduler benchmark — full stack including HTTP-layer overhead
// ---------------------------------------------------------------------------

// BenchmarkSchedulerSubmit measures the cost of Scheduler.Submit() —
// UUID generation + queue enqueue + map write.
func BenchmarkSchedulerSubmit(b *testing.B) {
	s, cleanup := newScheduler(4, b.N+1)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Submit(func() error { return nil }, 1, 1)
	}
}
