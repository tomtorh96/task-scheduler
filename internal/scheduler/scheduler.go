package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/queue"
)

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrJobNotCancellable = errors.New("job is not cancellable")
)

type Scheduler struct {
	pool  *pool.Pool
	queue *queue.PriorityQueue
	jobs  map[string]*pool.Job
	mu    sync.RWMutex
	quit  chan struct{}
}

// creates a new Pool and PriorityQueue, wires them together, returns the scheduler
func NewScheduler(workerCount int, bufferSize int) *Scheduler {
	q := queue.NewPriorityQueue()
	return &Scheduler{
		pool:  pool.NewPool(workerCount, bufferSize, q),
		queue: q,
		jobs:  make(map[string]*pool.Job),
		quit:  make(chan struct{}),
	}
}

// starts the worker pool, then launches the dispatch loop in a goroutine
func (s *Scheduler) Start() {
	s.pool.Start()
	go s.dispatch()
}

// runs in a background goroutine — loops forever:
// calls queue.Dequeue() (blocks until a job is available)
// then calls pool.Submit() to hand the job to a worker
// exits when quit is closed
func (s *Scheduler) dispatch() {
	for {
		job, ok := s.queue.Dequeue()
		if !ok {
			return // queue is closed, exit the dispatch loop
		}
		s.pool.Submit(job)
	}
}

// creates a new Job with a UUID, sets CreatedAt, Status pending
// calls queue.Enqueue() to add it to the priority queue
// returns the job ID so the caller can track it
func (s *Scheduler) Submit(fn func() error, priority int, maxRetries int) (string, error) {
	job := &pool.Job{
		ID:         uuid.New().String(),
		Fn:         fn,
		CreatedAt:  time.Now(),
		Status:     pool.StatusPending,
		Priority:   priority,
		MaxRetries: maxRetries,
	}
	s.queue.Enqueue(job, priority)
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	return job.ID, nil
}

// finds the job by ID, sets its status to cancelled
// returns ErrJobNotFound if the ID doesn't exist
// returns ErrJobNotCancellable if the job is already running/done/failed
func (s *Scheduler) Cancel(id string) error {
	job, err := s.GetJob(id)
	if err != nil {
		return err
	}
	if job.Status != pool.StatusPending {
		return ErrJobNotCancellable
	}
	job.Status = pool.StatusCancelled
	return nil
}

// looks up a job by ID, returns it or ErrJobNotFound
func (s *Scheduler) GetJob(id string) (*pool.Job, error) {
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrJobNotFound
	}
	return job, nil
}

// closes quit to stop the dispatch loop
// calls pool.Shutdown() with the timeout
func (s *Scheduler) Shutdown(timeout time.Duration) error {
	s.queue.Close() // wake up any waiting Dequeue calls
	close(s.quit)
	return s.pool.Shutdown(timeout)
}

// calls pool.Stats() and returns the stats
func (s *Scheduler) Stats() pool.Stats {
	return s.pool.Stats()
}

// returns all jobs, optionally filtered by status (empty string = all)
func (s *Scheduler) ListJobs(status pool.Status) []*pool.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*pool.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if status == "" || job.Status == status {
			result = append(result, job)
		}
	}
	return result
}
