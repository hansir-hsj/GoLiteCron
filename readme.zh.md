# GoLiteCron

[en](readme.md) [zh](readme.zh.md)

轻量化的Cron框架

### 安装方法
```
go get -u github.com/GoLite/GoLiteCron
```

### 使用方法
```
// 定义任务
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

// 创建调度器-基于时间轮
scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)
// 创建调度器-基于堆
// scheduler := cron.NewScheduler(cron.StorageTypeHeap)

// 注册任务
scheduler.AddTask(job.GetID(), job, expr)

// 启动调度器
scheduler.Start()
```