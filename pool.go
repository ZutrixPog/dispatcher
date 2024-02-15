package dispatcher

import (
	"sync"
	"time"
)

const (
	WORKER_TIMEOUT = time.Second * 30
)

type taskFn func()

type WorkerPool struct {
	jobs       chan taskFn
	maxWorkers int
	workers    int
	done       chan struct{}

	mu sync.Mutex
}

func NewPool(workers int) *WorkerPool {
	workerpool := &WorkerPool{
		jobs:       make(chan taskFn),
		done:       make(chan struct{}),
		workers:    0,
		maxWorkers: workers,
	}

	return workerpool
}

func (w *WorkerPool) Submit(f taskFn) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.workers < w.maxWorkers {
		w.addWorker()
		w.workers += 1
	}
	w.jobs <- f
}

func (w *WorkerPool) addWorker() {
	go func() {
		tout := time.NewTimer(WORKER_TIMEOUT)
		select {
		case job := <-w.jobs:
			job()
		case <-w.done:
			return
		case <-tout.C:
			w.mu.Lock()
			w.workers -= 1
			w.mu.Unlock()
			return
		}
	}()
}

func (w *WorkerPool) Release() {
	w.done <- struct{}{}
}
