# GoLiteCron 使用指南

本文档包含GoLiteCron的完整使用示例和API文档。

## 目录
- [快速开始](#快速开始)
- [高级用法](#高级用法)
- [链式API](#链式api)
- [配置](#配置)
- [任务管理](#任务管理)
- [API参考](#api参考)
- [最佳实践](#最佳实践)

## 快速开始

### 基础示例

```go
package main

import (
	"fmt"
	"time"
	golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	// 创建一个使用 TimeWheel 存储的调度器
	scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
	
	// 添加一个每分钟运行的任务
	err := scheduler.AddTask("@minutely", golitecron.WrapJob("minute-task", func() error {
		fmt.Printf("分钟任务在 %s 执行\n", time.Now().Format(time.RFC3339))
		return nil
	}))
	
	if err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	
	// 启动调度器
	scheduler.Start()
	defer scheduler.Stop()
	
	// 保持程序运行
	select {}
}
```

## 高级用法

### 使用自定义 Cron 表达式

```go
// 添加一个每30分钟运行一次的任务
scheduler.AddTask("*/30 * * * *", golitecron.WrapJob("30min-task", func() error {
    fmt.Println("每30分钟运行一次的任务")
    return nil
}))

// 添加一个每2秒运行一次的任务（需要 WithSeconds 选项）
scheduler.AddTask("0/2 * * * * *", golitecron.WrapJob("2sec-task", func() error {
    fmt.Println("每2秒运行一次的任务")
    return nil
}), golitecron.WithSeconds())
```

### 配置任务选项

```go
// 创建上海时区
shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

// 添加一个带有自定义选项的任务
scheduler.AddTask("0 9 * * 1-5", golitecron.WrapJob("workday-task", func() error {
    fmt.Println("工作日上海时间上午9点运行的任务")
    return nil
}), 
    golitecron.WithLocation(shanghaiLoc),    // 使用上海时区
    golitecron.WithTimeout(5*time.Second),   // 5秒后超时
    golitecron.WithRetry(3)                  // 失败最多重试3次
)
```

### 从配置文件加载任务

#### 1. 创建 YAML 配置文件（cron.yaml）：

```yaml
tasks:
- id: "daily-task"
  cron_expr: "0 0 * * *"
  timeout: 30000        # 30秒（毫秒）
  retry: 2
  location: "Asia/Shanghai"
  func_name: "dailyJob"
- id: "hourly-task"
  cron_expr: "0 * * * *"
  timeout: 10000        # 10秒（毫秒）
  retry: 1
  location: "UTC"
  func_name: "hourlyJob"
```

#### 2. 加载并注册任务：

```go
// 注册作业函数
golitecron.RegisterJob("dailyJob", func() error {
    fmt.Println("执行每日任务")
    return nil
})

golitecron.RegisterJob("hourlyJob", func() error {
    fmt.Println("执行每小时任务")
    return nil
})

// 加载配置
config, err := golitecron.LoadFromYaml("cron.yaml")
if err != nil {
    fmt.Printf("加载配置错误: %v\n", err)
    return
}

// 创建调度器并加载任务
scheduler := golitecron.NewScheduler(golitecron.StorageTypeHeap)
err = scheduler.LoadTasksFromConfig(config)
if err != nil {
    fmt.Printf("加载任务错误: %v\n", err)
    return
}

scheduler.Start()
defer scheduler.Stop()

// 保持运行
select {}
```

### 管理任务

```go
// 获取所有任务
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("任务ID: %s, 下次运行时间: %s\n", task.ID, task.NextRunTime)
}

// 获取任务信息
taskInfo := scheduler.GetTaskInfo("daily-task")
fmt.Println(taskInfo)

// 移除任务
taskToRemove := &golitecron.Task{ID: "hourly-task"}
scheduler.RemoveTask(taskToRemove)
```

## 链式API

链式API提供了更直观的方式来定义调度任务，使用自然语言语法。

### 基本用法

```go
// 基本用法
scheduler.Every(10).Seconds().Do(job)
scheduler.Every(5).Minutes().Do(job) 
scheduler.Every().Day().At("10:30").Do(job)
scheduler.Every().Monday().Do(job)
scheduler.Every().Wednesday().At("14:15").Do(job)
scheduler.Every(2).Weeks().Do(job)

// 带选项配置
scheduler.Every().Day().At("09:00").
    WithTimeout(30*time.Second).
    WithRetry(3).
    WithLocation(shanghaiLoc).
    Do(job, "custom-task-id")
```

### 支持的时间单位

#### 基本时间单位
- `Second()` / `Seconds()` - 秒
- `Minute()` / `Minutes()` - 分钟  
- `Hour()` / `Hours()` - 小时
- `Day()` / `Days()` - 天
- `Week()` / `Weeks()` - 周
- `Month()` / `Months()` - 月

#### 星期几
- `Monday()` - 周一
- `Tuesday()` - 周二
- `Wednesday()` - 周三
- `Thursday()` - 周四
- `Friday()` - 周五
- `Saturday()` - 周六
- `Sunday()` - 周日

### 时间指定

使用 `At()` 方法指定具体执行时间：

```go
// 支持 HH:MM 格式
scheduler.Every().Day().At("10:30").Do(job)

// 支持 HH:MM:SS 格式（自动启用秒级精度）
scheduler.Every().Day().At("10:30:15").Do(job)
```

### 任务函数类型

链式API支持多种任务函数类型：

```go
// 1. 无返回值函数
scheduler.Every(10).Seconds().Do(func() {
    fmt.Println("简单任务")
})

// 2. 带错误返回值的函数
scheduler.Every().Hour().Do(func() error {
    fmt.Println("带错误处理的任务")
    return nil
})

// 3. Job接口实现
job := golitecron.WrapJob("my-job", func() error {
    return nil
})
scheduler.Every().Day().Do(job)
```

### 链式选项配置

```go
shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

scheduler.Every().Day().At("09:00").
    WithTimeout(30*time.Second).    // 超时时间
    WithRetry(3).                   // 重试次数
    WithLocation(shanghaiLoc).      // 时区
    WithSeconds().                  // 启用秒级精度
    WithYears().                    // 启用年份字段
    Do(job, "custom-task-id")       // 自定义任务ID
```

### 链式API示例

#### 基础示例

```go
package main

import (
    "fmt"
    "time"
    golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
    
    // 每10秒执行
    scheduler.Every(10).Seconds().Do(func() {
        fmt.Println("每10秒运行一次任务")
    })
    
    // 每天上午10:30执行
    scheduler.Every().Day().At("10:30").Do(func() error {
        fmt.Println("每日早晨任务")
        return nil
    })
    
    // 每周一执行
    scheduler.Every().Monday().Do(func() {
        fmt.Println("周一任务")
    })
    
    scheduler.Start()
    defer scheduler.Stop()
    
    // 保持程序运行
    select {}
}
```

#### 高级示例

```go
package main

import (
    "fmt"
    "time"
    golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
    
    // 使用时区和超时配置
    shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")
    
    err := scheduler.Every().Day().At("09:00:30").
        WithTimeout(30*time.Second).
        WithRetry(3).
        WithLocation(shanghaiLoc).
        Do(func() error {
            fmt.Println("上海早晨任务，带超时和重试")
            return nil
        }, "shanghai-morning")
    
    if err != nil {
        fmt.Printf("错误: %v\n", err)
        return
    }
    
    // 复杂的业务逻辑任务
    scheduler.Every(2).Hours().Do(func() error {
        // 执行数据同步
        fmt.Println("数据同步任务运行中...")
        time.Sleep(5 * time.Second) // 模拟耗时操作
        return nil
    }, "data-sync")
    
    scheduler.Start()
    defer scheduler.Stop()
    
    // 显示所有任务
    tasks := scheduler.GetTasks()
    fmt.Println("已调度的任务:")
    for _, task := range tasks {
        fmt.Printf("- %s: 下次运行于 %s\n", 
            task.ID, task.NextRunTime.Format("2006-01-02 15:04:05"))
    }
    
    select {}
}
```

## 配置

### Cron表达式语法

GoLiteCron支持标准cron表达式和可选扩展：

#### 标准格式（5个字段）
```
* * * * *
│ │ │ │ │
│ │ │ │ └───── 星期 (0-7, 周日为0或7)
│ │ │ └─────── 月份 (1-12)
│ │ └───────── 日期 (1-31)
│ └─────────── 小时 (0-23)
└───────────── 分钟 (0-59)
```

#### 扩展格式（6个字段，包含秒）
```
* * * * * *
│ │ │ │ │ │
│ │ │ │ │ └───── 星期 (0-7, 周日为0或7)
│ │ │ │ └─────── 月份 (1-12)
│ │ │ └───────── 日期 (1-31)
│ │ └─────────── 小时 (0-23)
│ └───────────── 分钟 (0-59)
└─────────────── 秒 (0-59)
```

#### 预定义宏
- `@yearly` 或 `@annually` - 每年1月1日午夜运行一次
- `@monthly` - 每月1日午夜运行一次
- `@weekly` - 每周日午夜运行一次
- `@daily` 或 `@midnight` - 每日午夜运行一次
- `@hourly` - 每小时开始时运行一次
- `@minutely` - 每分钟开始时运行一次

### 存储类型

GoLiteCron提供两种存储实现：

#### TimeWheel存储
- 高效处理高频任务和大量定时作业
- 使用多级时间轮算法
- 推荐用于生产环境中任务较多的情况

```go
scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
```

#### Heap存储
- 简单的优先级队列实现
- 按下次执行时间排序任务
- 适用于简单应用或开发环境

```go
scheduler := golitecron.NewScheduler(golitecron.StorageTypeHeap)
```

### 任务选项

#### WithLocation
设置任务执行的时区：

```go
loc, _ := time.LoadLocation("Asia/Shanghai")
scheduler.AddTask("0 9 * * *", job, golitecron.WithLocation(loc))
```

#### WithTimeout
设置任务执行的超时时间：

```go
scheduler.AddTask("0 */5 * * *", job, golitecron.WithTimeout(30*time.Second))
```

#### WithRetry
设置失败任务的重试次数：

```go
scheduler.AddTask("0 0 * * *", job, golitecron.WithRetry(3))
```

#### WithSeconds
在cron表达式中启用秒级精度：

```go
scheduler.AddTask("*/30 * * * * *", job, golitecron.WithSeconds())
```

#### WithYears
在cron表达式中启用年份字段：

```go
scheduler.AddTask("0 0 1 1 * 2024", job, golitecron.WithYears())
```

## 任务管理

### 作业注册

注册可在配置文件中按名称引用的作业：

```go
golitecron.RegisterJob("myJob", func() error {
    fmt.Println("执行已注册的作业")
    return nil
})
```

### 任务信息

获取任务的详细信息：

```go
// 获取所有任务
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("任务ID: %s\n", task.ID)
    fmt.Printf("Cron表达式: %s\n", task.CronExpr)
    fmt.Printf("下次运行: %s\n", task.NextRunTime)
    fmt.Printf("上次运行: %s\n", task.LastRunTime)
}

// 获取特定任务信息
taskInfo := scheduler.GetTaskInfo("task-id")
fmt.Println(taskInfo)
```

### 任务生命周期

```go
// 添加任务
err := scheduler.AddTask("0 */10 * * * *", job, golitecron.WithSeconds())

// 移除任务
taskToRemove := &golitecron.Task{ID: "task-id"}
scheduler.RemoveTask(taskToRemove)

// 启动调度器
scheduler.Start()

// 停止调度器（优雅关闭）
scheduler.Stop()
```

### 错误处理

GoLiteCron提供内置的错误处理和恢复机制：

```go
// 作业可以返回错误
scheduler.AddTask("0 */5 * * *", func() error {
    // 你的逻辑
    if someCondition {
        return fmt.Errorf("something went wrong")
    }
    return nil
}, golitecron.WithRetry(3)) // 最多重试3次
```

## 最佳实践

1. **选择合适的存储类型**：生产环境任务多时使用TimeWheel，简单场景使用Heap
2. **设置适当的超时时间**：防止任务无限期运行
3. **使用重试机制**：优雅处理瞬时故障
4. **为配置注册作业**：使配置文件更易读
5. **处理panic**：GoLiteCron会自动恢复panic，但作业函数中仍应处理错误
6. **使用描述性任务ID**：使调试和监控更容易
7. **考虑时区**：分布式应用中始终指定时区
8. **优雅关闭**：始终调用`Stop()`确保运行中的任务完成