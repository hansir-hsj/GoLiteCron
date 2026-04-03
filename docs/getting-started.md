# Getting Started with GoLiteCron

This guide covers all features of GoLiteCron with detailed examples.

## Table of Contents

- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Chain API](#chain-api)
- [Cron Expressions](#cron-expressions)
- [Configuration Options](#configuration-options)
- [Loading from Config Files](#loading-from-config-files)
- [Task Management](#task-management)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Installation

```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## Basic Usage

### Creating a Scheduler

```go
package main

import (
    "fmt"
    cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    // Default scheduler with Heap storage
    scheduler := cron.NewScheduler()
    
    // Or use TimeWheel for better performance with many tasks
    scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)
    
    // Add tasks...
    
    scheduler.Start()
    defer scheduler.Stop()
    
    select {} // keep running
}
```

### Adding Tasks with Cron Expressions

```go
// Every 5 minutes
scheduler.AddTask("*/5 * * * *", cron.WrapJob("task-1", func() error {
    fmt.Println("Running every 5 minutes")
    return nil
}))

// Every day at 10:30 AM
scheduler.AddTask("30 10 * * *", cron.WrapJob("task-2", func() error {
    fmt.Println("Running at 10:30 AM")
    return nil
}))

// Every Monday at 9:00 AM
scheduler.AddTask("0 9 * * 1", cron.WrapJob("task-3", func() error {
    fmt.Println("Running every Monday at 9 AM")
    return nil
}))
```

## Chain API

The Chain API provides a more readable way to define scheduled tasks.

### Time Intervals

```go
// Every N seconds (requires second-level precision)
scheduler.Every(10).Seconds().Do(func() {
    fmt.Println("Every 10 seconds")
})

// Every N minutes
scheduler.Every(5).Minutes().Do(func() {
    fmt.Println("Every 5 minutes")
})

// Every N hours
scheduler.Every(2).Hours().Do(func() {
    fmt.Println("Every 2 hours")
})
```

### Specific Times

```go
// Daily at specific time
scheduler.Every().Day().At("10:30").Do(job)

// With seconds precision (HH:MM:SS)
scheduler.Every().Day().At("10:30:15").Do(job)
```

### Weekdays

```go
scheduler.Every().Monday().Do(job)
scheduler.Every().Tuesday().At("09:00").Do(job)
scheduler.Every().Wednesday().At("14:30").Do(job)
scheduler.Every().Thursday().Do(job)
scheduler.Every().Friday().At("17:00").Do(job)
scheduler.Every().Saturday().Do(job)
scheduler.Every().Sunday().At("08:00").Do(job)
```

### Weekly and Monthly

```go
// Every week (Sunday midnight)
scheduler.Every().Week().Do(job)

// Every 2 weeks
scheduler.Every(2).Weeks().Do(job)

// Every month (1st day midnight)
scheduler.Every().Month().Do(job)
```

### Task Function Types

The Chain API supports multiple function signatures:

```go
// Simple function (no return value)
scheduler.Every().Hour().Do(func() {
    fmt.Println("Simple task")
})

// Function with error return
scheduler.Every().Hour().Do(func() error {
    if err := doSomething(); err != nil {
        return err
    }
    return nil
})

// Job interface
job := cron.WrapJob("my-job", func() error {
    return nil
})
scheduler.Every().Hour().Do(job)
```

### Chain Options

```go
loc, _ := time.LoadLocation("Asia/Shanghai")

err := scheduler.Every().Day().At("09:00:30").
    WithTimeout(30*time.Second).    // Task timeout
    WithRetry(3).                   // Retry 3 times on failure
    WithLocation(loc).              // Use Shanghai timezone
    Do(func() error {
        fmt.Println("Task with options")
        return nil
    }, "custom-task-id")            // Custom task ID

if err != nil {
    log.Printf("Failed to add task: %v", err)
}
```

## Cron Expressions

### Field Definitions

**5-field format (standard):**
```
* * * * *
│ │ │ │ │
│ │ │ │ └── Day of week (0-6, Sunday=0)
│ │ │ └──── Month (1-12)
│ │ └────── Day of month (1-31)
│ └──────── Hour (0-23)
└────────── Minute (0-59)
```

**6-field format (with seconds, requires `WithSeconds()`):**
```
* * * * * *
│ │ │ │ │ │
│ │ │ │ │ └── Day of week (0-6)
│ │ │ │ └──── Month (1-12)
│ │ │ └────── Day of month (1-31)
│ │ └──────── Hour (0-23)
│ └────────── Minute (0-59)
└──────────── Second (0-59)
```

**7-field format (with years, requires `WithYears()`):**
```
* * * * * * *
│ │ │ │ │ │ │
│ │ │ │ │ │ └── Year (1970-2099)
│ │ │ │ │ └──── Day of week (0-6)
│ │ │ │ └────── Month (1-12)
│ │ │ └──────── Day of month (1-31)
│ │ └────────── Hour (0-23)
│ └──────────── Minute (0-59)
└────────────── Second (0-59)
```

### Special Characters

| Character | Description | Example |
|-----------|-------------|---------|
| `*` | Any value | `* * * * *` - every minute |
| `,` | List of values | `1,15,30 * * * *` - at minute 1, 15, 30 |
| `-` | Range | `1-5 * * * *` - minutes 1 through 5 |
| `/` | Step values | `*/15 * * * *` - every 15 minutes |
| `L` | Last | `0 0 L * *` - last day of month |
| `W` | Nearest weekday | `0 0 15W * *` - nearest weekday to 15th |

### Examples

```go
// Every minute
"* * * * *"

// Every 5 minutes
"*/5 * * * *"

// At minute 0 and 30
"0,30 * * * *"

// Every hour at minute 0
"0 * * * *"

// Every day at midnight
"0 0 * * *"

// Every day at 9:00 AM
"0 9 * * *"

// Every Monday at 9:00 AM
"0 9 * * 1"

// Every weekday at 9:00 AM
"0 9 * * 1-5"

// First day of every month at midnight
"0 0 1 * *"

// Last day of every month at midnight
"0 0 L * *"

// Every second (with WithSeconds())
"* * * * * *"

// Every 30 seconds (with WithSeconds())
"*/30 * * * * *"
```

### Predefined Macros

```go
scheduler.AddTask("@yearly", job)    // 0 0 1 1 * - Jan 1 midnight
scheduler.AddTask("@monthly", job)   // 0 0 1 * * - 1st day midnight
scheduler.AddTask("@weekly", job)    // 0 0 * * 0 - Sunday midnight
scheduler.AddTask("@daily", job)     // 0 0 * * * - Every day midnight
scheduler.AddTask("@hourly", job)    // 0 * * * * - Every hour
scheduler.AddTask("@minutely", job)  // * * * * * - Every minute
```

## Configuration Options

### WithTimeout

Limits task execution time. Task is cancelled if it exceeds the timeout.

```go
scheduler.AddTask("*/5 * * * *", job, cron.WithTimeout(30*time.Second))
```

### WithRetry

Automatically retries failed tasks.

```go
scheduler.AddTask("*/5 * * * *", job, cron.WithRetry(3)) // Retry up to 3 times
```

### WithLocation

Sets the timezone for task scheduling.

```go
loc, _ := time.LoadLocation("America/New_York")
scheduler.AddTask("0 9 * * *", job, cron.WithLocation(loc))
```

### WithSeconds

Enables 6-field cron expressions with second precision.

```go
scheduler.AddTask("*/30 * * * * *", job, cron.WithSeconds())
```

### WithYears

Enables 7-field cron expressions with year specification.

```go
scheduler.AddTask("0 0 1 1 * 2025", job, cron.WithSeconds(), cron.WithYears())
```

### Combining Options

```go
loc, _ := time.LoadLocation("Asia/Tokyo")
scheduler.AddTask("*/30 * * * * *", job,
    cron.WithSeconds(),
    cron.WithLocation(loc),
    cron.WithTimeout(10*time.Second),
    cron.WithRetry(2),
)
```

## Loading from Config Files

### YAML Configuration

**config.yaml:**
```yaml
tasks:
  - id: "daily-backup"
    cron_expr: "0 2 * * *"
    func_name: "backupDatabase"
    timeout: "5m"
    retry: 3
    location: "UTC"

  - id: "hourly-sync"
    cron_expr: "0 * * * *"
    func_name: "syncData"
    timeout: "1m"
    retry: 1

  - id: "realtime-check"
    cron_expr: "*/10 * * * * *"
    func_name: "healthCheck"
    enable_seconds: true
    timeout: "5s"
```

### JSON Configuration

**config.json:**
```json
{
  "tasks": [
    {
      "id": "daily-backup",
      "cron_expr": "0 2 * * *",
      "func_name": "backupDatabase",
      "timeout": "5m",
      "retry": 3,
      "location": "UTC"
    }
  ]
}
```

### Loading and Using Configuration

```go
package main

import (
    "fmt"
    "log"
    cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    // Register job functions first
    cron.RegisterJob("backupDatabase", func() error {
        fmt.Println("Backing up database...")
        return nil
    })

    cron.RegisterJob("syncData", func() error {
        fmt.Println("Syncing data...")
        return nil
    })

    cron.RegisterJob("healthCheck", func() error {
        fmt.Println("Health check...")
        return nil
    })

    // Load configuration
    config, err := cron.LoadFromYaml("config.yaml")
    // Or: config, err := cron.LoadFromJson("config.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create scheduler and load tasks
    scheduler := cron.NewScheduler()
    if err := scheduler.LoadTasksFromConfig(config); err != nil {
        log.Fatalf("Failed to load tasks: %v", err)
    }

    scheduler.Start()
    defer scheduler.Stop()

    select {}
}
```

## Task Management

### Listing Tasks

```go
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("Task: %s\n", task.ID)
    fmt.Printf("  Next Run: %s\n", task.NextRunTime.Format(time.RFC3339))
    fmt.Printf("  Last Run: %s\n", task.PreRunTime.Format(time.RFC3339))
}
```

### Getting Task Info

```go
info := scheduler.GetTaskInfo("my-task-id")
fmt.Println(info)
// Output: Task ID: my-task-id, Pre Run Time: 2024-01-01T10:00:00Z, Next Run Time: 2024-01-01T11:00:00Z
```

### Removing Tasks

```go
task := &cron.Task{ID: "task-to-remove"}
removed := scheduler.RemoveTask(task)
if removed {
    fmt.Println("Task removed successfully")
} else {
    fmt.Println("Task not found")
}
```

### Starting and Stopping

```go
// Start the scheduler
scheduler.Start()

// Stop gracefully (waits for running tasks to complete)
scheduler.Stop()
```

## Best Practices

1. **Choose the right storage backend**
   - Use `StorageTypeHeap` (default) for simple applications with few tasks
   - Use `StorageTypeTimeWheel` for high-performance scenarios with many tasks

2. **Set appropriate timeouts**
   - Always set timeouts to prevent tasks from running indefinitely
   - Consider the expected execution time and add some buffer

3. **Use retry wisely**
   - Set retries for tasks that may fail due to transient errors
   - Don't retry tasks that will consistently fail

4. **Handle errors in tasks**
   - Return errors from your task functions
   - Errors are logged to stderr by default

5. **Use descriptive task IDs**
   - Makes debugging and monitoring easier
   - Use meaningful names like "daily-backup" or "hourly-sync"

6. **Consider timezones**
   - Always specify timezone for production applications
   - Use UTC for consistency across servers

7. **Graceful shutdown**
   - Always call `scheduler.Stop()` before exiting
   - Running tasks will be allowed to complete

## Examples

Complete runnable examples are available in the [examples/](../examples/) directory:

| Example | Description |
|---------|-------------|
| [basic](../examples/basic) | Minimal setup, 5-minute quickstart |
| [fluent-api](../examples/fluent-api) | Chain-style `Every().Day().At()` |
| [cron-expr](../examples/cron-expr) | 5/6/7-field cron, L/W, macros |
| [config-file](../examples/config-file) | YAML/JSON config loading |
| [error-handling](../examples/error-handling) | Timeout, retry, context |
| [graceful](../examples/graceful) | Signal handling, graceful shutdown |

Run any example:

```bash
go run ./examples/basic
```