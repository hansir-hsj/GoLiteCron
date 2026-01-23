# GoLiteCron 入门指南

本指南涵盖 GoLiteCron 的所有功能，并提供详细示例。

## 目录

- [安装](#安装)
- [基本用法](#基本用法)
- [链式 API](#链式-api)
- [Cron 表达式](#cron-表达式)
- [配置选项](#配置选项)
- [从配置文件加载](#从配置文件加载)
- [任务管理](#任务管理)
- [API 参考](#api-参考)
- [最佳实践](#最佳实践)

## 安装

```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## 基本用法

### 创建调度器

```go
package main

import (
    "fmt"
    cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    // 默认使用 Heap 存储
    scheduler := cron.NewScheduler()
    
    // 或使用 TimeWheel 以获得更好的大量任务性能
    scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)
    
    // 添加任务...
    
    scheduler.Start()
    defer scheduler.Stop()
    
    select {} // 保持运行
}
```

### 使用 Cron 表达式添加任务

```go
// 每5分钟
scheduler.AddTask("*/5 * * * *", cron.WrapJob("task-1", func() error {
    fmt.Println("每5分钟运行一次")
    return nil
}))

// 每天上午 10:30
scheduler.AddTask("30 10 * * *", cron.WrapJob("task-2", func() error {
    fmt.Println("上午 10:30 运行")
    return nil
}))

// 每周一上午 9:00
scheduler.AddTask("0 9 * * 1", cron.WrapJob("task-3", func() error {
    fmt.Println("每周一上午 9:00 运行")
    return nil
}))
```

## 链式 API

链式 API 提供了一种更易读的方式来定义定时任务。

### 时间间隔

```go
// 每 N 秒 (需要秒级精度)
scheduler.Every(10).Seconds().Do(func() {
    fmt.Println("每10秒")
})

// 每 N 分钟
scheduler.Every(5).Minutes().Do(func() {
    fmt.Println("每5分钟")
})

// 每 N 小时
scheduler.Every(2).Hours().Do(func() {
    fmt.Println("每2小时")
})
```

### 指定时间

```go
// 每天指定时间
scheduler.Every().Day().At("10:30").Do(job)

// 带秒级精度 (HH:MM:SS)
scheduler.Every().Day().At("10:30:15").Do(job)
```

### 星期几

```go
scheduler.Every().Monday().Do(job)
scheduler.Every().Tuesday().At("09:00").Do(job)
scheduler.Every().Wednesday().At("14:30").Do(job)
scheduler.Every().Thursday().Do(job)
scheduler.Every().Friday().At("17:00").Do(job)
scheduler.Every().Saturday().Do(job)
scheduler.Every().Sunday().At("08:00").Do(job)
```

### 每周和每月

```go
// 每周 (周日午夜)
scheduler.Every().Week().Do(job)

// 每2周
scheduler.Every(2).Weeks().Do(job)

// 每月 (1号午夜)
scheduler.Every().Month().Do(job)
```

### 任务函数类型

链式 API 支持多种函数签名：

```go
// 简单函数 (无返回值)
scheduler.Every().Hour().Do(func() {
    fmt.Println("简单任务")
})

// 带错误返回的函数
scheduler.Every().Hour().Do(func() error {
    if err := doSomething(); err != nil {
        return err
    }
    return nil
})

// Job 接口
job := cron.WrapJob("my-job", func() error {
    return nil
})
scheduler.Every().Hour().Do(job)
```

### 链式选项

```go
loc, _ := time.LoadLocation("Asia/Shanghai")

err := scheduler.Every().Day().At("09:00:30").
    WithTimeout(30*time.Second).    // 任务超时
    WithRetry(3).                   // 失败重试3次
    WithLocation(loc).              // 使用上海时区
    Do(func() error {
        fmt.Println("带选项的任务")
        return nil
    }, "custom-task-id")            // 自定义任务ID

if err != nil {
    log.Printf("添加任务失败: %v", err)
}
```

## Cron 表达式

### 字段定义

**5字段格式 (标准):**
```
* * * * *
│ │ │ │ │
│ │ │ │ └── 星期 (0-6, 周日=0)
│ │ │ └─── 月份 (1-12)
│ │ └───── 日 (1-31)
│ └─────── 小时 (0-23)
└───────── 分钟 (0-59)
```

**6字段格式 (含秒, 需要 `WithSeconds()`):**
```
* * * * * *
│ │ │ │ │ │
│ │ │ │ │ └── 星期 (0-6)
│ │ │ │ └─── 月份 (1-12)
│ │ │ └───── 日 (1-31)
│ │ └─────── 小时 (0-23)
│ └───────── 分钟 (0-59)
└─────────── 秒 (0-59)
```

**7字段格式 (含年, 需要 `WithYears()`):**
```
* * * * * * *
│ │ │ │ │ │ │
│ │ │ │ │ │ └── 年份 (1970-2099)
│ │ │ │ │ └── 星期 (0-6)
│ │ │ │ └─── 月份 (1-12)
│ │ │ └───── 日 (1-31)
│ │ └─────── 小时 (0-23)
│ └───────── 分钟 (0-59)
└─────────── 秒 (0-59)
```

### 特殊字符

| 字符 | 描述 | 示例 |
|------|------|------|
| `*` | 任意值 | `* * * * *` - 每分钟 |
| `,` | 值列表 | `1,15,30 * * * *` - 第1, 15, 30分钟 |
| `-` | 范围 | `1-5 * * * *` - 第1到5分钟 |
| `/` | 步长 | `*/15 * * * *` - 每15分钟 |
| `L` | 最后 | `0 0 L * *` - 每月最后一天 |
| `W` | 最近工作日 | `0 0 15W * *` - 最接近15号的工作日 |

### 示例

```go
// 每分钟
"* * * * *"

// 每5分钟
"*/5 * * * *"

// 在第0和30分钟
"0,30 * * * *"

// 每天上午9点
"0 9 * * *"

// 工作日上午9点
"0 9 * * 1-5"

// 每月第一天午夜
"0 0 1 * *"

// 每月最后一天午夜
"0 0 L * *"

// 每秒 (需 WithSeconds())
"* * * * * *"
```

### 预定义宏

```go
scheduler.AddTask("@yearly", job)    // 0 0 1 1 * - 每年1月1日
scheduler.AddTask("@monthly", job)   // 0 0 1 * * - 每月1号
scheduler.AddTask("@weekly", job)    // 0 0 * * 0 - 每周日
scheduler.AddTask("@daily", job)     // 0 0 * * * - 每天午夜
scheduler.AddTask("@hourly", job)    // 0 * * * * - 每小时
scheduler.AddTask("@minutely", job)  // * * * * * - 每分钟
```

## 配置选项

### WithTimeout

限制任务执行时间。如果超时，任务将被取消。

```go
scheduler.AddTask("*/5 * * * *", job, cron.WithTimeout(30*time.Second))
```

### WithRetry

自动重试失败的任务。

```go
scheduler.AddTask("*/5 * * * *", job, cron.WithRetry(3)) // 重试3次
```

### WithLocation

设置任务调度的时区。

```go
loc, _ := time.LoadLocation("Asia/Shanghai")
scheduler.AddTask("0 9 * * *", job, cron.WithLocation(loc))
```

### WithSeconds

启用含秒的6字段 cron 表达式。

```go
scheduler.AddTask("*/30 * * * * *", job, cron.WithSeconds())
```

### WithYears

启用含年份的7字段 cron 表达式。

```go
scheduler.AddTask("0 0 1 1 * 2025", job, cron.WithSeconds(), cron.WithYears())
```

## 从配置文件加载

### YAML 配置

**config.yaml:**
```yaml
tasks:
  - id: "daily-backup"
    cron_expr: "0 2 * * *"
    func_name: "backupDatabase"
    timeout: 300000      # 5分钟 (毫秒)
    retry: 3
    location: "UTC"

  - id: "hourly-sync"
    cron_expr: "0 * * * *"
    func_name: "syncData"
    timeout: 60000       # 1分钟
    retry: 1

  - id: "realtime-check"
    cron_expr: "*/10 * * * * *"
    func_name: "healthCheck"
    enable_seconds: true
    timeout: 5000
```

### 加载并使用配置

```go
package main

import (
    "fmt"
    "log"
    cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    // 首先注册任务函数
    cron.RegisterJob("backupDatabase", func() error {
        fmt.Println("备份数据库...")
        return nil
    })

    cron.RegisterJob("syncData", func() error {
        fmt.Println("同步数据...")
        return nil
    })

    // 加载配置
    config, err := cron.LoadFromYaml("config.yaml")
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }

    // 创建调度器并加载任务
    scheduler := cron.NewScheduler()
    if err := scheduler.LoadTasksFromConfig(config); err != nil {
        log.Fatalf("加载任务失败: %v", err)
    }

    scheduler.Start()
    defer scheduler.Stop()

    select {}
}
```

## 任务管理

### 列出任务

```go
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("任务: %s\n", task.ID)
    fmt.Printf("  下次运行: %s\n", task.NextRunTime.Format(time.RFC3339))
    fmt.Printf("  上次运行: %s\n", task.PreRunTime.Format(time.RFC3339))
}
```

### 获取任务信息

```go
info := scheduler.GetTaskInfo("my-task-id")
fmt.Println(info)
```

### 移除任务

```go
task := &cron.Task{ID: "task-to-remove"}
removed := scheduler.RemoveTask(task)
if removed {
    fmt.Println("任务已移除")
} else {
    fmt.Println("任务未找到")
}
```

## 最佳实践

1. **选择合适的存储后端**
   - 简单应用使用 `StorageTypeHeap` (默认)
   - 高并发、大量任务使用 `StorageTypeTimeWheel`

2. **设置适当的超时**
   - 始终设置超时以防止任务无限期运行
   - 考虑预期执行时间并留出缓冲

3. **明智地使用重试**
   - 对可能因瞬态错误失败的任务设置重试
   - 不要对必然失败的任务进行重试

4. **处理任务中的错误**
   - 在任务函数中返回错误
   - 默认情况下错误会输出到标准错误流

5. **使用描述性任务ID**
   - 便于调试和监控
   - 使用有意义的名称，如 "daily-backup" 或 "hourly-sync"

6. **考虑时区**
   - 生产环境中始终明确指定时区
   - 服务器间一致性建议使用 UTC

7. **优雅关闭**
   - 退出前始终调用 `scheduler.Stop()`
   - 这将等待正在运行的任务完成
