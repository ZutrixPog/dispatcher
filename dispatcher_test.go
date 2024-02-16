package dispatcher_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ZutrixPog/dispatcher"
	"github.com/ZutrixPog/dispatcher/history"
	mocks "github.com/ZutrixPog/dispatcher/history/mock"
	"github.com/ZutrixPog/dispatcher/queue/mem"
	"github.com/stretchr/testify/require"
)

type DummyTask struct {
	Msg string
	sth bool
}

func (dt DummyTask) Type() string {
	return "dummy"
}

func (dt DummyTask) Retry() int {
	return 0
}

type LongDummyTask struct {
	Msg string
}

func (dt LongDummyTask) Type() string {
	return "longdummy"
}

func (dt LongDummyTask) Retry() int {
	return 0
}

type DummyTask2 struct {
	Msg string
}

func (dt DummyTask2) Type() string {
	return "dummy2"
}

func (dt DummyTask2) Retry() int {
	return 0
}

type LongDummyExecutor struct{}

func (de *LongDummyExecutor) Execute(ctx context.Context, task any) error {
	ts := task.(*LongDummyTask)
	time.Sleep(2 * time.Second)
	fmt.Println(ts.Msg)
	return nil
}

type DummyExecutor struct{}

func (de *DummyExecutor) Execute(ctx context.Context, task any) error {
	ts := task.(*DummyTask)
	fmt.Println(ts.Msg)
	fmt.Println(ts.sth)
	return nil
}

type FailingTask struct{}

func (dt FailingTask) Type() string {
	return "failing"
}

func (dt FailingTask) Retry() int {
	return 0
}

type RetryingTask struct{}

func (dt RetryingTask) Type() string {
	return "retrying"
}

func (dt RetryingTask) Retry() int {
	return 1
}

type FailingExecutor struct {
}

func (fe *FailingExecutor) Execute(ctx context.Context, task any) error {
	return dispatcher.ErrEmptyID
}

func TestDispatcher(t *testing.T) {
	queue := "queue"
	manager := initDispatcher()
	manager.Spawn(queue, DummyTask2{Msg: ""})

	cases := []struct {
		desc   string
		task   dispatcher.Task
		status string
		err    error
		derr   error
	}{
		{
			desc: "spawn a duplicate tasks",
			task: DummyTask2{
				Msg: "msg",
			},
			status: "success",
			err:    dispatcher.ErrTaskAlreadyExists,
			derr:   nil,
		},
		{
			desc: "spawn a dummy task",
			task: DummyTask{
				Msg: "msg",
				sth: true,
			},
			status: "success",
			err:    nil,
			derr:   nil,
		},
		{
			desc:   "spawn a failing task",
			task:   FailingTask{},
			status: "failed",
			err:    nil,
			derr:   dispatcher.ErrEmptyID,
		},
		{
			desc:   "spawn a retrying task",
			task:   RetryingTask{},
			status: "failed",
			err:    nil,
			derr:   nil,
		},
	}

	for _, c := range cases {
		_, err := manager.Spawn(queue, c.task)
		require.Equal(t, c.err, err)

		err = manager.Dispatch(context.Background(), queue)
		require.Equal(t, c.derr, err, c.desc)

		time.Sleep(10 * time.Millisecond)
		report := manager.RetrieveTaskHistory(context.Background(), history.Query{Limit: 1})
		require.Equal(t, c.status, report[0].Status, c.desc)
		fmt.Println(report)
	}
}

func TestFilteredDispatcher(t *testing.T) {
	queue := "queue"
	manager := initDispatcher()
	manager.Spawn(queue, DummyTask2{Msg: "hey"})
	manager.Spawn(queue, DummyTask{Msg: "geez"})

	cases := []struct {
		desc string
		tlen int
		err  error
	}{
		{
			desc: "dispatch tasks of type DummyTask",
			tlen: 1,
			err:  nil,
		},
	}

	for _, c := range cases {
		err := manager.DispatchFilter(context.Background(), queue, &DummyTask{})
		require.Equal(t, c.err, err, c.desc)

		time.Sleep(1 * time.Second)
		ts := manager.RetrivePendingTasks(context.Background(), queue)
		require.Equal(t, c.tlen, len(ts), c.desc)
	}
}

func TestBgRunner(t *testing.T) {
	manager := initDispatcher()

	cases := []struct {
		desc   string
		task   dispatcher.Task
		status string
		err    error
	}{
		{
			desc: "spawn a dummy task",
			task: LongDummyTask{
				Msg: "ran",
			},
			status: "success",
			err:    nil,
		},
		{
			desc:   "spawn a retrying task",
			task:   RetryingTask{},
			status: "failed",
			err:    nil,
		},
	}

	for _, c := range cases {
		handle, err := manager.SpawnBg(c.task)
		require.Nil(t, err, c.desc)

		require.NoError(t, err, c.desc)
		require.NotNil(t, handle, c.desc)

		time.Sleep(3 * time.Second)
	}
}

func TestRemoval(t *testing.T) {
	queue := "queue"
	manager := initDispatcher()

	id1, _ := manager.Spawn(queue, DummyTask{Msg: "hey"})
	manager.Spawn(queue, DummyTask{Msg: "hey"})
	id2, _ := manager.Spawn(queue, DummyTask2{Msg: "hola"})
	manager.Spawn(queue, DummyTask2{Msg: "hola"})

	cases := []struct {
		desc   string
		taskid int
		len    int
		err    error
	}{
		{
			desc:   "remove a task",
			taskid: id1,
			len:    1,
			err:    nil,
		},
		{
			desc:   "remove another task",
			taskid: id2 - 1,
			len:    0,
			err:    nil,
		},
		{
			desc:   "remove non-existing tasks",
			taskid: 3,
			len:    0,
			err:    dispatcher.ErrEntityNotFound,
		},
	}

	for _, c := range cases {
		err := manager.Remove(queue, c.taskid)
		require.Equal(t, c.err, err, c.desc)

		list := manager.RetrivePendingTasks(context.Background(), queue)
		require.Equal(t, c.len, len(list), c.desc)
	}

	hist := manager.RetrieveTaskHistory(context.Background(), history.Query{Queue: queue, Status: "removed", Limit: 10})
	require.Equal(t, 2, len(hist))
}

func initDispatcher() dispatcher.Dispatcher {
	list := mem.NewQueue(10)
	history := mocks.NewMockHistoryRepo()
	manager := dispatcher.Init(list, history, 5)

	d := &DummyExecutor{}
	ld := &LongDummyExecutor{}
	f := &FailingExecutor{}

	timerFunc := func(ctx context.Context, t any) error {
		fmt.Println("timer")
		return nil
	}

	manager.SpawnTimer(timerFunc, time.Second*60)
	manager.Task(&DummyTask{}, d.Execute)
	manager.Task(&LongDummyTask{}, ld.Execute)
	manager.Task(&DummyTask2{}, func(ctx context.Context, task any) error {
		ts := task.(*DummyTask2)
		fmt.Println(ts.Msg)
		return nil
	})
	manager.Task(&FailingTask{}, f.Execute)
	manager.Task(&RetryingTask{}, f.Execute)

	return manager
}
