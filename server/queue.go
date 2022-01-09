package server

import (
	"errors"
	"fmt"
	"sync"
)

var errNoJobsAvailable = errors.New("no jobs available")

type jobQueue[T any] struct {
	sync.Mutex
	waiters []chan struct{}
	queue   []T
	maxSize int
}

func newJobQueue[T any](maxSize int) *jobQueue[T] {
	return &jobQueue[T]{
		maxSize: maxSize,
	}
}

func (q *jobQueue[T]) push(job T) error {
	q.Lock()
	defer q.Unlock()
	if len(q.queue) > q.maxSize {
		return fmt.Errorf("queue full")
	}
	q.queue = append(q.queue, job)
	for _, waiter := range q.waiters {
		close(waiter)
	}
	q.waiters = nil
	return nil
}

func (q *jobQueue[T]) wait() chan struct{} {
	q.Lock()
	defer q.Unlock()
	ch := make(chan struct{})
	q.waiters = append(q.waiters, ch)
	return ch
}

func (q *jobQueue[T]) takeJob() (res T, err error) {
	q.Lock()
	defer q.Unlock()
	if len(q.queue) == 0 {
		return res, errNoJobsAvailable
	}
	res = q.queue[0]
	q.queue = q.queue[1:]
	return res, nil
}

func (q *jobQueue[T]) listJobs() []T {
	q.Lock()
	defer q.Unlock()
	ret := make([]T, len(q.queue))
	copy(ret, q.queue)
	return ret
}
