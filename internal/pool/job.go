package pool

import (
	"sync"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusDone      Status = "done"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Job struct {
	ID         string
	Fn         func() error
	Priority   int
	MaxRetries int
	Attempt    int
	Status     Status
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	Err        error
	mu         sync.RWMutex
}

func (j *Job) GetStatus() Status {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Status
}

func (j *Job) SetStatus(s Status) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = s
}

func (j *Job) GetAttempt() int {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Attempt
}

func (j *Job) IncrementAttempt() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Attempt++
	return j.Attempt
}
