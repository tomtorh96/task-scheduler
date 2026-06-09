package queue

import (
	"container/heap"
	"sync"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/pool"
)

type PriorityQueue struct {
	items    []*Item
	mu       sync.Mutex
	cond     *sync.Cond
	isClosed bool
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{}
	pq.cond = sync.NewCond(&pq.mu)
	return pq
}

// creates a new empty priority queue and initialises the cond variable linked to the mutex

// --- these 5 methods are required by container/heap ---

// returns the number of items currently in the queue
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// returns true if item at i should be processed before item at j
// higher priority wins; if equal, earlier EnqueuedAt wins
func (pq *PriorityQueue) Less(i, j int) bool {
	if pq.items[i].Priority == pq.items[j].Priority {
		return pq.items[i].EnqueuedAt.Before(pq.items[j].EnqueuedAt)
	}
	return pq.items[i].Priority > pq.items[j].Priority
}

// swaps items at index i and j, and updates their index fields
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

// appends a new item to the items slice, sets its index field
func (pq *PriorityQueue) Push(x any) {
	item := x.(*Item)
	item.index = len(pq.items)
	pq.items = append(pq.items, item)
}

// removes and returns the last item from the items slice
func (pq *PriorityQueue) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	pq.items = old[0 : n-1]
	return item
}

// --- these are the methods you call from outside ---

// wraps job in an Item, sets Priority and EnqueuedAt, calls heap.Push
// locks the mutex, signals cond to wake a waiting worker
func (pq *PriorityQueue) Enqueue(job *pool.Job, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	item := &Item{
		Job:        job,
		Priority:   priority,
		EnqueuedAt: time.Now(),
	}
	heap.Push(pq, item)
	pq.cond.Signal()
}

func (pq *PriorityQueue) Close() {
	pq.mu.Lock()
	pq.isClosed = true
	pq.cond.Broadcast()
	pq.mu.Unlock()
}

// blocks if queue is empty (waits on cond)
// once an item is available, calls heap.Pop and returns the job
// locks the mutex while accessing the heap
func (pq *PriorityQueue) Dequeue() (*pool.Job, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	for pq.Len() == 0 {
		if pq.isClosed {
			return nil, false
		}
		pq.cond.Wait()
	}
	if pq.isClosed {
		return nil, false
	}
	return pq.Pop().(*Item).Job, true
}
