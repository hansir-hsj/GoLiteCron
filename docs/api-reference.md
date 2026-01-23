# GoLiteCron API Reference

## Types

### Scheduler

`Scheduler` manages all scheduled tasks and coordinates their execution.

```go
type Scheduler struct {
    // ... private fields
}
```

### Task

`Task` represents a scheduled job with its configuration and status.

```go
type Task struct {
    ID          string
    Job         Job
    NextRunTime time.Time
    PreRunTime  time.Time
    Running     int32
    Removed     int32
}
```

### Job

`Job` is an interface that tasks must implement.

```go
type Job interface {
    Execute() error
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
    Timeout       int64  `yaml:"timeout" json:"timeout"`
    Retry         int    `yaml:"retry" json:"retry"`
    Location      string `yaml:"location" json:"location"`
    EnableSeconds bool   `yaml:"enable_seconds" json:"enable_seconds"`
    EnableYears   bool   `yaml:"enable_years" json:"enable_years"`
    FuncName      string `yaml:"func_name" json:"func_name"`
}
```

### StorageType

Enum for selecting the storage backend.

```go
type StorageType int

const (
    StorageTypeHeap StorageType = iota
    StorageTypeTimeWheel
)
```

## Functions

### NewScheduler

Creates a new scheduler instance.

```go
func NewScheduler(storageType ...StorageType) *Scheduler
```

- `storageType`: Optional. Defaults to `StorageTypeHeap`.

### WrapJob

Wraps a simple function into a `Job` interface.

```go
func WrapJob(id string, fn func() error) Job
```

### RegisterJob / GetJob

Manages job functions for configuration loading.

```go
func RegisterJob(name string, fn func() error)
func GetJob(name string) (func() error, bool)
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

Stops the scheduler and waits for running tasks to complete.

```go
func (s *Scheduler) Stop()
```

### AddTask

Adds a new task to the scheduler.

```go
func (s *Scheduler) AddTask(expr string, job Job, opts ...Option) error
```

### RemoveTask

Removes a task from the scheduler.

```go
func (s *Scheduler) RemoveTask(task *Task) bool
```

### GetTasks

Returns a slice of all currently scheduled tasks.

```go
func (s *Scheduler) GetTasks() []*Task
```

### GetTaskInfo

Returns a string description of a specific task.

```go
func (s *Scheduler) GetTaskInfo(taskID string) string
```

### LoadTasksFromConfig

Loads tasks from a parsed `Config` object.

```go
func (s *Scheduler) LoadTasksFromConfig(config *Config) error
```

### Every (Chain API)

Starts a chain builder for defining tasks.

```go
func (s *Scheduler) Every(intervals ...int) *ScheduleBuilder
```

## Options

Configuration options for `AddTask`.

- `WithSeconds()`: Enables second-level precision (6 fields).
- `WithYears()`: Enables year field (7 fields).
- `WithLocation(loc *time.Location)`: Sets timezone.
- `WithTimeout(timeout time.Duration)`: Sets execution timeout.
- `WithRetry(retry int)`: Sets retry count on failure.
