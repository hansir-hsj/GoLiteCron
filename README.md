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

### Usage
```
// Define the task
type MyJob struct {
	ID string
}

func (j *MyJob) Execute() error {
	fmt.Printf("Job %s is running at %s\n", j.ID, time.Now().Format(time.RFC3339))
	return nil
}

func (j *MyJob) GetID() string {
	return j.ID
}

job := &MyJob{ID: "every-30-min-job"}
expr, err := cron.NewStandardCronParser("*/30 * * * *")
if err != nil {
    log.Fatalf("Failed to parse cron expression: %v", err)
}

// Create a new scheduler based on TimeWheel
scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)
// Create a new schedule based on Heap
// scheduler := cron.NewScheduler(cron.StorageTypeHeap)

// Register the task
scheduler.AddTask(job.GetID(), job, expr)

// Start the scheduler
scheduler.Start()
```