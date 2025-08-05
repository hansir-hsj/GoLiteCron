# GoLiteCron

[en](readme.md) [zh](readme.zh.md)

A lightweight Cron framework

### Installation
```
go get -u github.com/GoLite/GoLiteCron
```

### Features
- Support Cron expressions
  - see [Wiki](https://en.wikipedia.org/wiki/Cron)
  - Example: 
    - `*/30 * * * *` every 30 minutes
    - `0/2 * * * * *` every 2 seconds (must use WithSeconds option)
    - `0 * * * * * 2025-2026` every seconds in 2025-2026 (must use WithYears option)
- Support multiple storage types: TimeWheel and Heap
- Support custom time zone
  - Example: `scheduler.AddTask("@minutely", cron.WithLocation(time.LoadLocation("Asia/Shanghai")))`
  - Default is UTC
- Support task timeout
  - Example: `scheduler.AddTask("@minutely", cron.WithTimeout(10*time.Second))`
- Support task retry
  - Example: `scheduler.AddTask("@minutely", cron.WithRetry(3))`

### Usage
```go
// Create a new scheduler based on TimeWheel
scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)

// Register the task
scheduler.AddTask("@minutely", cron.WrapJob("minutely-job", func() error {
	fmt.Printf("Job %s is running at %s\n", "minutely-job", time.Now().Format(time.RFC3339))
	return nil
}))

// Start the scheduler
scheduler.Start()
```