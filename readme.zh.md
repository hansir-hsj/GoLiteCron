# GoLiteCron

[en](readme.md) [zh](readme.zh.md)

轻量化的Cron框架

### 安装方法
```
go get -u github.com/GoLite/GoLiteCron
```

### 特性
- Cron表达式
  - 参考 [Wiki](https://en.wikipedia.org/wiki/Cron)
  - 样例: 
    - `*/30 * * * *` 每30分钟一次
    - `0/2 * * * * *` 每两秒一次 (必须使用WithSeconds选项)
    - `0 * * * * * 2025-2026` 2025年到2026年每秒一次 (必须使用WithYears选项)
- 支持多个存储类型: TimeWheel 和 Heap
- 支持任务时区选项
  - 样例: `scheduler.AddTask("@minutely", cron.WithLocation(time.LoadLocation("Asia/Shanghai")))`
  - 默认 `UTC`
- 支持任务超时时间
  - 样例: `scheduler.AddTask("@minutely", cron.WithTimeout(10*time.Second))`
- 支持任务重试次数
  - 样例: `scheduler.AddTask("@minutely", cron.WithRetry(1))`
  

### 使用方法
```go
// 创建调度器-基于时间轮
scheduler := cron.NewScheduler(cron.StorageTypeTimeWheel)

// 注册任务
scheduler.AddTask("@minutely", cron.WrapJob("minutely-job", func() error {
	fmt.Printf("Job %s is running at %s\n", "minutely-job", time.Now().Format(time.RFC3339))
	return nil
}))

// 启动调度器
scheduler.Start()
```