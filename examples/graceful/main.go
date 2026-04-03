// Graceful shutdown example: handle OS signals properly.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	s := cron.NewScheduler()

	s.Every(10).Seconds().Do(func() {
		fmt.Printf("[%s] Task running...\n", time.Now().Format("15:04:05"))
		time.Sleep(2 * time.Second) // Simulate work
		fmt.Printf("[%s] Task done\n", time.Now().Format("15:04:05"))
	})

	s.Start()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Scheduler running. Press Ctrl+C to stop.")
	<-quit

	fmt.Println("\nShutting down gracefully...")
	s.Stop() // Waits for running tasks to complete
	fmt.Println("Scheduler stopped.")
}
