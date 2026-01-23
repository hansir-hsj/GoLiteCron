package golitecron

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// TestTask_RetryOnError tests that task retries on error
func TestTask_RetryOnError(t *testing.T) {
	s := NewScheduler()

	attemptCount := int32(0)
	job := WrapJob("retry-test", func() error {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < 3 {
			return errors.New("intentional error")
		}
		return nil
	})

	// 2 retries means 3 total attempts
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithRetry(2), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(2 * time.Second)
	s.Stop()

	count := atomic.LoadInt32(&attemptCount)
	if count < 3 {
		t.Fatalf("expected at least 3 attempts (1 + 2 retries), got %d", count)
	}
}

// TestTask_TimeoutExecution tests that task timeout works
func TestTask_TimeoutExecution(t *testing.T) {
	s := NewScheduler()

	taskStarted := int32(0)
	taskCompleted := int32(0)

	job := WrapJob("timeout-test", func() error {
		atomic.AddInt32(&taskStarted, 1)
		time.Sleep(2 * time.Second) // Longer than timeout
		atomic.AddInt32(&taskCompleted, 1)
		return nil
	})

	// 100ms timeout
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithTimeout(100*time.Millisecond), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	started := atomic.LoadInt32(&taskStarted)
	if started == 0 {
		t.Fatal("expected task to start at least once")
	}
	// Note: taskCompleted might be 0 or non-zero depending on goroutine scheduling
	// The important thing is the scheduler didn't hang
}

// TestTask_TimeoutSkipsRetry tests that timeout skips retries
func TestTask_TimeoutSkipsRetry(t *testing.T) {
	s := NewScheduler()

	attemptCount := int32(0)

	job := WrapJob("timeout-retry-test", func() error {
		atomic.AddInt32(&attemptCount, 1)
		time.Sleep(500 * time.Millisecond) // Longer than timeout
		return nil
	})

	// 50ms timeout with 5 retries - but timeout should skip retries
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithTimeout(50*time.Millisecond), WithRetry(5), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	count := atomic.LoadInt32(&attemptCount)
	// Should only have 1 attempt per trigger (not 1 + 5 retries)
	// because timeout skips retries
	if count > 3 {
		t.Fatalf("expected few attempts due to timeout skipping retries, got %d", count)
	}
}

// TestTask_PanicRecovery tests that panic in task is recovered
func TestTask_PanicRecovery(t *testing.T) {
	s := NewScheduler()

	panicCount := int32(0)
	normalCount := int32(0)

	panicJob := WrapJob("panic-job", func() error {
		atomic.AddInt32(&panicCount, 1)
		panic("intentional panic")
	})

	normalJob := WrapJob("normal-job", func() error {
		atomic.AddInt32(&normalCount, 1)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", panicJob, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask panicJob failed: %v", err)
	}

	if err := s.AddTask("*/1 * * * * *", normalJob, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask normalJob failed: %v", err)
	}

	s.Start()
	time.Sleep(2 * time.Second)
	s.Stop()

	pCount := atomic.LoadInt32(&panicCount)
	nCount := atomic.LoadInt32(&normalCount)

	if pCount == 0 {
		t.Fatal("expected panic job to be triggered at least once")
	}

	if nCount == 0 {
		t.Fatal("expected normal job to continue running despite panic in other job")
	}
}

// TestTask_RemoveDuringExecution tests that task removed during execution is not rescheduled
func TestTask_RemoveDuringExecution(t *testing.T) {
	s := NewScheduler()

	executionCount := int32(0)
	taskStarted := make(chan struct{}, 1)
	removeConfirmed := make(chan struct{})
	var taskRef *Task

	job := WrapJob("remove-during-exec", func() error {
		count := atomic.AddInt32(&executionCount, 1)
		if count == 1 {
			// Signal that task has started
			select {
			case taskStarted <- struct{}{}:
			default:
			}
			// Wait for removal to be confirmed before continuing
			<-removeConfirmed
		}
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Get task reference BEFORE starting scheduler
	tasks := s.GetTasks()
	for _, task := range tasks {
		if task.ID == "remove-during-exec" {
			taskRef = task
			break
		}
	}

	if taskRef == nil {
		t.Fatal("failed to get task reference")
	}

	s.Start()

	// Wait for first execution to start
	select {
	case <-taskStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start within timeout")
	}

	// Remove task during execution (while it's waiting on removeConfirmed)
	s.RemoveTask(taskRef)

	// Now let the task continue
	close(removeConfirmed)

	// Wait a bit for any potential rescheduling
	time.Sleep(2 * time.Second)
	s.Stop()

	count := atomic.LoadInt32(&executionCount)
	// Should execute only once because it was removed during execution
	if count != 1 {
		t.Fatalf("expected task to execute exactly once after removal, got %d", count)
	}
}

// TestTask_PreventConcurrentExecution tests that same task doesn't run concurrently
func TestTask_PreventConcurrentExecution(t *testing.T) {
	s := NewScheduler()

	concurrentCount := int32(0)
	maxConcurrent := int32(0)

	job := WrapJob("no-concurrent", func() error {
		current := atomic.AddInt32(&concurrentCount, 1)
		defer atomic.AddInt32(&concurrentCount, -1)

		// Track max concurrent
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}

		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(3 * time.Second)
	s.Stop()

	max := atomic.LoadInt32(&maxConcurrent)
	if max > 1 {
		t.Fatalf("expected no concurrent execution of same task, max concurrent was %d", max)
	}
}

// TestTask_InvalidCronExpression tests that invalid cron expression returns error
func TestTask_InvalidCronExpression(t *testing.T) {
	s := NewScheduler()

	job := WrapJob("invalid-cron", func() error { return nil })

	testCases := []struct {
		expr string
		desc string
	}{
		{"* * * * * *", "6 fields without seconds enabled"},
		{"invalid", "non-cron string"},
		{"* * * *", "only 4 fields"},
		{"60 * * * *", "minute out of range"},
		{"* 24 * * *", "hour out of range"},
	}

	for _, tc := range testCases {
		err := s.AddTask(tc.expr, job)
		if err == nil {
			t.Errorf("expected error for %s: %s", tc.desc, tc.expr)
		}
	}
}

// TestTask_SuccessfulExecution tests normal successful execution
func TestTask_SuccessfulExecution(t *testing.T) {
	s := NewScheduler()

	results := make(chan int, 10)
	counter := int32(0)

	job := WrapJob("success-test", func() error {
		val := atomic.AddInt32(&counter, 1)
		results <- int(val)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(2500 * time.Millisecond)
	s.Stop()

	close(results)

	count := 0
	for range results {
		count++
	}

	if count < 2 {
		t.Fatalf("expected at least 2 successful executions, got %d", count)
	}
}

// TestTask_WithNegativeTimeout tests that negative timeout is treated as 0
func TestTask_WithNegativeTimeout(t *testing.T) {
	s := NewScheduler()

	executed := int32(0)
	job := WrapJob("neg-timeout", func() error {
		atomic.AddInt32(&executed, 1)
		return nil
	})

	// Negative timeout should be treated as 0 (no timeout)
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithTimeout(-100*time.Millisecond), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	count := atomic.LoadInt32(&executed)
	if count == 0 {
		t.Fatal("expected at least one execution")
	}
}

// TestTask_WithNegativeRetry tests that negative retry is treated as 0
func TestTask_WithNegativeRetry(t *testing.T) {
	s := NewScheduler()

	attemptCount := int32(0)
	job := WrapJob("neg-retry", func() error {
		atomic.AddInt32(&attemptCount, 1)
		return errors.New("intentional error")
	})

	// Negative retry should be treated as 0 (no retry)
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithRetry(-5), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	count := atomic.LoadInt32(&attemptCount)
	// With 0 retries, should only attempt once per trigger
	// In 1.5 seconds with 1-second interval, expect 1-2 attempts
	if count > 3 {
		t.Fatalf("expected few attempts with no retry, got %d", count)
	}
}