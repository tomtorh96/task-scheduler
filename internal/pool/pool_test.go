package pool_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/metrics"
	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/queue"
)

func TestMain(m *testing.M) {
	metrics.Init()
	os.Exit(m.Run())
}

func TestAllJobsAreDone(t *testing.T) {
	q := queue.NewPriorityQueue()
	p := pool.NewPool(5, 10, q)
	n := 5
	p.Start()
	for i := 0; i < n; i++ {
		job := &pool.Job{ID: fmt.Sprint(i), Priority: 1, Fn: func() error { return nil }}
		p.Submit(job)
	}
	p.Shutdown(5 * time.Second)
	if p.Stats().JobsCompleted != n {
		t.Errorf("expected %d completed jobs, got %d", n, p.Stats().JobsCompleted)
	}

}

func TestWorkerCount(t *testing.T) {
	q := queue.NewPriorityQueue()
	p := pool.NewPool(3, 10, q)
	stats := p.Stats()
	if stats.TotalWorkers != 3 {
		t.Errorf("expected 3 workers, got %d", stats.TotalWorkers)
	}
}

func TestConcurrentJobs(t *testing.T) {
	n := 10
	q := queue.NewPriorityQueue()
	p := pool.NewPool(2, time.Now().Nanosecond(), q)
	p.Start()
	done := make(chan struct{}, n)

	for i := 0; i < n; i++ {
		p.Submit(&pool.Job{
			Fn: func() error {
				done <- struct{}{}
				return nil
			},
		})
	}

	// wait for all 10
	for i := 0; i < n; i++ {
		<-done
	}
	stats := p.Stats()
	if stats.JobsCompleted != n {
		t.Errorf("expected %d completed jobs, got %d", n, stats.JobsCompleted)
	}
}

func TestShutdownWithRunningJob(t *testing.T) {
	q := queue.NewPriorityQueue()
	p := pool.NewPool(1, 10, q)
	p.Start()
	p.Submit(&pool.Job{
		Fn: func() error {
			time.Sleep(2 * time.Second)
			return nil
		},
	})
	err := p.Shutdown(5 * time.Second)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
