package golitecron

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestScheduler_StopAndRestart tests that scheduler can be stopped and restarted
func TestScheduler_StopAndRestart(t *testing.T) {
	s := NewScheduler()

	runCount := int32(0)

	// First task
	job1, _ := WrapJob("restart-test-1", func() error {
		atomic.AddInt32(&runCount, 1)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job1, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// First start
	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	firstRunCount := atomic.LoadInt32(&runCount)
	if firstRunCount == 0 {
		t.Fatalf("expected at least one execution before first stop, got 0")
	}

	// Add a new task after stop (old task won't be rescheduled after stop - this is by design)
	job2, _ := WrapJob("restart-test-2", func() error {
		atomic.AddInt32(&runCount, 1)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job2, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask after stop failed: %v", err)
	}

	// Restart
	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	finalRunCount := atomic.LoadInt32(&runCount)
	if finalRunCount <= firstRunCount {
		t.Fatalf("expected more executions after restart, first=%d, final=%d", firstRunCount, finalRunCount)
	}
}

// TestScheduler_StopIsIdempotent tests that calling Stop multiple times is safe
func TestScheduler_StopIsIdempotent(t *testing.T) {
	s := NewScheduler()

	job, _ := WrapJob("stop-test", func() error {
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(100 * time.Millisecond)

	// Multiple stops should not panic
	s.Stop()
	s.Stop()
	s.Stop()
}

// TestScheduler_StartIsIdempotent tests that calling Start multiple times is safe
func TestScheduler_StartIsIdempotent(t *testing.T) {
	s := NewScheduler()

	runCount := int32(0)
	job, _ := WrapJob("start-test", func() error {
		atomic.AddInt32(&runCount, 1)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Multiple starts should not create multiple goroutines
	s.Start()
	s.Start()
	s.Start()

	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	// Should have reasonable execution count (not 3x)
	count := atomic.LoadInt32(&runCount)
	if count > 5 {
		t.Fatalf("too many executions, possibly multiple run goroutines: %d", count)
	}
}

// TestScheduler_StopWaitsForRunningTasks tests that Stop waits for tasks to complete
func TestScheduler_StopWaitsForRunningTasks(t *testing.T) {
	s := NewScheduler()

	taskStarted := make(chan struct{})
	taskFinished := make(chan struct{})

	job, _ := WrapJob("long-task", func() error {
		close(taskStarted)
		time.Sleep(500 * time.Millisecond)
		close(taskFinished)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()

	// Wait for task to start
	select {
	case <-taskStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not start within timeout")
	}

	// Stop should block until task finishes
	stopDone := make(chan struct{})
	go func() {
		s.Stop()
		close(stopDone)
	}()

	// Stop should not return before task finishes
	select {
	case <-stopDone:
		// Check if task finished
		select {
		case <-taskFinished:
			// Good: task finished before Stop returned
		default:
			t.Fatal("Stop returned before task finished")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return within timeout")
	}
}

// TestScheduler_ConcurrentTaskExecution tests multiple tasks running concurrently
func TestScheduler_ConcurrentTaskExecution(t *testing.T) {
	s := NewScheduler()

	task1Count := int32(0)
	task2Count := int32(0)

	job1, _ := WrapJob("task1", func() error {
		atomic.AddInt32(&task1Count, 1)
		return nil
	})

	job2, _ := WrapJob("task2", func() error {
		atomic.AddInt32(&task2Count, 1)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job1, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask job1 failed: %v", err)
	}

	if err := s.AddTask("*/1 * * * * *", job2, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask job2 failed: %v", err)
	}

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	count1 := atomic.LoadInt32(&task1Count)
	count2 := atomic.LoadInt32(&task2Count)

	if count1 == 0 || count2 == 0 {
		t.Fatalf("expected both tasks to execute, task1=%d, task2=%d", count1, count2)
	}
}

// TestScheduler_DuplicateTaskID tests that adding task with same ID fails
func TestScheduler_DuplicateTaskID(t *testing.T) {
	s := NewScheduler()

	job1, _ := WrapJob("duplicate-id", func() error { return nil })
	job2, _ := WrapJob("duplicate-id", func() error { return nil })

	if err := s.AddTask("* * * * *", job1); err != nil {
		t.Fatalf("first AddTask failed: %v", err)
	}

	err := s.AddTask("* * * * *", job2)
	if err == nil {
		t.Fatal("expected error when adding task with duplicate ID")
	}
}

// TestScheduler_RemoveNonExistentTask tests removing a task that doesn't exist
func TestScheduler_RemoveNonExistentTask(t *testing.T) {
	s := NewScheduler()

	task := &Task{ID: "non-existent"}
	removed := s.RemoveTask(task)

	if removed {
		t.Fatal("expected RemoveTask to return false for non-existent task")
	}
}

// TestScheduler_GetTaskInfoNotFound tests GetTaskInfo for non-existent task
func TestScheduler_GetTaskInfoNotFound(t *testing.T) {
	s := NewScheduler()

	info := s.GetTaskInfo("non-existent")
	if info == "" {
		t.Fatal("expected non-empty info string")
	}
	// Should contain "not found"
	if !contains_string(info, "not found") {
		t.Fatalf("expected 'not found' in info, got: %s", info)
	}
}

// TestScheduler_WithTimeWheel tests scheduler with TimeWheel storage
func TestScheduler_WithTimeWheel(t *testing.T) {
	s := NewScheduler(StorageTypeTimeWheel)

	runCount := int32(0)
	job, _ := WrapJob("timewheel-test", func() error {
		atomic.AddInt32(&runCount, 1)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	count := atomic.LoadInt32(&runCount)
	if count == 0 {
		t.Fatal("expected at least one execution with TimeWheel storage")
	}
}

// helper function
func contains_string(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains_substring(s, substr))
}

func contains_substring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
