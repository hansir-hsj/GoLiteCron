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
    Removed     bool // Deprecated
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
    RetryInterval string `yaml:"retry_interval" json:"retry_interval"` // duration string, e.g. "2s"
    Location      string `yaml:"location" json:"location"`             // IANA name; empty means the host's local zone
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
// fn supports: func(), func() error, func(context.Context) error, or Job
```

### RegisterJob / GetJob

Manages scheduler-local jobs for configuration loading. `fn` may be a `Job`,
`func()`, `func() error`, or `func(context.Context) error`.

```go
func (s *Scheduler) RegisterJob(name string, fn any) error
func (s *Scheduler) GetJob(name string) (any, bool)
```

### LoadFromYAML / LoadFromJSON

Parses configuration files. Unknown keys are rejected. `LoadFromYaml` and `LoadFromJson` are deprecated aliases.

```go
func LoadFromYAML(path string) (*Config, error)
func LoadFromJSON(path string) (*Config, error)
```

## Scheduler Methods

### Start

Starts the scheduler in a background goroutine.

```go
func (s *Scheduler) Start()
```

### Stop

Stops the scheduler loop and waits for it to exit. It prevents new executions from being scheduled, but does not wait for already running tasks. Running tasks retain their next schedule after they finish; use `RemoveTaskByID` for permanent removal.

```go
func (s *Scheduler) Stop()
```

### Shutdown

Stops the scheduler, cancels contexts passed to running context-aware jobs, and waits for tasks started before the call to finish or until the context is canceled. Tasks keep their schedules, and `Start` may be called again without waiting for the drain. Calling it from a Job with that Job's context returns `ErrShutdownFromJob`. If tasks are running and `ctx` has no deadline, it stops the scheduler without waiting and returns `ErrShutdownDeadlineRequired`.

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

Returns read-only snapshots of all currently scheduled tasks, including running tasks. Mutating returned values never changes scheduler state.

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

Returns an error for a nil logger or when called while the scheduler is running.

### WithTickDuration

Sets how often the scheduler checks for due tasks. The default is 500 ms. Call it while the scheduler is stopped.

```go
func (s *Scheduler) WithTickDuration(duration time.Duration) error
```

## Options

Configuration options for `AddTask`. `WithTimeout` and `WithRetry` are task-only options and cannot be passed to `Parse`.

- `WithSeconds()`: Enables second-level precision (6 fields).
- `WithYears()`: Enables year field (7 fields).
- `WithLocation(loc *time.Location)`: Sets timezone. The default is the host's local zone (`time.Local`), so set it explicitly in production.
- `WithTimeout(timeout time.Duration)`: Sets how long the scheduler waits before reporting a timeout. Jobs that accept `context.Context` should stop when the context is done. A job that ignores it remains tracked and is not scheduled again until it returns. Timeouts are not retried.
- `WithRetry(retry int)`: Sets the number of retries on failure. Retries run immediately unless `WithRetryInterval` is set.
- `WithRetryInterval(interval time.Duration)`: Sets the wait between retry attempts.

## Behavior Changes

These changes can break existing callers when upgrading:

- `Shutdown` with no `ctx` deadline returns `ErrShutdownDeadlineRequired` after stopping if tasks are running. Pass a context with a timeout to wait.
- `WithLogger(nil)` returns an error instead of being ignored.
- `At()` combined with `Second`, `Minute`, or `Hour` units, and `Every(n)` with `n > 1` on weekday units, make `Do` return an error instead of silently ignoring the setting.
- `Every(n)` on `Second`, `Minute`, or `Hour` returns an error unless `n` divides 60 (or 24 for hours), because steps align to the clock and other values fire at uneven gaps.
- `ScheduleBuilder.Do` accepts the same function types as `WrapJob`, including `func(context.Context) error`.
- `LoadFromYAML` and `LoadFromJSON` reject unknown keys, and a negative `retry`, `timeout`, or `retry_interval` is an error.
- A removed task that is still running no longer holds its ID; the same ID can be added again immediately.
- A day-of-month or day-of-week field that starts with `*` (for example `*/2`) combines with the other day field using AND, as in Vixie cron. Two restricted fields still use OR.
- Day of week accepts `weekday#n` (for example `MON#2`, the second Monday).
- A number before `L` in day of month (for example `5L`) is rejected; use a bare `L`.
- Day of week accepts `7` as Sunday, and month and weekday names such as `JAN` and `MON`.
- A panicking custom logger no longer stops scheduling; the message is written to stderr instead.
