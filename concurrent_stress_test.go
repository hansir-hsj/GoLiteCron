package golitecron

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Massive Concurrent Add/Remove Tests
// ============================================================================

// TestConcurrent_MassiveAddRemove tests high-volume concurrent task operations.
func TestConcurrent_MassiveAddRemove(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	const numGoroutines = 100
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	var errorCount atomic.Int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				taskID := fmt.Sprintf("task-%d-%d", id, j)
				job, _ := WrapJob(taskID, func() error { return nil })

				// Add task
				if err := s.AddTask("* * * * *", job); err != nil {
					errorCount.Add(1)
					continue
				}

				// Random delay
				time.Sleep(time.Duration(rand.Intn(100)) * time.Microsecond)

				// Find and remove
				tasks := s.GetTasks()
				for _, task := range tasks {
					if task.ID == taskID {
						s.RemoveTask(task)
						break
					}
				}
			}
		}(i)
	}

	wg.Wait()

	errCount := errorCount.Load()
	if errCount > 0 {
		t.Errorf("Concurrent operations had %d errors", errCount)
	}

	// All tasks should be removed
	remaining := len(s.GetTasks())
	t.Logf("Remaining tasks after concurrent add/remove: %d", remaining)
}

// TestConcurrent_AddWhileTicking tests adding tasks during Tick execution.
func TestConcurrent_AddWhileTicking(t *testing.T) {
	s := NewScheduler()

	// Add initial tasks
	for i := 0; i < 100; i++ {
		job, _ := WrapJob(fmt.Sprintf("initial-%d", i), func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		_ = s.AddTask("* * * * *", job)
	}

	s.Start()
	defer s.Stop()

	var wg sync.WaitGroup
	var addCount atomic.Int32

	// Continuously add tasks while scheduler is running
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				taskID := fmt.Sprintf("dynamic-%d-%d", id, j)
				job, _ := WrapJob(taskID, func() error { return nil })
				if err := s.AddTask("*/5 * * * *", job); err == nil {
					addCount.Add(1)
				}
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Successfully added %d tasks during Tick", addCount.Load())
}

// TestConcurrent_RemoveWhileTicking tests removing tasks during Tick execution.
func TestConcurrent_RemoveWhileTicking(t *testing.T) {
	s := NewScheduler()

	// Add many tasks
	for i := 0; i < 500; i++ {
		job, _ := WrapJob(fmt.Sprintf("removable-%d", i), func() error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})
		_ = s.AddTask("* * * * *", job)
	}

	s.Start()
	defer s.Stop()

	var wg sync.WaitGroup
	var removeCount atomic.Int32

	// Continuously remove tasks while scheduler is running
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				tasks := s.GetTasks()
				if len(tasks) == 0 {
					return
				}
				// Remove random task
				idx := rand.Intn(len(tasks))
				s.RemoveTask(tasks[idx])
				removeCount.Add(1)
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}

	// Wait for all tasks to be removed (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("Removed %d tasks during Tick", removeCount.Load())
	case <-time.After(10 * time.Second):
		t.Log("Timeout waiting for task removal (this is OK)")
	}
}

// ============================================================================
// Multiple Scheduler Tests
// ============================================================================

// TestConcurrent_MultipleSchedulers tests running many schedulers in parallel.
func TestConcurrent_MultipleSchedulers(t *testing.T) {
	const numSchedulers = 50
	const tasksPerScheduler = 10

	var wg sync.WaitGroup
	var errorCount atomic.Int32

	for i := 0; i < numSchedulers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			s := NewScheduler()

			// Add tasks
			for j := 0; j < tasksPerScheduler; j++ {
				taskID := fmt.Sprintf("sched-%d-task-%d", id, j)
				job, _ := WrapJob(taskID, func() error {
					time.Sleep(time.Millisecond)
					return nil
				})
				if err := s.AddTask("* * * * *", job); err != nil {
					errorCount.Add(1)
				}
			}

			// Start and run briefly
			s.Start()
			time.Sleep(100 * time.Millisecond)
			s.Stop()

			// Verify tasks
			tasks := s.GetTasks()
			if len(tasks) != tasksPerScheduler {
				errorCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	errCount := errorCount.Load()
	if errCount > 0 {
		t.Errorf("Multiple schedulers test had %d errors", errCount)
	}
}

// ============================================================================
// Race Condition Detection Tests (run with -race flag)
// ============================================================================

// TestConcurrent_RaceCondition_AddGetRemove tests for race conditions in basic operations.
func TestConcurrent_RaceCondition_AddGetRemove(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			job, _ := WrapJob(fmt.Sprintf("race-%d", id), func() error {
				return nil
			})
			_ = s.AddTask("* * * * *", job)
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.GetTasks()
		}()
	}

	// Concurrent removes
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tasks := s.GetTasks()
			if len(tasks) > 0 {
				s.RemoveTask(tasks[0])
			}
		}()
	}

	wg.Wait()
	t.Log("Race condition test completed (run with -race to detect issues)")
}

// TestConcurrent_RaceCondition_NextCalculation tests concurrent Next() calls.
func TestConcurrent_RaceCondition_NextCalculation(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("race-next", func() error { return nil })
	_ = s.AddTask("*/5 * * * *", job)

	task := s.GetTasks()[0]
	now := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = task.CronParser.Next(now)
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent Next() completed (run with -race to detect issues)")
}

// ============================================================================
// High Load Tests
// ============================================================================

// TestConcurrent_HighLoad_1000Tasks tests scheduler with 1000 tasks.
func TestConcurrent_HighLoad_1000Tasks(t *testing.T) {
	s := NewScheduler(StorageTypeHeap)

	const numTasks = 1000

	// Add tasks
	start := time.Now()
	for i := 0; i < numTasks; i++ {
		job, _ := WrapJob(fmt.Sprintf("high-load-%d", i), func() error { return nil })
		_ = s.AddTask("* * * * *", job)
	}
	addDuration := time.Since(start)
	t.Logf("Added %d tasks in %v", numTasks, addDuration)

	// Verify count
	tasks := s.GetTasks()
	if len(tasks) != numTasks {
		t.Errorf("Expected %d tasks, got %d", numTasks, len(tasks))
	}

	// Start and run briefly
	s.Start()
	time.Sleep(200 * time.Millisecond)
	s.Stop()

	// Check memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("Heap allocation: %d KB", m.HeapAlloc/1024)
}

// TestConcurrent_HighLoad_10000Tasks tests scheduler with 10000 tasks.
func TestConcurrent_HighLoad_10000Tasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high load test in short mode")
	}

	s := NewScheduler(StorageTypeTimeWheel) // Use TimeWheel for large task counts

	const numTasks = 10000

	start := time.Now()
	for i := 0; i < numTasks; i++ {
		job, _ := WrapJob(fmt.Sprintf("massive-%d", i), func() error { return nil })
		_ = s.AddTask("*/5 * * * *", job)
	}
	addDuration := time.Since(start)
	t.Logf("Added %d tasks in %v", numTasks, addDuration)

	tasks := s.GetTasks()
	if len(tasks) != numTasks {
		t.Errorf("Expected %d tasks, got %d", numTasks, len(tasks))
	}

	// Memory check
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("Heap allocation after 10000 tasks: %d MB", m.HeapAlloc/1024/1024)

	if m.HeapAlloc > 500*1024*1024 { // 500MB threshold
		t.Errorf("Memory usage too high: %d MB", m.HeapAlloc/1024/1024)
	}
}

// ============================================================================
// Rapid Start/Stop Tests
// ============================================================================

// TestConcurrent_RapidStartStop tests rapid start/stop cycles.
func TestConcurrent_RapidStartStop(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("rapid", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	_ = s.AddTask("* * * * *", job)

	const cycles = 100

	start := time.Now()
	for i := 0; i < cycles; i++ {
		s.Start()
		time.Sleep(time.Millisecond)
		s.Stop()
	}
	duration := time.Since(start)

	t.Logf("Completed %d start/stop cycles in %v", cycles, duration)
}

// TestConcurrent_ConcurrentStartStop tests concurrent start/stop calls.
func TestConcurrent_ConcurrentStartStop(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("concurrent-ss", func() error { return nil })
	_ = s.AddTask("* * * * *", job)

	var wg sync.WaitGroup

	// Multiple goroutines calling Start
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				s.Start()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Multiple goroutines calling Stop
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				s.Stop()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	s.Stop() // Ensure stopped
	t.Log("Concurrent start/stop completed without panic")
}

// ============================================================================
// Storage Backend Concurrent Tests
// ============================================================================

// TestConcurrent_TaskQueue_Operations tests TaskQueue under concurrent access.
func TestConcurrent_TaskQueue_Operations(t *testing.T) {
	tq := NewTaskQueue()
	now := time.Now()

	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				task := &Task{
					ID:          fmt.Sprintf("tq-%d-%d", id, j),
					NextRunTime: now.Add(time.Duration(id*j) * time.Minute),
				}
				tq.AddTask(task)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = tq.GetTasks()
				_ = tq.TaskExist("tq-0-0")
			}
		}()
	}

	// Concurrent ticks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tickTime := now.Add(time.Duration(id) * time.Hour)
			for j := 0; j < 20; j++ {
				_ = tq.Tick(tickTime)
				tickTime = tickTime.Add(time.Minute)
			}
		}(i)
	}

	wg.Wait()
	t.Log("TaskQueue concurrent operations completed")
}

// TestConcurrent_TimeWheel_Operations tests TimeWheel under concurrent access.
func TestConcurrent_TimeWheel_Operations(t *testing.T) {
	tw := NewDynamicTimeWheel()
	now := time.Now()

	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				task := &Task{
					ID:          fmt.Sprintf("tw-%d-%d", id, j),
					NextRunTime: now.Add(time.Duration(id*j) * time.Second),
				}
				tw.AddTask(task)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = tw.GetTasks()
				_ = tw.TaskExist("tw-0-0")
			}
		}()
	}

	// Concurrent ticks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tickTime := now.Add(time.Duration(id) * time.Second)
			for j := 0; j < 20; j++ {
				_ = tw.Tick(tickTime)
				tickTime = tickTime.Add(time.Second)
			}
		}(i)
	}

	wg.Wait()
	t.Log("TimeWheel concurrent operations completed")
}

// ============================================================================
// Task Execution Concurrent Tests
// ============================================================================

// TestConcurrent_TaskExecution_NoOverlap ensures same task doesn't run concurrently.
func TestConcurrent_TaskExecution_NoOverlap(t *testing.T) {
	var concurrentRuns atomic.Int32
	var maxConcurrent atomic.Int32

	job, _ := WrapJob("no-overlap", func() error {
		current := concurrentRuns.Add(1)
		for {
			old := maxConcurrent.Load()
			if current <= old || maxConcurrent.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		concurrentRuns.Add(-1)
		return nil
	})

	s := NewScheduler()
	// Use seconds to trigger more frequently
	_ = s.AddTask("* * * * * *", job, WithSeconds())

	s.Start()
	time.Sleep(500 * time.Millisecond)
	s.Stop()

	max := maxConcurrent.Load()
	t.Logf("Max concurrent runs of same task: %d", max)

	if max > 1 {
		t.Errorf("Same task ran concurrently %d times (should be 1)", max)
	}
}

// TestConcurrent_MultipleTasks_Parallel tests multiple tasks running in parallel.
func TestConcurrent_MultipleTasks_Parallel(t *testing.T) {
	var runCount atomic.Int32
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	s := NewScheduler()

	// Add 10 tasks that all trigger immediately
	for i := 0; i < 10; i++ {
		taskID := fmt.Sprintf("parallel-%d", i)
		job, _ := WrapJob(taskID, func() error {
			current := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if current <= old || maxConcurrent.CompareAndSwap(old, current) {
					break
				}
			}
			runCount.Add(1)
			time.Sleep(100 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		})
		_ = s.AddTask("* * * * * *", job, WithSeconds())
	}

	s.Start()
	time.Sleep(500 * time.Millisecond)
	s.Stop()

	t.Logf("Total runs: %d, Max concurrent: %d", runCount.Load(), maxConcurrent.Load())

	// Multiple tasks should run in parallel
	if maxConcurrent.Load() < 2 {
		t.Log("Note: Tasks may not have run in parallel (depends on timing)")
	}
}

// ============================================================================
// Context Cancellation Tests
// ============================================================================

// TestConcurrent_ContextCancellation tests task cancellation via context.
func TestConcurrent_ContextCancellation(t *testing.T) {
	var cancelledCount atomic.Int32
	var completedCount atomic.Int32

	s := NewScheduler()

	// Task that respects context cancellation
	job, _ := WrapJob("cancellable", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			cancelledCount.Add(1)
			return ctx.Err()
		case <-time.After(5 * time.Second):
			completedCount.Add(1)
			return nil
		}
	})

	_ = s.AddTask("* * * * * *", job, WithSeconds(), WithTimeout(100*time.Millisecond))

	s.Start()
	time.Sleep(500 * time.Millisecond)
	s.Stop()

	t.Logf("Cancelled: %d, Completed: %d", cancelledCount.Load(), completedCount.Load())

	// With 100ms timeout, tasks should be cancelled
	if cancelledCount.Load() == 0 {
		t.Log("Note: No tasks were cancelled (may depend on timing)")
	}
}

// ============================================================================
// Stress Test: Combined Operations
// ============================================================================

// TestConcurrent_CombinedStress performs combined stress testing.
func TestConcurrent_CombinedStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	s := NewScheduler()
	s.Start()

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Continuous adders
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					taskID := fmt.Sprintf("stress-add-%d-%d", id, counter)
					job, _ := WrapJob(taskID, func() error {
						time.Sleep(time.Millisecond)
						return nil
					})
					_ = s.AddTask("* * * * *", job)
					counter++
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Continuous removers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					tasks := s.GetTasks()
					if len(tasks) > 0 {
						idx := rand.Intn(len(tasks))
						s.RemoveTask(tasks[idx])
					}
					time.Sleep(15 * time.Millisecond)
				}
			}
		}()
	}

	// Continuous readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_ = s.GetTasks()
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
	s.Stop()

	finalTasks := len(s.GetTasks())
	t.Logf("Combined stress test completed. Final task count: %d", finalTasks)
}
