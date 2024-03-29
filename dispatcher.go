package dispatcher

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/ZutrixPog/dispatcher/history"
	"github.com/ZutrixPog/dispatcher/queue"
	"github.com/ZutrixPog/dispatcher/queue/mem"
	serial "github.com/ZutrixPog/dispatcher/serialization"
)

const (
	BgQueue      = "bg-queue"
	BG_PRODUCERS = 5
)

type Dispatcher interface {
	Task(task any, executor Executor) error

	Spawn(queue string, task Task) (int, error)

	SpawnTimer(executor Executor, interval time.Duration)

	SpawnBg(task Task) (int, error)

	SpawnRealtimeBg(executor RealtimeExecutor)

	Dispatch(ctx context.Context, queue string) error

	DispatchAll(ctx context.Context, queue string)

	DispatchFilter(ctx context.Context, queue string, t Task) error

	RetrivePendingTasks(ctx context.Context, queue string) []history.TaskReport

	Remove(queue string, index int) error

	RetrieveTaskHistory(ctx context.Context, query history.Query) []history.TaskReport

	Release()
}

type TaskDispatcher struct {
	queue   queue.TaskQueue
	history history.TaskHistoryRepo
	pool    *WorkerPool
	ctx     context.Context
	cancel  context.CancelFunc

	types     *sync.Map
	executors *sync.Map
}

func Default(runners int, limit int64) Dispatcher {
	return Init(mem.NewQueue(limit), &history.DummyTaskHistoryRepo{}, runners)
}

func Init(queue queue.TaskQueue, historyrepo history.TaskHistoryRepo, runners int) Dispatcher {
	if historyrepo == nil {
		historyrepo = &history.DummyTaskHistoryRepo{}
	}

	pool := NewPool(runners)
	ctx, cancel := context.WithCancel(context.Background())
	d := &TaskDispatcher{
		queue,
		historyrepo,
		pool,
		ctx,
		cancel,
		&sync.Map{},
		&sync.Map{},
	}
	for i := 0; i < BG_PRODUCERS; i++ {
		d.initRunner()
	}

	return d
}

func (tm *TaskDispatcher) Task(task any, executor Executor) error {
	if !isPointer(task) {
		return ErrTaskNotPtr
	}
	if _, ok := task.(Task); !ok {
		return ErrWrongType
	}

	tm.types.Store(task.(Task).Type(), task)
	tm.executors.Store(task.(Task).Type(), executor)
	return nil
}

func (tm *TaskDispatcher) SpawnTimer(executor Executor, interval time.Duration) {
	ticker := time.NewTicker(interval)
	tm.spawnTimer(executor, ticker)
}

func (tm *TaskDispatcher) initRunner() {
	go func() {
		for {
			select {
			case <-tm.ctx.Done():
				return
			case task := <-tm.queue.BlockingPop(BgQueue):
				decodedTask, wrapper, err := tm.deserialize(task)
				if err != nil {
					continue
				}

				tm.pool.Submit(func() {
					execute, _ := tm.executors.Load(wrapper.Type)

					if _, ok := execute.(Executor); !ok {
						return
					}
					err = execute.(Executor)(tm.ctx, decodedTask)
					if err != nil && wrapper.Retries > 0 {
						wrapper.Retries--
						encodedTask, err := serial.Serialize(decodedTask)
						if err == nil {
							tm.queue.Push(BgQueue, encodedTask)
						}
					}
				})
			}
		}
	}()
}

func (tm *TaskDispatcher) spawnTimer(exec Executor, ticker *time.Ticker) {
	go func() {
		for {
			select {
			case <-tm.ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				exec(tm.ctx, nil)
			}
		}
	}()
}

func (tm *TaskDispatcher) Spawn(queue string, task Task) (int, error) {
	if _, exists := tm.types.Load(task.Type()); !exists {
		return 0, ErrUnregisteredTask
	}
	if tm.TaskExists(context.Background(), queue, task.Type()) {
		return 0, ErrTaskAlreadyExists
	}

	encodedTask, err := serial.Serialize(task)
	if err != nil {
		return 0, err
	}

	data, err := serial.Serialize(TaskWrapper{Type: task.Type(), Task: encodedTask, Submitted: time.Now(), Retries: task.Retry()})
	if err != nil {
		return 0, err
	}
	index, err := tm.queue.Push(queue, data)
	if err != nil {
		return 0, err
	}

	return index, nil
}

func (tm *TaskDispatcher) TaskExists(ctx context.Context, queue, taskType string) bool {
	if queue == BgQueue {
		return false
	}
	tasks := tm.RetrivePendingTasks(ctx, queue)

	for i := range tasks {
		if tasks[i].Type == taskType {
			return true
		}
	}

	return false
}

func (tm *TaskDispatcher) SpawnBg(task Task) (int, error) {
	i, err := tm.Spawn(BgQueue, task)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (tm *TaskDispatcher) SpawnRealtimeBg(exec RealtimeExecutor) {
	tm.pool.Submit(func() {
		exec(tm.ctx)
	})
}

func (tm *TaskDispatcher) Dispatch(ctx context.Context, queue string) error {
	task, err := tm.queue.Pop(queue)
	if err != nil {
		return err
	}

	return tm.dispatch(ctx, task, queue)

}

func (tm *TaskDispatcher) dispatch(ctx context.Context, task []byte, queue string) error {
	decodedTask, wrapper, err := tm.deserialize(task)
	if err != nil {
		return err
	}

	report := history.TaskReport{
		Type:      decodedTask.(Task).Type(),
		Status:    "success",
		Queue:     queue,
		Submitted: wrapper.Submitted,
	}

	execute, _ := tm.executors.Load(wrapper.Type)

	err = execute.(Executor)(ctx, decodedTask)
	if err != nil {
		if decodedTask.(Task).Retry() > 0 {
			tm.queue.Push(queue, task)
			return nil
		} else {
			report.Status = "failed"
		}
	}

	if report.Queue != BgQueue {
		go tm.history.Append(context.Background(), report)
	}
	return err
}

func (tm *TaskDispatcher) DispatchFilter(ctx context.Context, queue string, t Task) error {
	tlist, err := tm.queue.List(queue)
	if err != nil {
		return err
	}

	for i, task := range tlist {
		_, wrapper, err := tm.deserialize(task)
		if err != nil {
			return err
		}

		if wrapper.Type == t.Type() {
			tm.dispatch(ctx, task, queue)
			tm.queue.Remove(queue, i)
			return nil
		}
	}

	return nil
}

func (tm *TaskDispatcher) DispatchAll(ctx context.Context, queue string) {
	for {
		err := tm.Dispatch(ctx, queue)
		if err != nil {
			return
		}
	}
}

func (tm *TaskDispatcher) RetrivePendingTasks(ctx context.Context, queue string) []history.TaskReport {
	tasks, err := tm.queue.List(queue)
	if err != nil {
		return nil
	}

	res := make([]history.TaskReport, len(tasks))
	for i := range tasks {
		_, wrapper, _ := tm.deserialize(tasks[i])

		res[i] = history.TaskReport{
			ID:        uint(i),
			Type:      wrapper.Type,
			Status:    "pending",
			Queue:     queue,
			Submitted: wrapper.Submitted.UTC(),
		}
	}

	return res
}

func (tm *TaskDispatcher) Remove(queue string, index int) error {
	data, err := tm.queue.Get(queue, index)
	if err != nil {
		return err
	}

	task, wrapper, err := tm.deserialize(data)
	if err != nil {
		return err
	}

	tm.history.Append(context.Background(), history.TaskReport{
		Type:      task.(Task).Type(),
		Status:    "removed",
		Queue:     queue,
		Submitted: wrapper.Submitted.UTC(),
	})
	return tm.queue.Remove(queue, index)
}

func (tm *TaskDispatcher) RetrieveTaskHistory(ctx context.Context, query history.Query) []history.TaskReport {
	history, _ := tm.history.Retrieve(ctx, query)
	pending := tm.RetrivePendingTasks(ctx, query.Queue)

	pending = append(pending, history...)

	return pending
}

func (tm *TaskDispatcher) deserialize(task []byte) (any, TaskWrapper, error) {
	var wrapper TaskWrapper
	if err := serial.Deserialize(task, &wrapper); err != nil {
		return nil, TaskWrapper{}, err
	}

	t, _ := tm.types.Load(wrapper.Type)
	if err := serial.Deserialize(wrapper.Task, t); err != nil {
		return nil, TaskWrapper{}, err
	}
	decodedTask, _ := tm.types.Load(wrapper.Type)

	return decodedTask, wrapper, nil
}

func (tm *TaskDispatcher) Release() {
	tm.cancel()
	if tm.pool != nil {
		tm.pool.Release()
	}
}

func isPointer(value any) bool {
	return reflect.TypeOf(value).Kind() == reflect.Ptr
}
