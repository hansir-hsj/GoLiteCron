// Error handling example: timeout, retry, and error logging.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	s := cron.NewScheduler()

	// Task with 5s timeout
	s.AddTask("* * * * *", jobWithTimeout(), cron.WithTimeout(5*time.Second))

	// Task with 3 retries on failure
	s.AddTask("* * * * *", jobThatFails(), cron.WithRetry(3))

	// Task with both timeout and retry
	s.AddTask("* * * * *", reliableJob(),
		cron.WithTimeout(10*time.Second),
		cron.WithRetry(2),
	)

	s.Start()
	defer s.Stop()

	fmt.Println("Scheduler running. Press Ctrl+C to stop.")
	time.Sleep(5 * time.Minute)
}

func jobWithTimeout() cron.Job {
	job, _ := cron.WrapJob("timeout-job", func(ctx context.Context) error {
		select {
		case <-time.After(3 * time.Second):
			fmt.Println("Job completed")
			return nil
		case <-ctx.Done():
			fmt.Println("Job timed out")
			return ctx.Err()
		}
	})
	return job
}

func jobThatFails() cron.Job {
	attempt := 0
	job, _ := cron.WrapJob("retry-job", func() error {
		attempt++
		if attempt < 3 {
			fmt.Printf("Attempt %d failed\n", attempt)
			return errors.New("temporary error")
		}
		fmt.Printf("Attempt %d succeeded\n", attempt)
		attempt = 0
		return nil
	})
	return job
}

func reliableJob() cron.Job {
	job, _ := cron.WrapJob("reliable-job", func(ctx context.Context) error {
		fmt.Println("Running reliable job")
		return nil
	})
	return job
}
