package pool

import (
	"sync"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/queue"
	"github.com/tomtorh96/task-scheduler/pkg/backoff"
)

type Worker struct {
	ID     int
	jobs   chan *Job
	quit   chan struct{}
	queue  *queue.PriorityQueue
	active bool
	mutex  sync.Mutex
}

// creates a new worker with the given ID, wires it to the shared job channel'
func NewWorker(id int, jobs chan *Job, q *queue.PriorityQueue) *Worker {
	var new_worker = &Worker{ID: id, jobs: jobs, quit: make(chan struct{}), queue: q}

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
				job.Status = StatusRunning
				job.StartedAt = time.Now()
				w.mutex.Unlock()

				err := job.Fn()
				w.mutex.Lock()
				if err != nil {
					job.Attempt++
					if job.Attempt < job.MaxRetries {
						delay := backoff.Calculate(job.Attempt, backoff.DefaultConfig())
						go func() {
							time.Sleep(delay)
							w.queue.Enqueue(job, job.Priority)
						}()
					} else {
						job.Status = StatusFailed
						job.Err = err
					}
				} else {
					job.Status = StatusDone
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
