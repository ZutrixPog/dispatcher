package dispatcher

import (
	"context"
	"time"
)

type Task interface {
	Type() string
	Retry() int
}

type TaskWrapper struct {
	Type      string
	Submitted time.Time
	Task      []byte
	Retries   int
}

type Executor = func(ctx context.Context, task any) error
