package integration_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/metrics"
	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/scheduler"
)

func TestMain(m *testing.M) {
	metrics.Init()
	os.Exit(m.Run())
}

func waitForStatus(t *testing.T, s *scheduler.Scheduler, id string, want pool.Status, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, _ := s.GetJob(id)
		if job.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("job never reached status %s", want)
}

func TestCreateJobForScheduler(t *testing.T) {
	sched := scheduler.NewScheduler(1, 10)
	sched.Start()
	id, _ := sched.Submit(func() error { return nil }, 1, 3)
	waitForStatus(t, sched, id, pool.StatusDone, 2*time.Second)
	sched.Shutdown(2 * time.Second)
}

func TestPriorityCheck(t *testing.T) {
	sched := scheduler.NewScheduler(1, 10)
	var fastFinished, slowFinished time.Time
	sched.Submit(func() error {
		time.Sleep(100 * time.Millisecond)
		slowFinished = time.Now()
		return nil
	}, 1, 1)

	sched.Submit(func() error {
		fastFinished = time.Now()
		return nil
	}, 5, 1)
	
	sched.Start()

	time.Sleep(500 * time.Millisecond) // wait for both jobs to finish
	if fastFinished.IsZero() || slowFinished.IsZero() {
		t.Error("jobs did not complete")
	}
	if !fastFinished.Before(slowFinished) {
		t.Errorf("expected fast job to finish before slow job")
	}
}

func TestRetryFunc(t *testing.T) {
	sched := scheduler.NewScheduler(1, 10)
	attempts := 0
	sched.Start()
	id, _ := sched.Submit(func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("failing on purpose")
		}
		return nil
	}, 1, 3)

	waitForStatus(t, sched, id, pool.StatusDone, 10*time.Second)

	job, _ := sched.GetJob(id)
	if job.Attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", job.Attempt)
	}
}

func TestCancelingJob(t *testing.T) {
	sched := scheduler.NewScheduler(1, 10)
	id, _ := sched.Submit(func() error { return nil }, 1, 3)
	sched.Cancel(id)
	job, _ := sched.GetJob(id)
	if job.Status != pool.StatusCancelled {
		t.Errorf("expected job cancelled, got %v", job.Status)
	}
}
