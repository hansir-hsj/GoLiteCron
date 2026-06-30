# GoLiteCron API 参考

## 类型

### Scheduler

`Scheduler` 管理所有调度任务并协调它们的执行。

```go
type Scheduler struct {
    // ... 私有字段
}
```

### TaskInfo

`TaskInfo` 是 `GetTasks` 返回的只读任务快照。

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

`Job` 是任务必须实现的接口。

```go
type Job interface {
    Execute(ctx context.Context) error
    ID() string
}
```

### Config & TaskConfig

用于从文件加载配置的结构体。

```go
type Config struct {
    Tasks []TaskConfig `yaml:"tasks" json:"tasks"`
}

type TaskConfig struct {
    ID            string `yaml:"id" json:"id"`
    CronExpr      string `yaml:"cron_expr" json:"cron_expr"`
    Timeout       string `yaml:"timeout" json:"timeout"` // duration 字符串, 如 "30s"
    Retry         int    `yaml:"retry" json:"retry"`
    Location      string `yaml:"location" json:"location"`
    EnableSeconds bool   `yaml:"enable_seconds" json:"enable_seconds"`
    EnableYears   bool   `yaml:"enable_years" json:"enable_years"`
    FuncName      string `yaml:"func_name" json:"func_name"`
}
```

### Logger

自定义日志接口。如未设置，默认输出到 `os.Stderr`。

```go
type Logger interface {
    Printf(format string, args ...any)
}
```

## 函数

### NewScheduler

创建一个新的调度器实例。

```go
func NewScheduler() *Scheduler
```

### WrapJob

将简单函数包装为 `Job` 接口。

```go
func WrapJob(id string, fn any) (Job, error)
// fn 支持: func() error 或 func(context.Context) error
```

### RegisterJob / GetJob

管理当前 scheduler 实例用于配置加载的作业函数。

```go
func (s *Scheduler) RegisterJob(name string, fn any) error
func (s *Scheduler) GetJob(name string) (any, bool)
```

### LoadFromYaml / LoadFromJson

解析配置文件。

```go
func LoadFromYaml(path string) (*Config, error)
func LoadFromJson(path string) (*Config, error)
```

## 调度器方法

### Start

在后台 goroutine 中启动调度器。

```go
func (s *Scheduler) Start()
```

### Stop

停止调度循环并等待循环退出。它会阻止新的执行被调度，但不会等待已经运行中的任务完成。

```go
func (s *Scheduler) Stop()
```

### Shutdown

停止调度器，取消传给正在运行且支持 context 的任务的 context，并等待已跟踪的 goroutine 完成，或直到 context 被取消。

```go
func (s *Scheduler) Shutdown(ctx context.Context) error
```

### AddTask

向调度器添加新任务。

```go
func (s *Scheduler) AddTask(expr string, job Job, opts ...TaskOption) error
```

### RemoveTaskByID

按 ID 从调度器中移除任务。推荐使用这个方法，因为它也会阻止正在运行的同 ID 任务在结束后重新调度。

```go
func (s *Scheduler) RemoveTaskByID(taskID string) bool
```

### GetTasks

返回当前已调度任务的只读快照。修改返回值不会改变调度器内部状态。

```go
func (s *Scheduler) GetTasks() []TaskInfo
```

### GetTask

返回指定已调度任务的只读快照。

```go
func (s *Scheduler) GetTask(taskID string) (TaskInfo, bool)
```

### LoadTasksFromConfig

从解析后的 `Config` 对象加载任务。加载过程是原子的：只要任一任务无效，就不会添加该配置中的任何任务。

```go
func (s *Scheduler) LoadTasksFromConfig(config *Config) error
```

### Every (链式 API)

启动链式构建器以定义任务。

```go
func (s *Scheduler) Every(intervals ...int) *ScheduleBuilder
```

### WithLogger

为调度器设置自定义日志记录器。必须在 `Start()` 前调用。

```go
func (s *Scheduler) WithLogger(l Logger) error
```

## 选项

用于 `AddTask` 的配置选项。`WithTimeout` 和 `WithRetry` 是仅用于任务的选项，不能传给 `Parse`。

- `WithSeconds()`: 启用秒级精度（6字段）。
- `WithYears()`: 启用年份字段（7字段）。
- `WithLocation(loc *time.Location)`: 设置时区。
- `WithTimeout(timeout time.Duration)`: 设置调度器等待单次执行的时间。接收 `context.Context` 的任务应在 context 结束时停止；忽略 context 的任务可能继续运行到函数返回。
- `WithRetry(retry int)`: 设置失败时的重试次数。
