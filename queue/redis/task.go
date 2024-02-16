package redis

import (
	"github.com/ZutrixPog/dispatcher"
	"github.com/ZutrixPog/dispatcher/queue"
	"github.com/go-redis/redis"
)

var _ queue.TaskQueue = (*List)(nil)

type List struct {
	client *redis.Client
	limit  int64
}

func NewTaskQueue(client *redis.Client, limit int64) queue.TaskQueue {
	return &List{client, limit}
}

func (q *List) Push(queue string, ts []byte) (int, error) {
	length, err := q.client.LLen(queue).Result()
	if err != nil {
		return 0, dispatcher.ErrCreateEntity
	}
	if length >= q.limit && queue != dispatcher.BgQueue {
		return 0, dispatcher.ErrFullQueue
	}

	if err = q.client.LPush(queue, ts).Err(); err != nil {
		return 0, dispatcher.ErrCreateEntity
	}

	return int(length), nil
}

func (q *List) Pop(queue string) ([]byte, error) {
	data, err := q.client.RPop(queue).Result()
	if err != nil {
		return nil, dispatcher.ErrEmptyQueue
	}

	return []byte(data), err
}

func (q *List) BlockingPop(queue string) <-chan []byte {
	waitchan := make(chan []byte)
	go func() {
		data, err := q.client.BRPop(0, queue).Result()
		if err != nil {
			waitchan <- nil
			return
		}
		waitchan <- []byte(data[1])
	}()

	return waitchan
}

func (q *List) Get(queue string, index int) ([]byte, error) {
	data, err := q.client.LIndex(queue, int64(index)).Result()
	if err != nil {
		return nil, dispatcher.ErrEntityNotFound
	}

	return []byte(data), err
}

func (q *List) List(queue string) ([][]byte, error) {
	data, err := q.client.LRange(queue, 0, q.limit).Result()
	if err != nil {
		return nil, dispatcher.ErrRetrieveEntity
	}
	if len(data) == 0 {
		return nil, dispatcher.ErrEmptyQueue
	}

	ts := make([][]byte, len(data))
	for i, taskData := range data {
		ts[i] = []byte(taskData)
	}

	return ts, nil
}

func (q *List) Remove(queue string, index int) error {
	tag := "DELETED"
	if err := q.client.LSet(queue, int64(index), tag).Err(); err != nil {
		return dispatcher.ErrEntityNotFound
	}

	if err := q.client.LRem(queue, 1, tag).Err(); err != nil {
		return dispatcher.ErrRemoveEntity
	}

	return nil
}
