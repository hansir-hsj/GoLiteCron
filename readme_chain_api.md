# GoLiteCron Chain API Documentation

## Overview

To provide a more user-friendly experience, GoLiteCron has added a chain API similar to Python's schedule library, allowing you to define scheduled tasks using more natural language.

## New Features

### Chain API Syntax

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

## API Reference

### ScheduleBuilder Methods

#### Interval Methods
- `Every(intervals ...int) *ScheduleBuilder` - Start building, optionally specify interval count

#### Time Unit Methods
- `Second() / Seconds() *ScheduleBuilder` - second unit
- `Minute() / Minutes() *ScheduleBuilder` - minute unit
- `Hour() / Hours() *ScheduleBuilder` - hour unit
- `Day() / Days() *ScheduleBuilder` - day unit
- `Week() / Weeks() *ScheduleBuilder` - week unit
- `Month() / Months() *ScheduleBuilder` - month unit

#### Weekday Methods
- `Monday() / Tuesday() / ... / Sunday() *ScheduleBuilder` - specify day of the week

#### Time Specification Methods
- `At(timeStr string) *ScheduleBuilder` - specify execution time (HH:MM or HH:MM:SS)

#### Option Configuration Methods
- `WithTimeout(timeout time.Duration) *ScheduleBuilder` - set timeout duration
- `WithRetry(retry int) *ScheduleBuilder` - set retry count
- `WithLocation(loc *time.Location) *ScheduleBuilder` - set time zone
- `WithSeconds() *ScheduleBuilder` - enable second-level precision
- `WithYears() *ScheduleBuilder` - enable year field

#### Execution Methods
- `Do(job interface{}, taskID ...string) error` - add task to scheduler

## Usage Examples

### Basic Example

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

### Advanced Example

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

## Compatibility with Existing API

The chain API is an enhancement built on top of the existing API, fully compatible with existing code:

```go
// Existing API still available
scheduler.AddTask("0 */30 * * *", golitecron.WrapJob("legacy-task", func() error {
    return nil
}))

// New chain API
scheduler.Every(30).Minutes().Do(func() error {
    return nil
}, "chain-task")
```

## Important Notes

1. **Time Zone Handling**: Use `WithLocation()` to specify time zone, defaults to system local time zone
2. **Second-level Precision**: Use `At("HH:MM:SS")` format or `WithSeconds()` to enable second-level scheduling
3. **Task ID**: You can customize task ID, otherwise a descriptive ID will be auto-generated
4. **Error Handling**: All chain calls are finally error-checked in the `Do()` method
5. **Performance**: Chain API is ultimately converted to standard cron expressions, performance is the same as the original API

## Migration Guide

Migrating from the existing API to the chain API is very simple:

```go
// Original approach
scheduler.AddTask("*/10 * * * * *", 
    golitecron.WrapJob("task", func() error { return nil }),
    golitecron.WithSeconds())

// New chain approach
scheduler.Every(10).Seconds().Do(func() error { return nil }, "task")
```

The chain API makes code more intuitive and readable, recommended for use in new projects.