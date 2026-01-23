# GoLiteCron

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-88%25-brightgreen.svg)](.)

A lightweight, high-performance cron job scheduler for Go.

[中文文档](docs/README.zh.md)

## Features

| Feature | Description |
|---------|-------------|
| 🕐 Cron Expressions | Standard 5-field, 6-field (with seconds), 7-field (with years) |
| 🔗 Chain API | Fluent API: `scheduler.Every(10).Seconds().Do(job)` |
| ⏱️ Timeout & Retry | Built-in timeout control and automatic retry |
| 🌍 Time Zones | Full timezone support for task execution |
| 📦 Storage Backends | TimeWheel (high performance) or Heap (simple) |
| 📄 Config Files | Load tasks from YAML/JSON configuration |
| 🛡️ Panic Recovery | Automatic recovery from panicked tasks |

## Installation

```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## Quick Start

```go
package main

import (
    "fmt"
    cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    scheduler := cron.NewScheduler()

    // Chain API (recommended)
    scheduler.Every(10).Seconds().Do(func() {
        fmt.Println("runs every 10 seconds")
    })

    // Cron expression
    scheduler.AddTask("*/5 * * * *", cron.WrapJob("five-min", func() error {
        fmt.Println("runs every 5 minutes")
        return nil
    }))

    scheduler.Start()
    defer scheduler.Stop()
    select {} // keep running
}
```

## Cron Expression

### Standard Format (5 fields)
```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sunday=0)
* * * * *
```

### Extended Format (6 fields, requires `WithSeconds()`)
```
┌───────────── second (0-59)
│ ┌───────────── minute (0-59)
│ │ ┌───────────── hour (0-23)
│ │ │ ┌───────────── day of month (1-31)
│ │ │ │ ┌───────────── month (1-12)
│ │ │ │ │ ┌───────────── day of week (0-6)
* * * * * *
```

### Special Characters

| Char | Description | Example |
|------|-------------|---------|
| `*` | Any value | `* * * * *` every minute |
| `,` | List | `1,15 * * * *` minute 1 and 15 |
| `-` | Range | `1-5 * * * *` minutes 1-5 |
| `/` | Step | `*/10 * * * *` every 10 minutes |
| `L` | Last | `0 0 L * *` last day of month |
| `W` | Weekday | `0 0 15W * *` nearest weekday to 15th |

### Predefined Macros

| Macro | Equivalent | Description |
|-------|------------|-------------|
| `@yearly` | `0 0 1 1 *` | Once a year (Jan 1) |
| `@monthly` | `0 0 1 * *` | Once a month (1st) |
| `@weekly` | `0 0 * * 0` | Once a week (Sunday) |
| `@daily` | `0 0 * * *` | Once a day (midnight) |
| `@hourly` | `0 * * * *` | Once an hour |
| `@minutely` | `* * * * *` | Once a minute |

## Chain API

```go
// Time intervals
scheduler.Every(30).Seconds().Do(job)
scheduler.Every(5).Minutes().Do(job)
scheduler.Every(2).Hours().Do(job)

// Specific time
scheduler.Every().Day().At("10:30").Do(job)
scheduler.Every().Monday().At("09:00").Do(job)

// With options
loc, _ := time.LoadLocation("Asia/Shanghai")
scheduler.Every().Day().At("09:00").
    WithTimeout(30*time.Second).
    WithRetry(3).
    WithLocation(loc).
    Do(job, "custom-task-id")
```

## Options

```go
cron.WithTimeout(30 * time.Second)  // Task timeout
cron.WithRetry(3)                   // Retry on failure
cron.WithLocation(loc)              // Timezone
cron.WithSeconds()                  // Enable 6-field cron
cron.WithYears()                    // Enable 7-field cron
```

## Storage Backends

```go
// Heap (default) - simple, good for fewer tasks
scheduler := cron.NewScheduler()

// TimeWheel - efficient for many tasks
scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)
```

## Load from Config

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

## Task Management

```go
// List tasks
for _, task := range scheduler.GetTasks() {
    fmt.Printf("%s -> %s\n", task.ID, task.NextRunTime)
}

// Remove task
scheduler.RemoveTask(&cron.Task{ID: "task-id"})

// Graceful shutdown
scheduler.Stop()
```

## Documentation

- [Getting Started](docs/getting-started.md) - Detailed guide with examples
- [中文文档](docs/README.zh.md) - Chinese documentation

## License

MIT License - see [LICENSE](LICENSE)
