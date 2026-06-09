package queue

import (
	"time"

	"github.com/tomtorh96/task-scheduler/internal/pool"
)

type Item struct {
	Job        *pool.Job
	Priority   int
	EnqueuedAt time.Time
	index      int // used internally by the heap package to track position
}
