package dispatcher

import "errors"

var (
	ErrTaskNotPtr        = errors.New("task refrence is required for registration")
	ErrWrongType         = errors.New("wrong task type")
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
