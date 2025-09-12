# GoLiteCron

[Chinese](readme.zh.md)

## Overview
GoLiteCron is a lightweight, high-performance cron job scheduling framework for Go applications. It provides a simple yet powerful interface for managing scheduled tasks with support for various cron expressions, time zones, task timeouts, and retries. The framework offers flexible storage options (TimeWheel and Heap) to suit different application scenarios.

## Features
- Support Cron expressions
  - Standard cron syntax (minutes, hours, day of month, month, day of week)
  - Extended syntax with seconds (using WithSeconds() option)
  - Year specification (using WithYears() option)
  - Predefined macros (@yearly, @monthly, @weekly, @daily, @hourly, @minutely)
- Flexible Storage Options
  - TimeWheel: Efficient for high-frequency tasks and large numbers of scheduled jobs
  - Heap: Simple implementation suitable for general-purpose scheduling
- Task Management Features
  - Custom time zones for task execution
  - Configurable timeout for task execution
  - Automatic retry mechanism for failed tasks
  - Task registration by ID for easy management
  - Support for loading tasks from configuration files (YAML/JSON)
- Reliability
  - Panic recovery for individual tasks
  - Atomic operations for task status management
  - Proper resource cleanup and graceful shutdown

## Installation
```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## Quick Start
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
- Using Custom Cron Expressions
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

- Configuring Task Options

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

- Loading Tasks from Configuration Files
1. Create a YAML configuration file (cron.yaml):
```go
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

2. Load and register tasks:
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

- Managing Tasks
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

### Architecture
GoLiteCron consists of several key components:
- Scheduler: The core component that manages task execution and coordinates with the storage backend.
- Task Storage: Implements the storage and retrieval of tasks. Two implementations are provided:
  - TimeWheel: A multi-level time wheel implementation that efficiently handles large numbers of tasks with varying intervals.
  - Heap: A priority queue implementation that orders tasks by their next execution time.
- Cron Parser: Parses cron expressions and calculates the next execution time for tasks.
- Job Registry: Manages job functions that can be referenced by name in configuration files.
- Config Loader: Loads task configurations from YAML or JSON files.

## License
GoLiteCron is released under the MIT License. See the [LICENSE](LICENSE) file for details.