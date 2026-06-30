# GoLiteCron API Reference

## Types

### Scheduler

`Scheduler` manages all scheduled tasks and coordinates their execution.

```go
type Scheduler struct {
    // ... private fields
}
```

### TaskInfo

`TaskInfo` is a read-only snapshot returned by `GetTasks`.

```go
type TaskInfo struct {
    ID          string
    NextRunTime time.Time
    PreRunTime  time.Time
    Running     bool
    Removed     bool
    Location    *time.Location
    Timeout     time.Duration
    Retry       int
}
```

### Job

`Job` is an interface that tasks must implement.

```go
type Job interface {
    Execute(ctx context.Context) error
    ID() string
}
```

### Config & TaskConfig

Structures for loading configuration from files.

```go
type Config struct {
    Tasks []TaskConfig `yaml:"tasks" json:"tasks"`
}

type TaskConfig struct {
    ID            string `yaml:"id" json:"id"`
    CronExpr      string `yaml:"cron_expr" json:"cron_expr"`
    Timeout       string `yaml:"timeout" json:"timeout"` // duration string, e.g. "30s"
    Retry         int    `yaml:"retry" json:"retry"`
    Location      string `yaml:"location" json:"location"`
    EnableSeconds bool   `yaml:"enable_seconds" json:"enable_seconds"`
    EnableYears   bool   `yaml:"enable_years" json:"enable_years"`
    FuncName      string `yaml:"func_name" json:"func_name"`
}
```

### Logger

Interface for custom logging. If not set, defaults to writing to `os.Stderr`.

```go
type Logger interface {
    Printf(format string, args ...any)
}
```

## Functions

### NewScheduler

Creates a new scheduler instance.

```go
func NewScheduler() *Scheduler
```


### WrapJob

Wraps a simple function into a `Job` interface.

```go
func WrapJob(id string, fn any) (Job, error)
// fn supports: func() error or func(context.Context) error
```

### RegisterJob / GetJob

Manages scheduler-local job functions for configuration loading.

```go
func (s *Scheduler) RegisterJob(name string, fn any) error
func (s *Scheduler) GetJob(name string) (any, bool)
```

### LoadFromYaml / LoadFromJson

Parses configuration files.

```go
func LoadFromYaml(path string) (*Config, error)
func LoadFromJson(path string) (*Config, error)
```

## Scheduler Methods

### Start

Starts the scheduler in a background goroutine.

```go
func (s *Scheduler) Start()
```

### Stop

Stops the scheduler loop and waits for it to exit. It prevents new executions from being scheduled, but does not wait for already running tasks.

```go
func (s *Scheduler) Stop()
```

### Shutdown

Stops the scheduler, cancels contexts passed to running context-aware jobs, and waits for tracked goroutines to finish or until the context is canceled.

```go
func (s *Scheduler) Shutdown(ctx context.Context) error
```

### AddTask

Adds a new task to the scheduler.

```go
func (s *Scheduler) AddTask(expr string, job Job, opts ...TaskOption) error
```

### RemoveTaskByID

Removes a task from the scheduler by ID. This is the recommended removal
method because it also prevents a currently running task with the same ID from
being rescheduled after completion.

```go
func (s *Scheduler) RemoveTaskByID(taskID string) bool
```

### GetTasks

Returns read-only snapshots of all currently scheduled tasks. Mutating returned values never changes scheduler state.

```go
func (s *Scheduler) GetTasks() []TaskInfo
```

### GetTask

Returns a read-only snapshot for one scheduled task.

```go
func (s *Scheduler) GetTask(taskID string) (TaskInfo, bool)
```

### LoadTasksFromConfig

Loads tasks from a parsed `Config` object. Loading is atomic: if any task is invalid, no tasks from the config are added.

```go
func (s *Scheduler) LoadTasksFromConfig(config *Config) error
```

### Every (Chain API)

Starts a chain builder for defining tasks.

```go
func (s *Scheduler) Every(intervals ...int) *ScheduleBuilder
```

### WithLogger

Sets a custom logger for the scheduler. Must be called before `Start()`.

```go
func (s *Scheduler) WithLogger(l Logger) error
```

## Options

Configuration options for `AddTask`. `WithTimeout` and `WithRetry` are task-only options and cannot be passed to `Parse`.

- `WithSeconds()`: Enables second-level precision (6 fields).
- `WithYears()`: Enables year field (7 fields).
- `WithLocation(loc *time.Location)`: Sets timezone.
- `WithTimeout(timeout time.Duration)`: Sets how long the scheduler waits for one execution. Jobs that accept `context.Context` should stop when the context is done; jobs that ignore it may continue running until they return.
- `WithRetry(retry int)`: Sets retry count on failure.
