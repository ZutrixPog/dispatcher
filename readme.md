# Dispatcher Library

The **Dispatcher** library is designed to offer a straightforward yet highly adaptable task dispatching framework. It empowers you to effortlessly spawn various types of tasks, including cronjobs, realtime tasks, background tasks, and those triggered manually. The library comes with a default in-memory queue, and it also supports a Redis queue implementation for added scalability. To ensure comprehensive task tracking, the library allows you to persist task results by integrating a custom **TaskHistoryRepo** implementation, and a PostgreSQL implementation is already provided for your convenience

## Task Definition

Tasks can be defined by implementing the task Interface:
```go
type Task interface {
	Type() string
	Retry() int
}
```
As an Example:
```go
type DataRetrievalTask struct {
    query Query // you can embed any data you need in the execution phase
}

func (task *DataRetrievalTask) Type() string {
    return "task-retrieval"
}

func (task *DataRetrievalTask) Retry() int {
    return 5
}
```
Each task is associated with an executor which uses the embedded data inside the task and executes it. Executors should have the 
following singnature:
```go
type Executor = func(ctx context.Context, task any) error
```
As an Example:
```go
func RetrieveData(ctx context.Context, t any) error {
    task := t.(DataRetrievalTask) // embedded data in task

    return ps.Query(task.Query)
}
```

## Task Submission and Execution

After defining your tasks, you can use an instance of TaskDispatcher to register and then submit tasks. note that task registration using 
```Task()``` method must happen in program entrypoint so that the executors are valid on app restart:
```go
// 10 is the task limit for each queue and 20 is the number of worker goroutines
td := dispatcher.Default(10, 20)

td.Task(&DataRetrievalTask{}, RetrieveData)

td.SpawnBg(DataRetrievalTask{
    // Some Embedded Data
})

// Release resources
td.Release()
```

## Tasks

There are four task spawning methods each specific to a different kind of task:

1. ```Spawn(queue string, task Task) (int, error)```: <br>
    spawns a maually triggered task on the specified channel which can be triggered by executing one of the following methods:
    - ```Dispatch(ctx context.Context, queue string)``` : triggers a single task from the specified queue.
    - ```DispatchAll(ctx context.Context, queue string)``` : triggers all tasks in the specified queue.
    - ```DispatchFilter(ctx context.Context, queue string, t Task)```: triggers a tasks of the provided type in a queue.
    
2. ```SpawnTimer(executor Executor, interval time.Duration)```: <br> spawns a cronjob.

3. ```SpawnBg(task Task) (int, error)```: <br> Spawns a backgroud task that can be persisted and executed by available runners.

4. ```SpawnRealtimeBg(executor RealtimeExecutor)```: <br> submits a task to the worker pool without persisting it in a queue. the task is lost on system restart. 
