package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/queue"
)

var (
	ErrQueueFull       = errors.New("queue is full")
	ErrShutdownTimeout = errors.New("shutdown timed out")
)

type Pool struct {
	workers       []*Worker
	jobs          chan *Job
	queue         *queue.PriorityQueue
	size          int
	mu            sync.Mutex
	jobsCompleted int64
	jobsFailed    int64
}

// creates a pool with `size` workers and a job channel with capacity `bufferSize`
func NewPool(size int, bufferSize int, q *queue.PriorityQueue) *Pool {
	var new_pool = &Pool{size: size, jobs: make(chan *Job, bufferSize), queue: q}

	for i := 0; i < size; i++ {
		new_pool.workers = append(new_pool.workers, NewWorker(i, new_pool.jobs, new_pool.queue, func() { atomic.AddInt64(&new_pool.jobsCompleted, 1) },
			func() { atomic.AddInt64(&new_pool.jobsFailed, 1) }))
	}

	return new_pool
}

// calls Start() on every worker, bringing the pool online and ready to accept jobs
func (p *Pool) Start() {
	for _, worker := range p.workers {
		worker.Start()
	}
}

// pushes a job onto the channel; returns ErrQueueFull if the channel is at capacity
func (p *Pool) Submit(job *Job) error {
	select {
	case p.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// adds workers if n > current size, stops excess workers if n < current size
func (p *Pool) Resize(n int) error {
	if n > p.size {
		for i := p.size; i < n; i++ {
			new_worker := NewWorker(i, p.jobs, p.queue, func() { atomic.AddInt64(&p.jobsCompleted, 1) },
				func() { atomic.AddInt64(&p.jobsFailed, 1) })
			p.workers = append(p.workers, new_worker)
			new_worker.Start()
		}
	} else if n < p.size {
		for i := n; i < p.size; i++ {
			p.workers[i].Stop()
		}
	}
	p.size = n
	return nil
}

// stops accepting new jobs, waits up to `timeout` for in-flight jobs to finish, then force-stops any remaining workers
func (p *Pool) Shutdown(timeout time.Duration) error {
	var wg sync.WaitGroup
	done := make(chan struct{})
	go func() {
		for _, worker := range p.workers {
			wg.Add(1)
			go func(w *Worker) {
				defer wg.Done()
				w.Stop()
			}(worker)
		}
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}

}

// returns a snapshot of current pool state (active workers, idle workers, queue depth, etc.)
func (p *Pool) Stats() Stats {
	var stats Stats
	stats.TotalWorkers = p.size
	stats.QueueDepth = len(p.jobs)
	active_workers := 0
	for _, worker := range p.workers {
		if worker.IsActive() {
			active_workers++
		}
	}
	stats.ActiveWorkers = active_workers
	stats.IdleWorkers = stats.TotalWorkers - stats.ActiveWorkers
	stats.JobsCompleted = int(atomic.LoadInt64(&p.jobsCompleted))
	stats.JobsFailed = int(atomic.LoadInt64(&p.jobsFailed))
	return stats
}
