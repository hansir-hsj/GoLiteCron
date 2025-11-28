# GoLiteCron Usage Guide

This document contains comprehensive usage examples and API documentation for GoLiteCron.

## Table of Contents
- [Quick Start](#quick-start)
- [Advanced Usage](#advanced-usage)
- [Chain API](#chain-api)
- [Configuration](#configuration)
- [Task Management](#task-management)
- [API Reference](#api-reference)
- [Best Practices](#best-practices)

## Quick Start

### Basic Example

```go
package main

import (
	"fmt"
	"time"
	golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	// Create a new scheduler with TimeWheel storage
	scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
	
	// Add a task that runs every minute
	err := scheduler.AddTask("@minutely", golitecron.WrapJob("minute-task", func() error {
		fmt.Printf("Minute task executed at %s\n", time.Now().Format(time.RFC3339))
		return nil
	}))
	
	if err != nil {
		fmt.Printf("Failed to add task: %v\n", err)
		return
	}
	
	// Start the scheduler
	scheduler.Start()
	defer scheduler.Stop()
	
	// Keep the program running
	select {}
}
```

## Advanced Usage

### Using Custom Cron Expressions

```go
// Add a task that runs every 30 minutes
scheduler.AddTask("*/30 * * * *", golitecron.WrapJob("30min-task", func() error {
    fmt.Println("Task running every 30 minutes")
    return nil
}))

// Add a task that runs every 2 seconds (requires WithSeconds option)
scheduler.AddTask("0/2 * * * * *", golitecron.WrapJob("2sec-task", func() error {
    fmt.Println("Task running every 2 seconds")
    return nil
}), golitecron.WithSeconds())
```

### Configuring Task Options

```go
// Create a location for Shanghai time
shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

// Add a task with custom options
scheduler.AddTask("0 9 * * 1-5", golitecron.WrapJob("workday-task", func() error {
    fmt.Println("Task running at 9:00 AM on workdays in Shanghai")
    return nil
}), 
    golitecron.WithLocation(shanghaiLoc),    // Use Shanghai time zone
    golitecron.WithTimeout(5*time.Second),   // Timeout after 5 seconds
    golitecron.WithRetry(3)                  // Retry up to 3 times on failure
)
```

### Loading Tasks from Configuration Files

#### 1. Create a YAML configuration file (cron.yaml):

```yaml
tasks:
- id: "daily-task"
  cron_expr: "0 0 * * *"
  timeout: 30000        # 30 seconds in milliseconds
  retry: 2
  location: "Asia/Shanghai"
  func_name: "dailyJob"
- id: "hourly-task"
  cron_expr: "0 * * * *"
  timeout: 10000        # 10 seconds in milliseconds
  retry: 1
  location: "UTC"
  func_name: "hourlyJob"
```

#### 2. Load and register tasks:

```go
// Register job functions
golitecron.RegisterJob("dailyJob", func() error {
    fmt.Println("Executing daily job")
    return nil
})

golitecron.RegisterJob("hourlyJob", func() error {
    fmt.Println("Executing hourly job")
    return nil
})

// Load configuration
config, err := golitecron.LoadFromYaml("cron.yaml")
if err != nil {
    fmt.Printf("Error loading config: %v\n", err)
    return
}

// Create scheduler and load tasks
scheduler := golitecron.NewScheduler(golitecron.StorageTypeHeap)
err = scheduler.LoadTasksFromConfig(config)
if err != nil {
    fmt.Printf("Error loading tasks: %v\n", err)
    return
}

scheduler.Start()
defer scheduler.Stop()

// Keep running
select {}
```

### Managing Tasks

```go
// Get all tasks
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("Task ID: %s, Next Run: %s\n", task.ID, task.NextRunTime)
}

// Get task information
taskInfo := scheduler.GetTaskInfo("daily-task")
fmt.Println(taskInfo)

// Remove a task
taskToRemove := &golitecron.Task{ID: "hourly-task"}
scheduler.RemoveTask(taskToRemove)
```

## Chain API

The chain API provides a more intuitive way to define scheduled tasks using natural language syntax.

### Basic Usage

```go
// Basic usage
scheduler.Every(10).Seconds().Do(job)
scheduler.Every(5).Minutes().Do(job) 
scheduler.Every().Day().At("10:30").Do(job)
scheduler.Every().Monday().Do(job)
scheduler.Every().Wednesday().At("14:15").Do(job)
scheduler.Every(2).Weeks().Do(job)

// With options configuration
scheduler.Every().Day().At("09:00").
    WithTimeout(30*time.Second).
    WithRetry(3).
    WithLocation(shanghaiLoc).
    Do(job, "custom-task-id")
```

### Supported Time Units

#### Basic Time Units
- `Second()` / `Seconds()` - seconds
- `Minute()` / `Minutes()` - minutes  
- `Hour()` / `Hours()` - hours
- `Day()` / `Days()` - days
- `Week()` / `Weeks()` - weeks
- `Month()` / `Months()` - months

#### Days of the Week
- `Monday()` - Monday
- `Tuesday()` - Tuesday
- `Wednesday()` - Wednesday
- `Thursday()` - Thursday
- `Friday()` - Friday
- `Saturday()` - Saturday
- `Sunday()` - Sunday

### Time Specification

Use the `At()` method to specify exact execution time:

```go
// Supports HH:MM format
scheduler.Every().Day().At("10:30").Do(job)

// Supports HH:MM:SS format (automatically enables second-level precision)
scheduler.Every().Day().At("10:30:15").Do(job)
```

### Task Function Types

The chain API supports multiple task function types:

```go
// 1. Function without return value
scheduler.Every(10).Seconds().Do(func() {
    fmt.Println("Simple task")
})

// 2. Function with error return value
scheduler.Every().Hour().Do(func() error {
    fmt.Println("Task with error handling")
    return nil
})

// 3. Job interface implementation
job := golitecron.WrapJob("my-job", func() error {
    return nil
})
scheduler.Every().Day().Do(job)
```

### Chain Options Configuration

```go
shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

scheduler.Every().Day().At("09:00").
    WithTimeout(30*time.Second).    // Timeout duration
    WithRetry(3).                   // Retry count
    WithLocation(shanghaiLoc).      // Time zone
    WithSeconds().                  // Enable second-level precision
    WithYears().                    // Enable year field
    Do(job, "custom-task-id")       // Custom task ID
```

### Chain API Examples

#### Basic Example

```go
package main

import (
    "fmt"
    "time"
    golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
    
    // Execute every 10 seconds
    scheduler.Every(10).Seconds().Do(func() {
        fmt.Println("Task runs every 10 seconds")
    })
    
    // Execute daily at 10:30 AM
    scheduler.Every().Day().At("10:30").Do(func() error {
        fmt.Println("Daily morning task")
        return nil
    })
    
    // Execute every Monday
    scheduler.Every().Monday().Do(func() {
        fmt.Println("Monday task")
    })
    
    scheduler.Start()
    defer scheduler.Stop()
    
    // Keep the program running
    select {}
}
```

#### Advanced Example

```go
package main

import (
    "fmt"
    "time"
    golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
    scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
    
    // Use time zone and timeout configuration
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
    
    // Complex business logic task
    scheduler.Every(2).Hours().Do(func() error {
        // Execute data synchronization
        fmt.Println("Data sync task running...")
        time.Sleep(5 * time.Second) // Simulate time-consuming operation
        return nil
    }, "data-sync")
    
    scheduler.Start()
    defer scheduler.Stop()
    
    // Display all tasks
    tasks := scheduler.GetTasks()
    fmt.Println("Scheduled tasks:")
    for _, task := range tasks {
        fmt.Printf("- %s: next run at %s\n", 
            task.ID, task.NextRunTime.Format("2006-01-02 15:04:05"))
    }
    
    select {}
}
```

## Configuration

### Cron Expression Syntax

GoLiteCron supports standard cron expressions with optional extensions:

#### Standard Format (5 fields)
```
* * * * *
│ │ │ │ │
│ │ │ │ └───── Day of week (0-7, Sunday is 0 or 7)
│ │ │ └─────── Month (1-12)
│ │ └───────── Day of month (1-31)
│ └─────────── Hour (0-23)
└───────────── Minute (0-59)
```

#### Extended Format (6 fields with seconds)
```
* * * * * *
│ │ │ │ │ │
│ │ │ │ │ └───── Day of week (0-7, Sunday is 0 or 7)
│ │ │ │ └─────── Month (1-12)
│ │ │ └───────── Day of month (1-31)
│ │ └─────────── Hour (0-23)
│ └───────────── Minute (0-59)
└─────────────── Second (0-59)
```

#### Predefined Macros
- `@yearly` or `@annually` - Run once a year at midnight on January 1
- `@monthly` - Run once a month at midnight on first day
- `@weekly` - Run once a week at midnight on Sunday
- `@daily` or `@midnight` - Run once a day at midnight
- `@hourly` - Run once an hour at the beginning of the hour
- `@minutely` - Run once a minute at the beginning of the minute

### Storage Types

GoLiteCron provides two storage implementations:

#### TimeWheel Storage
- Efficient for high-frequency tasks and large numbers of scheduled jobs
- Uses a multi-level time wheel algorithm
- Recommended for production environments with many tasks

```go
scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
```

#### Heap Storage
- Simple priority queue implementation
- Orders tasks by their next execution time
- Suitable for simple applications or development

```go
scheduler := golitecron.NewScheduler(golitecron.StorageTypeHeap)
```

### Task Options

#### WithLocation
Set the time zone for task execution:

```go
loc, _ := time.LoadLocation("America/New_York")
scheduler.AddTask("0 9 * * *", job, golitecron.WithLocation(loc))
```

#### WithTimeout
Set a timeout for task execution:

```go
scheduler.AddTask("0 */5 * * *", job, golitecron.WithTimeout(30*time.Second))
```

#### WithRetry
Set the number of retries for failed tasks:

```go
scheduler.AddTask("0 0 * * *", job, golitecron.WithRetry(3))
```

#### WithSeconds
Enable second-level precision in cron expressions:

```go
scheduler.AddTask("*/30 * * * * *", job, golitecron.WithSeconds())
```

#### WithYears
Enable year field in cron expressions:

```go
scheduler.AddTask("0 0 1 1 * 2024", job, golitecron.WithYears())
```

## Task Management

### Job Registration

Register jobs that can be referenced by name in configuration files:

```go
golitecron.RegisterJob("myJob", func() error {
    fmt.Println("Executing registered job")
    return nil
})
```

### Task Information

Get detailed information about tasks:

```go
// Get all tasks
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("Task ID: %s\n", task.ID)
    fmt.Printf("Cron Expression: %s\n", task.CronExpr)
    fmt.Printf("Next Run: %s\n", task.NextRunTime)
    fmt.Printf("Last Run: %s\n", task.LastRunTime)
}

// Get specific task info
taskInfo := scheduler.GetTaskInfo("task-id")
fmt.Println(taskInfo)
```

### Task Lifecycle

```go
// Add a task
err := scheduler.AddTask("0 */10 * * * *", job, golitecron.WithSeconds())

// Remove a task
taskToRemove := &golitecron.Task{ID: "task-id"}
scheduler.RemoveTask(taskToRemove)

// Start the scheduler
scheduler.Start()

// Stop the scheduler (graceful shutdown)
scheduler.Stop()
```

### Error Handling

GoLiteCron provides built-in error handling and recovery:

```go
// Jobs can return errors
scheduler.AddTask("0 */5 * * *", func() error {
    // Your logic here
    if someCondition {
        return fmt.Errorf("something went wrong")
    }
    return nil
}, golitecron.WithRetry(3)) // Will retry up to 3 times
```

## Best Practices

1. **Choose the right storage type**: Use TimeWheel for production with many tasks, Heap for simple use cases
2. **Set appropriate timeouts**: Prevent tasks from running indefinitely
3. **Use retry mechanisms**: Handle transient failures gracefully
4. **Register jobs for configuration**: Makes your configuration files more readable
5. **Handle panics**: GoLiteCron automatically recovers from panics, but you should still handle errors in your job functions
6. **Use descriptive task IDs**: Makes debugging and monitoring easier
7. **Consider time zones**: Always specify time zones for distributed applications
8. **Graceful shutdown**: Always call `Stop()` to ensure running tasks complete