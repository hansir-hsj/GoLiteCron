# GoLiteCron

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](../LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-88%25-brightgreen.svg)](.)

轻量级、高性能的 Go 定时任务调度器。

[English](../README.md)

## 特性

| 特性 | 描述 |
|------|------|
| 🕐 Cron 表达式 | 标准5字段、6字段（含秒）、7字段（含年） |
| 🔗 链式 API | 流式接口：`scheduler.Every(10).Seconds().Do(job)` |
| ⏱️ 超时与重试 | 内置超时控制和自动重试 |
| 🌍 时区支持 | 完整的时区支持 |
| 📦 存储后端 | TimeWheel（高性能）或 Heap（简单） |
| 📄 配置文件 | 从 YAML/JSON 加载任务 |
| 🛡️ Panic 恢复 | 自动从崩溃任务中恢复 |

## 安装

```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## 快速开始

```go
package main

import (
    "fmt"
    cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    scheduler := cron.NewScheduler()

    // 链式 API（推荐）
    scheduler.Every(10).Seconds().Do(func() {
        fmt.Println("每10秒运行一次")
    })

    // Cron 表达式
    scheduler.AddTask("*/5 * * * *", cron.WrapJob("five-min", func() error {
        fmt.Println("每5分钟运行一次")
        return nil
    }))

    scheduler.Start()
    defer scheduler.Stop()
    select {} // 保持运行
}
```

## Cron 表达式

### 标准格式（5字段）
```
┌───────────── 分钟 (0-59)
│ ┌───────────── 小时 (0-23)
│ │ ┌───────────── 日 (1-31)
│ │ │ ┌───────────── 月 (1-12)
│ │ │ │ ┌───────────── 星期 (0-6, 周日=0)
* * * * *
```

### 扩展格式（6字段，需要 `WithSeconds()`）
```
┌───────────── 秒 (0-59)
│ ┌───────────── 分钟 (0-59)
│ │ ┌───────────── 小时 (0-23)
│ │ │ ┌───────────── 日 (1-31)
│ │ │ │ ┌───────────── 月 (1-12)
│ │ │ │ │ ┌───────────── 星期 (0-6)
* * * * * *
```

### 特殊字符

| 字符 | 描述 | 示例 |
|------|------|------|
| `*` | 任意值 | `* * * * *` 每分钟 |
| `,` | 列表 | `1,15 * * * *` 第1和15分钟 |
| `-` | 范围 | `1-5 * * * *` 第1-5分钟 |
| `/` | 步长 | `*/10 * * * *` 每10分钟 |
| `L` | 最后 | `0 0 L * *` 每月最后一天 |
| `W` | 工作日 | `0 0 15W * *` 最接近15号的工作日 |

### 预定义宏

| 宏 | 等价表达式 | 描述 |
|----|-----------|------|
| `@yearly` | `0 0 1 1 *` | 每年一次（1月1日） |
| `@monthly` | `0 0 1 * *` | 每月一次（1号） |
| `@weekly` | `0 0 * * 0` | 每周一次（周日） |
| `@daily` | `0 0 * * *` | 每天一次（午夜） |
| `@hourly` | `0 * * * *` | 每小时一次 |
| `@minutely` | `* * * * *` | 每分钟一次 |

## 链式 API

```go
// 时间间隔
scheduler.Every(30).Seconds().Do(job)
scheduler.Every(5).Minutes().Do(job)
scheduler.Every(2).Hours().Do(job)

// 指定时间
scheduler.Every().Day().At("10:30").Do(job)
scheduler.Every().Monday().At("09:00").Do(job)

// 带选项
loc, _ := time.LoadLocation("Asia/Shanghai")
scheduler.Every().Day().At("09:00").
    WithTimeout(30*time.Second).
    WithRetry(3).
    WithLocation(loc).
    Do(job, "custom-task-id")
```

## 配置选项

```go
cron.WithTimeout(30 * time.Second)  // 任务超时
cron.WithRetry(3)                   // 失败重试
cron.WithLocation(loc)              // 时区
cron.WithSeconds()                  // 启用6字段cron
cron.WithYears()                    // 启用7字段cron
```

## 存储后端

```go
// Heap（默认）- 简单，适合少量任务
scheduler := cron.NewScheduler()

// TimeWheel - 高效，适合大量任务
scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)
```

## 从配置文件加载

**config.yaml:**
```yaml
tasks:
  - id: "backup"
    cron_expr: "0 2 * * *"
    func_name: "backupJob"
    timeout: 60000
    retry: 2
```

**main.go:**
```go
cron.RegisterJob("backupJob", func() error {
    return doBackup()
})

config, _ := cron.LoadFromYaml("config.yaml")
scheduler := cron.NewScheduler()
scheduler.LoadTasksFromConfig(config)
scheduler.Start()
```

## 任务管理

```go
// 列出任务
for _, task := range scheduler.GetTasks() {
    fmt.Printf("%s -> %s\n", task.ID, task.NextRunTime)
}

// 移除任务
scheduler.RemoveTask(&cron.Task{ID: "task-id"})

// 优雅关闭
scheduler.Stop()
```

## 详细文档

- [入门指南](getting-started.zh.md) - 详细使用示例
- [API 参考](api-reference.zh.md) - 类型与函数说明

## 许可证

MIT License - 详见 [LICENSE](../LICENSE)