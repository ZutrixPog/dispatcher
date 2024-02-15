package mem

import (
	"sync"

	tq "github.com/ZutrixPog/dispatcher/queue"
)

const BgChannel = "bg-channel"

var _ tq.TaskQueue = (*MemQueue)(nil)

type MemQueue struct {
	data    map[string][]string
	blocked map[string]*sync.Cond
	limit   int64
	lock    sync.RWMutex
}

func NewQueue(limit int64) tq.TaskQueue {
	return &MemQueue{
		data:    make(map[string][]string),
		blocked: make(map[string]*sync.Cond),
		limit:   limit,
	}
}

func (q *MemQueue) Push(queue string, ts []byte) (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.data[queue]) >= int(q.limit) && queue != BgChannel {
		return 0, tq.ErrFullQueue
	}

	q.data[queue] = append(q.data[queue], string(ts))
	if _, ok := q.blocked[queue]; ok {
		q.blocked[queue].Signal()
	}
	return len(q.data[queue]) - 1, nil
}

func (q *MemQueue) Pop(queue string) ([]byte, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.data[queue]) == 0 {
		return nil, tq.ErrEmptyQueue
	}

	item := q.data[queue][len(q.data[queue])-1]
	q.data[queue] = q.data[queue][:len(q.data[queue])-1]

	return []byte(item), nil
}

func (q *MemQueue) BlockingPop(queue string) <-chan []byte {
	waitchan := make(chan []byte)
	go func() {
		q.lock.Lock()
		defer q.lock.Unlock()
		for len(q.data[queue]) == 0 {
			q.blocked[queue] = sync.NewCond(&q.lock)
			q.blocked[queue].Wait()
		}

		item := q.data[queue][len(q.data[queue])-1]
		q.data[queue] = q.data[queue][:len(q.data[queue])-1]
		waitchan <- []byte(item)
	}()

	return waitchan
}

func (q *MemQueue) Get(queue string, index int) ([]byte, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	if index < 0 || index >= len(q.data[queue]) {
		return nil, tq.ErrEntityNotFound
	}

	item := q.data[queue][index]
	return []byte(item), nil
}

func (q *MemQueue) List(queue string) ([][]byte, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	items := q.data[queue]
	if len(items) == 0 {
		return nil, tq.ErrEmptyQueue
	}

	result := make([][]byte, len(items))
	for i, item := range items {
		result[i] = []byte(item)
	}

	return result, nil
}

func (q *MemQueue) Remove(queue string, index int) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if index < 0 || index >= len(q.data[queue]) {
		return tq.ErrEntityNotFound
	}

	q.data[queue][index] = "DELETED"

	cleanData := []string{}
	for _, item := range q.data[queue] {
		if item != "DELETED" {
			cleanData = append(cleanData, item)
		}
	}
	q.data[queue] = cleanData

	return nil
}
