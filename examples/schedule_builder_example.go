package main

import (
	"fmt"
	"time"

	golitecron "github.com/hansir-hsj/GoLiteCron"
)

// 示例任务函数
func printJob(message string) func() error {
	return func() error {
		fmt.Printf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
		return nil
	}
}

func main() {
	// 创建调度器
	scheduler := golitecron.NewScheduler(golitecron.StorageTypeTimeWheel)

	// 使用链式API添加各种任务
	fmt.Println("Setting up scheduled tasks with chain API...")

	// 每10秒执行一次
	err := scheduler.Every(10).Seconds().Do(printJob("Task runs every 10 seconds"))
	if err != nil {
		fmt.Printf("Error adding 10-second task: %v\n", err)
		return
	}

	err = scheduler.Every(30).Seconds().Do(printJob("Task runs every 30 seconds"))
	if err != nil {
		fmt.Printf("Error adding 30-second task: %v\n", err)
		return
	}

	// 每5分钟执行一次
	err = scheduler.Every(5).Minutes().Do(printJob("Task runs every 5 minutes"))
	if err != nil {
		fmt.Printf("Error adding 5-minute task: %v\n", err)
		return
	}

	// 每天10:30执行
	err = scheduler.Every().Day().At("10:30").Do(printJob("Daily task at 10:30"))
	if err != nil {
		fmt.Printf("Error adding daily task: %v\n", err)
		return
	}

	// 每周一执行
	err = scheduler.Every().Monday().Do(printJob("Task runs every Monday"))
	if err != nil {
		fmt.Printf("Error adding Monday task: %v\n", err)
		return
	}

	// 每周三14:15执行
	err = scheduler.Every().Wednesday().At("14:15").Do(printJob("Task runs every Wednesday at 14:15"))
	if err != nil {
		fmt.Printf("Error adding Wednesday task: %v\n", err)
		return
	}

	// 每2周执行
	err = scheduler.Every(2).Weeks().Do(printJob("Task runs every 2 weeks"))
	if err != nil {
		fmt.Printf("Error adding bi-weekly task: %v\n", err)
		return
	}

	// 支持不同的函数签名
	// 1. 无返回值函数
	err = scheduler.Every(20).Seconds().Do(func() {
		fmt.Println("Simple function without return value")
	})
	if err != nil {
		fmt.Printf("Error adding simple task: %v\n", err)
		return
	}

	// 2. 带错误返回值的函数
	err = scheduler.Every().Hour().Do(func() error {
		fmt.Println("Function with error return")
		return nil
	})
	if err != nil {
		fmt.Printf("Error adding error-return task: %v\n", err)
		return
	}

	// 启动调度器
	scheduler.Start()
	defer scheduler.Stop()

	// 显示所有任务信息
	fmt.Println("\nScheduled tasks:")
	tasks := scheduler.GetTasks()
	for _, task := range tasks {
		fmt.Printf("- Task ID: %s, Next Run: %s\n",
			task.ID, task.NextRunTime.Format("2006-01-02 15:04:05"))
	}

	// 运行一段时间来观察任务执行
	fmt.Println("\nScheduler is running. Press Ctrl+C to stop...")
	time.Sleep(2 * time.Minute)
}
