// Basic example: minimal setup to run a scheduled task.
package main

import (
	"fmt"
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	s := cron.NewScheduler()

	// Add a task that runs every minute
	s.AddTask("* * * * *", myJob("hello"))

	s.Start()
	defer s.Stop()

	fmt.Println("Scheduler running. Press Ctrl+C to stop.")
	time.Sleep(5 * time.Minute)
}

func myJob(msg string) cron.Job {
	job, _ := cron.WrapJob("my-task", func() error {
		fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), msg)
		return nil
	})
	return job
}
