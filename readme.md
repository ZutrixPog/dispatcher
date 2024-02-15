# Dispatcher Library

The **Dispatcher** library is designed to offer a straightforward yet highly adaptable task dispatching framework. It empowers you to effortlessly spawn various types of tasks, including cronjobs, realtime tasks, background tasks, and those triggered manually. The library comes with a default in-memory queue, and it also supports a Redis queue implementation for added scalability. To ensure comprehensive task tracking, the library allows you to persist task results by integrating a custom **TaskHistoryRepo** implementation, and a PostgreSQL implementation is already provided for your convenience

## Task Definition

Tasks can be defined by implementing the task Interface:
```
type Task interface {
	Type() string
	Retry() int
}
```
As an Example:
```
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
```
type Executor = func(ctx context.Context, task any) error
```
As an Example:
```
func RetrieveData(ctx context.Context, t any) error {
    task := t.(DataRetrievalTask) // embedded data in task

    return ps.Query(task.Query)
}
```

## Task Submission and Execution

After defining your tasks, you can use an instance of TaskDispatcher to register and then submit tasks:
```
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

TODO: document different types of tasks
