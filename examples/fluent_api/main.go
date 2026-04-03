// Fluent API example: chain-style task scheduling.
package main

import (
	"fmt"
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	s := cron.NewScheduler()

	// Every 30 seconds
	s.Every(30).Seconds().Do(func() {
		fmt.Println("Every 30 seconds")
	})

	// Every 5 minutes
	s.Every(5).Minutes().Do(func() {
		fmt.Println("Every 5 minutes")
	})

	// Daily at 09:30
	s.Every().Day().At("09:30").Do(func() {
		fmt.Println("Daily at 09:30")
	})

	// Every Monday at 10:00
	s.Every().Monday().At("10:00").Do(func() {
		fmt.Println("Monday at 10:00")
	})

	// Monthly on the 1st at 00:00
	s.Every().Month().At("00:00").Do(func() {
		fmt.Println("Monthly task")
	})

	s.Start()
	defer s.Stop()

	fmt.Println("Scheduler running. Press Ctrl+C to stop.")
	time.Sleep(5 * time.Minute)
}
