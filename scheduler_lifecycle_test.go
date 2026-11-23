package golitecron

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
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
	defer s.Stop()

	// Wait for task to start
	select {
	case <-taskStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not start within timeout")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- s.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown failed: %v", err)
		}
		// Check if task finished
		select {
		case <-taskFinished:
			// Good: task finished before Shutdown returned
		default:
			t.Fatal("Shutdown returned before task finished")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not return within timeout")
	}
}

// TestSchedulerStartDuringShutdownDoesNotBlock tests that Start returns while
// Shutdown is still waiting for tasks started before it.
func TestSchedulerStartDuringShutdownDoesNotBlock(t *testing.T) {
	s := NewScheduler()

	taskStarted := make(chan struct{})
	var startedOnce sync.Once
	job, _ := WrapJob("slow-drain", func() error {
		startedOnce.Do(func() { close(taskStarted) })
		time.Sleep(3 * time.Second)
		return nil
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// The restarted run may start another 3s execution; wait for it so it cannot leak.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	s.Start()
	select {
	case <-taskStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- s.Shutdown(ctx) }()

	time.Sleep(300 * time.Millisecond) // drain in progress

	startDone := make(chan struct{})
	go func() {
		s.Start()
		close(startDone)
	}()

	select {
	case <-startDone:
	case <-time.After(time.Second):
		t.Fatal("Start blocked on an in-progress Shutdown")
	}
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned %v before the running task finished", err)
	default:
	}
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown did not return")
	}
}

type panicLogger struct{}

func (panicLogger) Printf(string, ...any) { panic("logger failure") }

func TestSchedulerPanickingLoggerKeepsTaskScheduled(t *testing.T) {
	s := NewScheduler()
	if err := s.WithLogger(panicLogger{}); err != nil {
		t.Fatalf("WithLogger failed: %v", err)
	}
	var runs int32
	job, _ := WrapJob("logger-panic", func() error {
		atomic.AddInt32(&runs, 1)
		return errors.New("logged failure")
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	s.Start()
	time.Sleep(2500 * time.Millisecond)
	s.Stop()
	if got := atomic.LoadInt32(&runs); got < 2 {
		t.Fatalf("expected task to keep running despite logger panics, got %d runs", got)
	}
}

func TestSchedulerLifecycleInterleaving(t *testing.T) {
	s := NewScheduler()
	if err := s.WithTickDuration(5 * time.Millisecond); err != nil {
		t.Fatalf("WithTickDuration failed: %v", err)
	}
	var runs int32
	job, _ := WrapJob("interleave", func(ctx context.Context) error {
		atomic.AddInt32(&runs, 1)
		select {
		case <-ctx.Done():
		case <-time.After(20 * time.Millisecond):
		}
		return nil
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Run for at least two second boundaries so Shutdown also meets running tasks.
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for w := 0; w < 3; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; i < 220; i++ {
					switch (i + w) % 3 {
					case 0:
						s.Start()
					case 1:
						s.Stop()
					default:
						ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
						_ = s.Shutdown(ctx)
						cancel()
					}
					time.Sleep(10 * time.Millisecond)
				}
			}(w)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("interleaved Start/Stop/Shutdown deadlocked")
	}
	s.Stop()
	if atomic.LoadInt32(&runs) == 0 {
		t.Fatal("expected the task to run during the interleaving")
	}
}

func TestSchedulerConfigurationValidation(t *testing.T) {
	s := NewScheduler()
	if err := s.WithLogger(nil); err == nil {
		t.Fatal("expected nil logger error")
	}
	if err := s.WithTickDuration(0); err == nil {
		t.Fatal("expected invalid tick duration error")
	}
	if err := s.WithTickDuration(10 * time.Millisecond); err != nil {
		t.Fatalf("WithTickDuration failed: %v", err)
	}
	s.Start()
	defer s.Stop()
	if err := s.WithTickDuration(time.Second); err == nil {
		t.Fatal("expected error while scheduler is running")
	}
}

func TestSchedulerShutdownFromJobReturnsError(t *testing.T) {
	s := NewScheduler()
	result := make(chan error, 1)
	job, _ := WrapJob("self-shutdown", func(ctx context.Context) error {
		result <- s.Shutdown(ctx)
		return nil
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	s.Start()
	defer s.Stop()
	select {
	case err := <-result:
		if !errors.Is(err, ErrShutdownFromJob) {
			t.Fatalf("Shutdown error = %v, want %v", err, ErrShutdownFromJob)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Shutdown called from job deadlocked")
	}
}

func TestSchedulerRepeatedShutdownWaitsForEarlierTasks(t *testing.T) {
	s := NewScheduler()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once
	job, _ := WrapJob("repeat-shutdown", func() error {
		once.Do(func() {
			close(started)
			<-release
			close(finished)
		})
		return nil
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	s.Start()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start")
	}

	short, cancelShort := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelShort()
	if err := s.Shutdown(short); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first Shutdown error = %v, want deadline exceeded", err)
	}

	second := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		second <- s.Shutdown(ctx)
	}()
	select {
	case err := <-second:
		t.Fatalf("second Shutdown returned %v before the earlier task finished", err)
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	if err := <-second; err != nil {
		t.Fatalf("second Shutdown failed: %v", err)
	}
	select {
	case <-finished:
	default:
		t.Fatal("second Shutdown returned before the task finished")
	}
}

func TestSchedulerShutdownFromJobWithNewContextCannotDeadlock(t *testing.T) {
	s := NewScheduler()
	result := make(chan error, 1)
	job, _ := WrapJob("self-shutdown-new-context", func(context.Context) error {
		result <- s.Shutdown(context.Background())
		return nil
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	s.Start()
	defer s.Stop()
	select {
	case err := <-result:
		if !errors.Is(err, ErrShutdownDeadlineRequired) {
			t.Fatalf("Shutdown error = %v, want %v", err, ErrShutdownDeadlineRequired)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Shutdown called from job deadlocked")
	}
}

func TestScheduler_ShutdownCancelsRunningContextJobs(t *testing.T) {
	s := NewScheduler()

	taskStarted := make(chan struct{})
	taskCancelled := make(chan struct{})

	job, _ := WrapJob("cancel-on-shutdown", func(ctx context.Context) error {
		close(taskStarted)
		<-ctx.Done()
		close(taskCancelled)
		return ctx.Err()
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()

	select {
	case <-taskStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not start within timeout")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	select {
	case <-taskCancelled:
	default:
		t.Fatal("Shutdown returned before canceling the running job context")
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

	task := &task{ID: "non-existent"}
	removed := s.RemoveTaskByID(task.ID)

	if removed {
		t.Fatal("expected RemoveTask to return false for non-existent task")
	}
}

// TestScheduler_RemoveTaskByID tests removing a task using only its ID.
func TestScheduler_RemoveTaskByID(t *testing.T) {
	s := NewScheduler()

	job, _ := WrapJob("remove-by-id", func() error { return nil })
	if err := s.AddTask("* * * * *", job, WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if !s.RemoveTaskByID("remove-by-id") {
		t.Fatal("expected RemoveTaskByID to remove existing task")
	}

	if s.taskStorage.TaskExist("remove-by-id") {
		t.Fatal("expected task to be removed from storage")
	}

	if s.RemoveTaskByID("missing-task") {
		t.Fatal("expected RemoveTaskByID to return false for missing task")
	}
}

func TestScheduler_StopCanBeCalledFromRunningTask(t *testing.T) {
	s := NewScheduler()
	done := make(chan struct{})

	job, _ := WrapJob("self-stop", func() error {
		s.Stop()
		close(done)
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop called from a running task deadlocked")
	}
}

func TestScheduler_StartWaitsForStopToFinish(t *testing.T) {
	s := NewScheduler()
	oldStopChan := make(chan struct{})
	oldRunDone := make(chan struct{})
	s.stopChan = oldStopChan
	s.runDone = oldRunDone
	atomic.StoreInt32(&s.running, 1)

	stopStarted := make(chan struct{})
	stopBlocked := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(stopStarted)
		s.Stop()
	}()

	<-stopStarted
	<-oldStopChan
	time.AfterFunc(50*time.Millisecond, func() {
		close(stopBlocked)
	})
	<-stopBlocked

	startReturned := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Start()
		close(startReturned)
	}()

	select {
	case <-startReturned:
		t.Fatal("Start returned before the previous Stop completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(oldRunDone)
	wg.Wait()

	select {
	case <-startReturned:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after the previous Stop completed")
	}

	s.Stop()
}

func TestScheduler_WithLoggerRejectsRunningScheduler(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("logger-running", func() error { return nil })
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	defer s.Stop()

	if err := s.WithLogger(log.New(io.Discard, "", 0)); err == nil {
		t.Fatal("expected WithLogger to reject changes while scheduler is running")
	}
}

func TestScheduler_AddTaskDoesNotHoldTaskLockWhileParsing(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("lock-scope", func() error { return nil })

	option := testTaskOption(func(*taskSettings) {
		s.RemoveTaskByID("missing")
	})

	done := make(chan error, 1)
	go func() {
		done <- s.AddTask("* * * * *", job, option)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("AddTask held task lock while parsing cron options")
	}
}

type testTaskOption func(*taskSettings)

func (t testTaskOption) applyTaskOption(settings *taskSettings) {
	t(settings)
}

// TestScheduler_RemoveTaskByIDDuringExecutionDoesNotReschedule verifies ID-based
// removal marks the running task so it cannot requeue itself after finishing.
func TestScheduler_RemoveTaskByIDDuringExecutionDoesNotReschedule(t *testing.T) {
	s := NewScheduler()

	executionCount := int32(0)
	taskStarted := make(chan struct{}, 1)
	releaseTask := make(chan struct{})

	job, _ := WrapJob("remove-by-id-running", func() error {
		count := atomic.AddInt32(&executionCount, 1)
		if count == 1 {
			taskStarted <- struct{}{}
			<-releaseTask
		}
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()

	select {
	case <-taskStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start within timeout")
	}

	if !s.RemoveTaskByID("remove-by-id-running") {
		t.Fatal("expected RemoveTaskByID to mark running task as removed")
	}

	close(releaseTask)
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	if count := atomic.LoadInt32(&executionCount); count != 1 {
		t.Fatalf("expected task to execute once after ID-based removal, got %d", count)
	}
}

func TestScheduler_AddTaskRejectsRunningTaskID(t *testing.T) {
	s := NewScheduler()

	taskStarted := make(chan struct{}, 1)
	releaseTask := make(chan struct{})

	job, _ := WrapJob("running-duplicate", func() error {
		taskStarted <- struct{}{}
		<-releaseTask
		return nil
	})

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	s.Start()
	defer s.Stop()

	select {
	case <-taskStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start within timeout")
	}

	duplicateJob, _ := WrapJob("running-duplicate", func() error { return nil })
	if err := s.AddTask("*/1 * * * * *", duplicateJob, WithSeconds(), WithLocation(time.UTC)); err == nil {
		close(releaseTask)
		t.Fatal("expected AddTask to reject ID that is currently running")
	}

	close(releaseTask)
}

func TestScheduler_GetTaskNotFound(t *testing.T) {
	s := NewScheduler()

	if _, ok := s.GetTask("non-existent"); ok {
		t.Fatal("expected GetTask to return false for a non-existent task")
	}
}
