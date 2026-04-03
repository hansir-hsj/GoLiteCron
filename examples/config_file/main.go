// Config file example: load tasks from YAML or JSON.
package main

import (
	"context"
	"fmt"
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	// Register job functions before loading config
	cron.RegisterJob("sendReport", sendReport)
	cron.RegisterJob("cleanup", cleanup)

	// Load from YAML (or use LoadFromJSON for JSON)
	cfg, err := cron.LoadFromYAML("tasks.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	s := cron.NewScheduler()
	if err := s.LoadTasksFromConfig(cfg); err != nil {
		fmt.Printf("Failed to apply config: %v\n", err)
		return
	}

	s.Start()
	defer s.Stop()

	fmt.Println("Loaded tasks from config. Press Ctrl+C to stop.")
	time.Sleep(5 * time.Minute)
}

func sendReport(ctx context.Context) error {
	fmt.Println("Sending report...")
	return nil
}

func cleanup(ctx context.Context) error {
	fmt.Println("Running cleanup...")
	return nil
}

/*
Example tasks.yaml:

tasks:
  - id: daily-report
    cron_expr: "0 9 * * *"
    func_name: sendReport
    timeout: 30s

  - id: weekly-cleanup
    cron_expr: "0 0 * * 0"
    func_name: cleanup
    retry: 3
*/
