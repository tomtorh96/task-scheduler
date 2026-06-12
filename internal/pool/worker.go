package pool

import (
	"sync"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/metrics"
	"github.com/tomtorh96/task-scheduler/internal/queue"
	"github.com/tomtorh96/task-scheduler/pkg/backoff"
)

type Worker struct {
	ID         int
	jobs       chan *Job
	quit       chan struct{}
	queue      *queue.PriorityQueue
	active     bool
	mutex      sync.Mutex
	onComplete func() // called when job succeeds
	onFailed   func() // called when job permanently fails
}

// creates a new worker with the given ID, wires it to the shared job channel'
func NewWorker(id int, jobs chan *Job, q *queue.PriorityQueue, onComplete func(), onFailed func()) *Worker {
	var new_worker = &Worker{ID: id, jobs: jobs, quit: make(chan struct{}), queue: q, onComplete: onComplete, onFailed: onFailed}

	return new_worker
}

// launches a goroutine that loops: pull job from channel → set status running → call job.Fn() → set status done/failed
func (w *Worker) Start() {
	go func() {
		for {
			select {
			case <-w.quit:
				return
			case job, ok := <-w.jobs:
				if !ok {
					return
				}
				w.mutex.Lock()
				w.active = true
				job.SetStatus(StatusRunning)
				job.StartedAt = time.Now()
				metrics.JobQueueWait.Observe(time.Since(job.CreatedAt).Seconds())
				metrics.ActiveWorkers.Inc()
				w.mutex.Unlock()

				err := job.Fn()
				w.mutex.Lock()
				metrics.JobDuration.Observe(time.Since(job.StartedAt).Seconds())
				metrics.ActiveWorkers.Dec()
				metrics.QueueDepth.Dec()
				attempt := job.IncrementAttempt()
				if err != nil {
					if attempt < job.MaxRetries {
						delay := backoff.Calculate(attempt, backoff.DefaultConfig())
						go func() {
							time.Sleep(delay)
							w.queue.Enqueue(job, job.Priority)
						}()
					} else {
						job.SetStatus(StatusFailed)
						job.Err = err
						job.FinishedAt = time.Now()
						metrics.JobsFailed.Inc()
						w.onFailed()
					}
				} else {
					job.SetStatus(StatusDone)
					job.FinishedAt = time.Now()
					metrics.JobsCompleted.Inc()
					w.onComplete()
				}
				w.active = false
				w.mutex.Unlock()
			}
		}
	}()
}

// sends a signal on quit channel, causing the goroutine in Start() to exit cleanly after finishing its current job
func (w *Worker) Stop() {
	close(w.quit)
}

// returns true if the worker is currently executing a job, false if idle
func (w *Worker) IsActive() bool {
	w.mutex.Lock()
	active := w.active
	w.mutex.Unlock()
	return active
}
