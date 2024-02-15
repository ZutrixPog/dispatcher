package queue

import "errors"

var (
	ErrEmptyID           = errors.New("empty ID")
	ErrUnregisteredTask  = errors.New("task is not registered")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrEntityNotFound    = errors.New("entity not found")
	ErrFullQueue         = errors.New("queue is full")
	ErrEmptyQueue        = errors.New("queue is empty")
	ErrRetrieveEntity    = errors.New("failed to retrieve entity")
	ErrCreateEntity      = errors.New("failed to create entity")
	ErrRemoveEntity      = errors.New("failed to remove entity")
)

type TaskQueue interface {
	// Push pushes a task to the queue.
	Push(queue string, task []byte) (int, error)

	// Removes a task from the queue given the index
	Remove(queue string, index int) error

	// Pop pops a task from the queue.
	Pop(queue string) ([]byte, error)

	BlockingPop(queue string) <-chan []byte

	// Get gets a task with a specific ID from the queue
	Get(queue string, id int) ([]byte, error)

	// List lists all tasks in the queue
	List(queue string) ([][]byte, error)
}
