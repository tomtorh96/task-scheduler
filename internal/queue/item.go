package queue

import (
	"time"
)

type Item struct {
	Job        any
	Priority   int
	EnqueuedAt time.Time
	index      int // used internally by the heap package to track position
}
