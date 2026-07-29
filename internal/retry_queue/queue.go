package retry_queue

import (
	"container/heap"
	"context"
	"okop/internal/work"
	"sync"
	"time"
)

const (
	baseBackoff = 5 * time.Second
	maxBackoff  = 5 * time.Minute
)

// backoff returns the delay before the next attempt.
// At maxBackoff the caller resets the attempt counter to 0 so the cycle restarts.
func backoff(attempt int) time.Duration {
	if attempt > 10 {
		attempt = 10
	}
	d := baseBackoff * (1 << attempt)
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

type queueItem struct {
	item    work.Item
	retryAt time.Time
}

// minHeap is a min-heap of queueItems ordered by retryAt (soonest first).
type minHeap []queueItem

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].retryAt.Before(h[j].retryAt) }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(queueItem)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Queue holds apps that failed and are waiting to be retried after a backoff delay.
type Queue struct {
	mu   sync.Mutex
	heap minHeap
}

func New() *Queue {
	q := &Queue{}
	heap.Init(&q.heap)
	return q
}

// Add enqueues an app for retry. attempt is the current failure count and is used
// to compute the backoff delay. When the delay reaches maxBackoff the attempt counter
// resets to 0 so the cycle restarts from the shortest delay.
func (q *Queue) Add(item work.Item) {
	delay := backoff(item.Attempt)

	item.Attempt = item.Attempt + 1
	if delay >= maxBackoff {
		item.Attempt = 0
	}

	q.mu.Lock()
	heap.Push(&q.heap, queueItem{
		item:    item,
		retryAt: time.Now().Add(delay),
	})
	q.mu.Unlock()
}

// Len returns the number of items currently waiting in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.heap.Len()
}

// Run dispatches items whose retry time has passed into out.
// It runs until ctx is cancelled.
func (q *Queue) Run(ctx context.Context, out chan<- work.Item) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.mu.Lock()
			now := time.Now()
			for q.heap.Len() > 0 && !q.heap[0].retryAt.After(now) {
				it := heap.Pop(&q.heap).(queueItem)
				q.mu.Unlock()
				select {
				case out <- it.item:
				case <-ctx.Done():
					return
				}
				q.mu.Lock()
			}
			q.mu.Unlock()
		}
	}
}
