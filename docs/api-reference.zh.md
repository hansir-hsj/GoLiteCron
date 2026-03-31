# GoLiteCron API 参考

## 类型

### Scheduler

`Scheduler` 管理所有调度任务并协调它们的执行。

```go
type Scheduler struct {
    // ... 私有字段
}
```

### Task

`Task` 表示一个带有配置和状态的调度作业。

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

### StorageType

用于选择存储后端的枚举。

```go
type StorageType int

const (
    StorageTypeHeap StorageType = iota
    StorageTypeTimeWheel
)
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
func NewScheduler(storageType ...StorageType) *Scheduler
```

- `storageType`: 可选。默认为 `StorageTypeHeap`。

### WrapJob

将简单函数包装为 `Job` 接口。

```go
func WrapJob(id string, fn any) (Job, error)
// fn 支持: func() error 或 func(context.Context) error
```

### RegisterJob / GetJob

管理用于配置加载的作业函数。

```go
func RegisterJob(name string, fn any)    // fn: func() error 或 func(context.Context) error
func GetJob(name string) (any, bool)     // 返回 fn 或 nil, ok
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

停止调度器并等待运行中的任务完成。

```go
func (s *Scheduler) Stop()
```

### AddTask

向调度器添加新任务。

```go
func (s *Scheduler) AddTask(expr string, job Job, opts ...Option) error
```

### RemoveTask

从调度器中移除任务。

```go
func (s *Scheduler) RemoveTask(task *Task) bool
```

### GetTasks

返回当前所有调度任务的切片。

```go
func (s *Scheduler) GetTasks() []*Task
```

### GetTaskInfo

返回特定任务的字符串描述。

```go
func (s *Scheduler) GetTaskInfo(taskID string) string
```

### LoadTasksFromConfig

从解析后的 `Config` 对象加载任务。

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
func (s *Scheduler) WithLogger(l Logger)
```

## 选项

用于 `AddTask` 的配置选项。

- `WithSeconds()`: 启用秒级精度（6字段）。
- `WithYears()`: 启用年份字段（7字段）。
- `WithLocation(loc *time.Location)`: 设置时区。
- `WithTimeout(timeout time.Duration)`: 设置执行超时。
- `WithRetry(retry int)`: 设置失败时的重试次数。