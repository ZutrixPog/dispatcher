package redis_test

import (
	"testing"

	"github.com/ZutrixPog/dispatcher/queue/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors "github.com/ZutrixPog/dispatcher"
)

var (
	task1 = []byte{}
	task2 = []byte{}
)

func TestList_Push(t *testing.T) {
	queue := redis.NewTaskQueue(client, 10)

	fullQueue := "full"
	for i := 0; i < 11; i++ {
		queue.Push(fullQueue, task1)
	}

	cases := []struct {
		desc        string
		queue       string
		task        []byte
		expectedErr error
	}{
		{
			desc:        "Push task",
			queue:       "queue",
			task:        task1,
			expectedErr: nil,
		},
		{
			desc:        "Push task to a full queue",
			queue:       fullQueue,
			task:        task1,
			expectedErr: errors.ErrFullQueue,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			id, err := queue.Push(c.queue, c.task)
			if c.expectedErr != nil {
				require.Error(t, err, c.desc)
				assert.Equal(t, c.expectedErr, err, c.desc)
			} else {
				require.NoError(t, err, c.desc)
				assert.Equal(t, id, 0, c.desc)
				task, err := queue.Get("queue", 0)
				assert.Nil(t, err)
				assert.NotNil(t, task)
			}
		})
	}
}

func TestList_Pop(t *testing.T) {
	queue := redis.NewTaskQueue(client, 10)

	nonEmptyQueue := "queue"
	queue.Push(nonEmptyQueue, task1)

	cases := []struct {
		desc        string
		queue       string
		expectedErr error
	}{
		{
			desc:        "Pop task from non-empty queue",
			queue:       nonEmptyQueue,
			expectedErr: nil,
		},
		{
			desc:        "Pop task from an empty queue",
			queue:       "emptyQueue",
			expectedErr: errors.ErrEmptyQueue,
		},
		{
			desc:        "Pop task from a non-existent queue",
			queue:       "nonExistentQueue",
			expectedErr: errors.ErrEmptyQueue,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			_, err := queue.Pop(c.queue)
			if c.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, c.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

}

func TestList_Get(t *testing.T) {
	queue := redis.NewTaskQueue(client, 10)

	validQueue := "queue"
	queue.Push(validQueue, task1)
	queue.Push(validQueue, task2)

	cases := []struct {
		desc        string
		queue       string
		index       int
		expected    []byte
		expectedErr error
	}{
		{
			desc:        "Get task from a valid queue",
			queue:       validQueue,
			index:       0,
			expected:    task2,
			expectedErr: nil,
		},
		{
			desc:        "Get task from a non-existent queue",
			queue:       "nonExistentQueue",
			index:       0,
			expected:    nil,
			expectedErr: errors.ErrEntityNotFound,
		},
		{
			desc:        "Get task at an invalid index",
			queue:       validQueue,
			index:       100,
			expected:    nil,
			expectedErr: errors.ErrEntityNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ts, err := queue.Get(c.queue, c.index)
			if c.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, c.expectedErr, err)
			} else {
				require.Equal(t, c.expected, ts)
				require.NoError(t, err)
			}
		})
	}
}

func TestList_List(t *testing.T) {
	queue := redis.NewTaskQueue(client, 10)

	validQueue := "queue"
	for i := 0; i < 5; i++ {
		queue.Push(validQueue, task1)
	}

	cases := []struct {
		desc        string
		queue       string
		len         int
		expectedErr error
	}{
		{
			desc:        "List tasks from a valid queue",
			queue:       validQueue,
			len:         5,
			expectedErr: nil,
		},
		{
			desc:        "List tasks from a non-existent queue",
			queue:       "nonExistentQueue",
			len:         0,
			expectedErr: errors.ErrEmptyQueue,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ts, err := queue.List(c.queue)
			if c.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, c.expectedErr, err)
			} else {
				require.GreaterOrEqual(t, len(ts), c.len)
				require.NoError(t, err)
			}
		})
	}
}

func TestList_Remove(t *testing.T) {
	queue := redis.NewTaskQueue(client, 10)

	nonEmptyQueue := "queue"
	id, _ := queue.Push(nonEmptyQueue, task2)
	queue.Push(nonEmptyQueue, task1)

	cases := []struct {
		desc        string
		queue       string
		index       int
		expectedErr error
	}{
		{
			desc:        "Remove task from a non-empty queue",
			queue:       nonEmptyQueue,
			index:       id,
			expectedErr: nil,
		},
		{
			desc:        "Remove task from a non-existent queue",
			queue:       "nonExistentQueue",
			index:       0,
			expectedErr: errors.ErrEntityNotFound,
		},
		{
			desc:        "Remove task at an invalid index",
			queue:       nonEmptyQueue,
			index:       100,
			expectedErr: errors.ErrEntityNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := queue.Remove(c.queue, c.index)
			if c.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, c.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	task, _ := queue.Pop(nonEmptyQueue)
	require.Equal(t, task1, task)
}
