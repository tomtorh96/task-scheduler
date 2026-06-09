package pool

import "time"

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
}
