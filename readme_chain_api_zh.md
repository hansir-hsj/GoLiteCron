# GoLiteCron 链式API文档

## 概述

为了提供更友好的用户体验，GoLiteCron新增了类似Python schedule库的链式API，让你可以用更自然的语言来定义调度任务。

## 新增功能

### 链式API语法

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
    fmt.Println("Simple task")
})

// 2. 带错误返回值的函数
scheduler.Every().Hour().Do(func() error {
    fmt.Println("Task with error handling")
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

## API参考

### ScheduleBuilder 方法

#### 时间间隔方法
- `Every(intervals ...int) *ScheduleBuilder` - 开始构建，可选指定间隔数

#### 时间单位方法
- `Second() / Seconds() *ScheduleBuilder` - 秒单位
- `Minute() / Minutes() *ScheduleBuilder` - 分钟单位
- `Hour() / Hours() *ScheduleBuilder` - 小时单位
- `Day() / Days() *ScheduleBuilder` - 天单位
- `Week() / Weeks() *ScheduleBuilder` - 周单位
- `Month() / Months() *ScheduleBuilder` - 月单位

#### 星期几方法
- `Monday() / Tuesday() / ... / Sunday() *ScheduleBuilder` - 指定星期几

#### 时间指定方法
- `At(timeStr string) *ScheduleBuilder` - 指定执行时间（HH:MM 或 HH:MM:SS）

#### 选项配置方法
- `WithTimeout(timeout time.Duration) *ScheduleBuilder` - 设置超时时间
- `WithRetry(retry int) *ScheduleBuilder` - 设置重试次数
- `WithLocation(loc *time.Location) *ScheduleBuilder` - 设置时区
- `WithSeconds() *ScheduleBuilder` - 启用秒级精度
- `WithYears() *ScheduleBuilder` - 启用年份字段

#### 执行方法
- `Do(job interface{}, taskID ...string) error` - 添加任务到调度器

## 使用示例

### 基础示例

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
        fmt.Println("Task runs every 10 seconds")
    })
    
    // 每天上午10:30执行
    scheduler.Every().Day().At("10:30").Do(func() error {
        fmt.Println("Daily morning task")
        return nil
    })
    
    // 每周一执行
    scheduler.Every().Monday().Do(func() {
        fmt.Println("Monday task")
    })
    
    scheduler.Start()
    defer scheduler.Stop()
    
    // 保持程序运行
    select {}
}
```

### 高级示例

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
            fmt.Println("Shanghai morning task with timeout and retry")
            return nil
        }, "shanghai-morning")
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    // 复杂的业务逻辑任务
    scheduler.Every(2).Hours().Do(func() error {
        // 执行数据同步
        fmt.Println("Data sync task running...")
        time.Sleep(5 * time.Second) // 模拟耗时操作
        return nil
    }, "data-sync")
    
    scheduler.Start()
    defer scheduler.Stop()
    
    // 显示所有任务
    tasks := scheduler.GetTasks()
    fmt.Println("Scheduled tasks:")
    for _, task := range tasks {
        fmt.Printf("- %s: next run at %s\n", 
            task.ID, task.NextRunTime.Format("2006-01-02 15:04:05"))
    }
    
    select {}
}
```

## 与原有API的兼容性

链式API是在原有API基础上的增强，完全兼容现有代码：

```go
// 原有API仍然可用
scheduler.AddTask("0 */30 * * *", golitecron.WrapJob("legacy-task", func() error {
    return nil
}))

// 新的链式API
scheduler.Every(30).Minutes().Do(func() error {
    return nil
}, "chain-task")
```

## 注意事项

1. **时区处理**：使用 `WithLocation()` 指定时区，默认使用系统本地时区
2. **秒级精度**：使用 `At("HH:MM:SS")` 格式或 `WithSeconds()` 启用秒级调度
3. **任务ID**：可以自定义任务ID，否则会自动生成描述性ID
4. **错误处理**：所有链式调用最终在 `Do()` 方法中进行错误检查
5. **性能**：链式API最终转换为标准cron表达式，性能与原有API相同

## 升级指南

从原有API迁移到链式API非常简单：

```go
// 原有写法
scheduler.AddTask("*/10 * * * * *", 
    golitecron.WrapJob("task", func() error { return nil }),
    golitecron.WithSeconds())

// 新的链式写法
scheduler.Every(10).Seconds().Do(func() error { return nil }, "task")
```

链式API让代码更加直观易读，推荐在新项目中使用。