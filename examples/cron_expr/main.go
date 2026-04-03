// Cron expression example: standard and extended cron syntax.
package main

import (
	"fmt"
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	s := cron.NewScheduler()

	// Standard 5-field: minute hour day month weekday
	s.AddTask("*/5 * * * *", task("Every 5 minutes"))
	s.AddTask("0 9 * * 1-5", task("Weekdays at 9:00"))
	s.AddTask("0 0 1 * *", task("Monthly on 1st"))

	// 6-field with seconds: second minute hour day month weekday
	s.AddTask("*/10 * * * * *", task("Every 10 seconds"), cron.WithSeconds())

	// 7-field with year: second minute hour day month weekday year
	s.AddTask("0 0 0 1 1 * 2025", task("New Year 2025"), cron.WithSeconds(), cron.WithYears())

	// Special characters: L (last), W (weekday)
	s.AddTask("0 0 L * *", task("Last day of month"))
	s.AddTask("0 0 15W * *", task("Nearest weekday to 15th"))

	// Predefined macros
	s.AddTask(cron.Daily, task("Daily macro"))
	s.AddTask(cron.Hourly, task("Hourly macro"))

	s.Start()
	defer s.Stop()

	fmt.Println("Scheduler running. Press Ctrl+C to stop.")
	time.Sleep(5 * time.Minute)
}

func task(name string) cron.Job {
	job, _ := cron.WrapJob(name, func() error {
		fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), name)
		return nil
	})
	return job
}
