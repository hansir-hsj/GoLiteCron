# GoLiteCron

[英文](readme.md)

## 概述
GoLiteCron 是一个轻量级、高性能的 cron 任务调度框架，适用于 Go 应用程序。它提供了简单而强大的接口来管理定时任务，支持各种 cron 表达式、时区、任务超时和重试功能。该框架提供了灵活的存储选项（TimeWheel 和 Heap），以适应不同的应用场景。

## 特性
- Cron 表达式支持
  - 标准 cron 语法（分钟、小时、日、月、星期）
  - 扩展语法支持秒级调度（使用 WithSeconds() 选项）
  - 支持年份指定（使用 WithYears() 选项）
  - 预定义宏（@yearly, @monthly, @weekly, @daily, @hourly, @minutely）
- 灵活的存储选项
  - TimeWheel：高效处理高频任务和大量定时作业
  - Heap：简单实现，适用于通用调度场景
- 任务管理功能
  - 任务管理功能
  - 可配置的任务执行超时时间
  - 失败任务的自动重试机制
  - 通过 ID 注册任务，便于管理
  - 支持从配置文件（YAML/JSON）加载任务
- 可靠性
  - 单个任务的 panic 恢复
  - 任务状态的原子操作管理
  - 适当的资源清理和优雅关闭

## 安装
```bash
go get -u github.com/hansir-hsj/GoLiteCron
```

## 快速开始
```go
package main

import (
	"fmt"
	"time"
	golitecron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	// 创建一个使用 TimeWheel 存储的调度器
	scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)
	
	// 添加一个每分钟运行的任务
	err := scheduler.AddTask("@minutely", golitecron.WrapJob("minute-task", func() error {
		fmt.Printf("分钟任务在 %s 执行\n", time.Now().Format(time.RFC3339))
		return nil
	}))
	
	if err != nil {
		fmt.Printf("添加任务失败: %v\n", err)
		return
	}
	
	// 启动调度器
	scheduler.Start()
	defer scheduler.Stop()
	
	// 保持程序运行
	select {}
}
```
    
## 高级用法

- 使用自定义 Cron 表达式
```go
// 添加一个每30分钟运行一次的任务
scheduler.AddTask("*/30 * * * *", golitecron.WrapJob("30min-task", func() error {
    fmt.Println("每30分钟运行一次的任务")
    return nil
}))

// 添加一个每2秒运行一次的任务（需要 WithSeconds 选项）
scheduler.AddTask("0/2 * * * * *", golitecron.WrapJob("2sec-task", func() error {
    fmt.Println("每2秒运行一次的任务")
    return nil
}), golitecron.WithSeconds())
```

- 配置任务选项
```go
// 创建上海时区
shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

// 添加一个带有自定义选项的任务
scheduler.AddTask("0 9 * * 1-5", golitecron.WrapJob("workday-task", func() error {
    fmt.Println("工作日上海时间上午9点运行的任务")
    return nil
}), 
    golitecron.WithLocation(shanghaiLoc),    // 使用上海时区
    golitecron.WithTimeout(5*time.Second),   // 5秒后超时
    golitecron.WithRetry(3)                  // 失败最多重试3次
)
```

- 从配置文件加载任务
1. 创建 YAML 配置文件（cron.yaml）：
```go
tasks:
- id: "daily-task"
  cron_expr: "0 0 * * *"
  timeout: 30000        # 30秒（毫秒）
  retry: 2
  location: "Asia/Shanghai"
  func_name: "dailyJob"
- id: "hourly-task"
  cron_expr: "0 * * * *"
  timeout: 10000        # 10秒（毫秒）
  retry: 1
  location: "UTC"
  func_name: "hourlyJob"
```

2. 加载并注册任务：
```go
// 注册作业函数
golitecron.RegisterJob("dailyJob", func() error {
    fmt.Println("执行每日任务")
    return nil
})

golitecron.RegisterJob("hourlyJob", func() error {
    fmt.Println("执行每小时任务")
    return nil
})

// 加载配置
config, err := golitecron.LoadFromYaml("cron.yaml")
if err != nil {
    fmt.Printf("加载配置错误: %v\n", err)
    return
}

// 创建调度器并加载任务
scheduler := golitecron.NewScheduler(golitecron.StorageTypeHeap)
err = scheduler.LoadTasksFromConfig(config)
if err != nil {
    fmt.Printf("加载任务错误: %v\n", err)
    return
}

scheduler.Start()
defer scheduler.Stop()

// 保持运行
select {}
```

- 管理任务
```go
// 获取所有任务
tasks := scheduler.GetTasks()
for _, task := range tasks {
    fmt.Printf("任务ID: %s, 下次运行时间: %s\n", task.ID, task.NextRunTime)
}

// 获取任务信息
taskInfo := scheduler.GetTaskInfo("daily-task")
fmt.Println(taskInfo)

// 移除任务
taskToRemove := &golitecron.Task{ID: "hourly-task"}
scheduler.RemoveTask(taskToRemove)
```

### 架构

GoLiteCron 由几个关键组件组成：
- 调度器（Scheduler）：核心组件，管理任务执行并与存储后端协调。
- 任务存储（Task Storage）：实现任务的存储和检索。提供两种实现：
  - TimeWheel: 多级时间轮实现，高效处理大量不同间隔的任务。
  - Heap: 优先级队列实现，按下次执行时间排序任务。
- Cron 解析器（Cron Parser）：解析 cron 表达式并计算任务的下次执行时间。
- 作业注册表（Job Registry）：管理可在配置文件中通过名称引用的作业函数。
- 配置加载器（Config Loader）：从 YAML 或 JSON 文件加载任务配置。

## 许可证
GoLiteCron 以 MIT 许可证发布。详情请参见 [LICENSE](LICENSE) 文件。
