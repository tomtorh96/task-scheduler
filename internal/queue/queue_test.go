package queue_test

import (
	"testing"
	"time"

	"github.com/tomtorh96/task-scheduler/internal/pool"
	"github.com/tomtorh96/task-scheduler/internal/queue"
)

func TestHigherPriorityDequeuesFirst(t *testing.T) {
	// arrange
	pq := queue.NewPriorityQueue()

	lowPriorityJob := &pool.Job{ID: "low", Priority: 1}
	highPriorityJob := &pool.Job{ID: "high", Priority: 5}

	// enqueue low priority first, then high
	pq.Enqueue(lowPriorityJob, 1)
	pq.Enqueue(highPriorityJob, 5)

	// act
	first, _ := pq.Dequeue()
	second, _ := pq.Dequeue()

	// assert
	if first.(*pool.Job).ID != "high" {
		t.Errorf("expected high priority job first, got %s", first.(*pool.Job).ID)
	}
	if second.(*pool.Job).ID != "low" {
		t.Errorf("expected low priority job second, got %s", second.(*pool.Job).ID)
	}
}

func TestSamePriorityEarlierJobFirst(t *testing.T) {
	// arrange
	pq := queue.NewPriorityQueue()

	firstJob := &pool.Job{ID: "first", Priority: 3}
	secondJob := &pool.Job{ID: "second", Priority: 3}

	// enqueue first job, then second
	pq.Enqueue(firstJob, 3)
	time.Sleep(10 * time.Millisecond)
	pq.Enqueue(secondJob, 3)

	// act
	first, _ := pq.Dequeue()
	second, _ := pq.Dequeue()

	// assert
	if first.(*pool.Job).ID != "first" {
		t.Errorf("expected first job first, got %s", first.(*pool.Job).ID)
	}
	if second.(*pool.Job).ID != "second" {
		t.Errorf("expected second job second, got %s", second.(*pool.Job).ID)
	}
}

func TestDequeueAfterCloseReturnsFalse(t *testing.T) {
	pq := queue.NewPriorityQueue()
	pq.Close()
	_, ok := pq.Dequeue()
	if ok {
		t.Error("expected false, got true")
	}
}

func TestDequeueBlocksUntilJobArrives(t *testing.T) {
	pq := queue.NewPriorityQueue()
	job := &pool.Job{ID: "delayed", Priority: 1}

	go func() {
		time.Sleep(10 * time.Millisecond)
		pq.Enqueue(job, 1)
	}()

	result, ok := pq.Dequeue()
	if !ok {
		t.Error("expected ok to be true")
	}
	if result.(*pool.Job).ID != "delayed" {
		t.Errorf("expected job ID 'delayed', got %s", result.(*pool.Job).ID)
	}
}
