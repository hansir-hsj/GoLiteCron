package golitecron

import (
	"testing"
	"time"
)

// TestTimeWheel_LevelExpansion tests that levels expand for long-duration tasks
func TestTimeWheel_LevelExpansion(t *testing.T) {
	tw := NewDynamicTimeWheel()

	// Initial coverage should be 60 seconds (1 level * 60 slots * 1 second)
	initialCoverage := tw.maxCoverage()
	expectedInitial := 60 * time.Second
	if initialCoverage != expectedInitial {
		t.Errorf("expected initial coverage %v, got %v", expectedInitial, initialCoverage)
	}

	// Add a task far in the future (2 hours)
	futureTask := &Task{
		ID:          "future-task",
		NextRunTime: time.Now().UTC().Add(2 * time.Hour),
	}
	tw.AddTask(futureTask)

	// Coverage should have expanded
	newCoverage := tw.maxCoverage()
	if newCoverage <= initialCoverage {
		t.Errorf("expected coverage to expand, initial=%v, new=%v", initialCoverage, newCoverage)
	}

	// Task should exist
	if !tw.TaskExist("future-task") {
		t.Error("expected future-task to exist after add")
	}
}

// TestTimeWheel_MultipleExpansions tests multiple level expansions
func TestTimeWheel_MultipleExpansions(t *testing.T) {
	tw := NewDynamicTimeWheel()

	// Add tasks at increasing distances
	durations := []time.Duration{
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour, // 1 week
	}

	now := time.Now().UTC()
	for i, d := range durations {
		task := &Task{
			ID:          "task-" + d.String(),
			NextRunTime: now.Add(d),
		}
		tw.AddTask(task)

		if !tw.TaskExist(task.ID) {
			t.Errorf("task %d (%s) should exist after add", i, task.ID)
		}
	}

	// Get all tasks
	tasks := tw.GetTasks()
	if len(tasks) != len(durations) {
		t.Errorf("expected %d tasks, got %d", len(durations), len(tasks))
	}
}

// TestTimeWheel_TaskMigration tests tasks moving between levels during tick
func TestTimeWheel_TaskMigration(t *testing.T) {
	// Use a custom tick duration for faster testing
	tw := NewDynamicTimeWheel(50 * time.Millisecond)

	// Add a task in the past (should be immediately ready)
	task := &Task{
		ID:          "migrate-task",
		NextRunTime: time.Now().UTC().Add(-100 * time.Millisecond), // In the past
	}
	tw.AddTask(task)

	// Task should exist (even past tasks are added first)
	if !tw.TaskExist("migrate-task") {
		t.Fatal("expected task to exist after add")
	}

	// Tick should return the past task as ready
	readyTasks := tw.Tick(time.Now().UTC())

	// Task should be ready
	found := false
	for _, rt := range readyTasks {
		if rt.ID == "migrate-task" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected migrate-task to be ready after tick")
	}

	// Task should no longer exist in wheel
	if tw.TaskExist("migrate-task") {
		t.Error("expected task to be removed after being returned as ready")
	}
}

// TestTimeWheel_ConcurrentAddTask tests concurrent task additions
func TestTimeWheel_ConcurrentAddTask(t *testing.T) {
	tw := NewDynamicTimeWheel()

	done := make(chan bool)
	taskCount := 100

	// Concurrent adds
	for i := 0; i < taskCount; i++ {
		go func(id int) {
			task := &Task{
				ID:          "concurrent-" + string(rune('A'+id%26)) + string(rune('0'+id%10)),
				NextRunTime: time.Now().UTC().Add(time.Duration(id) * time.Second),
			}
			tw.AddTask(task)
			done <- true
		}(i)
	}

	// Wait for all adds to complete
	for i := 0; i < taskCount; i++ {
		<-done
	}

	// Should have all tasks (or close to it, allowing for ID collisions)
	tasks := tw.GetTasks()
	if len(tasks) < taskCount/2 {
		t.Errorf("expected at least %d tasks, got %d", taskCount/2, len(tasks))
	}
}

// TestTimeWheel_ConcurrentTickAndAdd tests concurrent tick and add operations
func TestTimeWheel_ConcurrentTickAndAdd(t *testing.T) {
	tw := NewDynamicTimeWheel(50 * time.Millisecond)

	done := make(chan bool)

	// Goroutine adding tasks
	go func() {
		for i := 0; i < 50; i++ {
			task := &Task{
				ID:          "add-" + string(rune('0'+i%10)),
				NextRunTime: time.Now().UTC().Add(time.Duration(i*10) * time.Millisecond),
			}
			tw.AddTask(task)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine ticking
	go func() {
		for i := 0; i < 50; i++ {
			tw.Tick(time.Now().UTC())
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// Should not have panicked
}

// TestTimeWheel_RemoveTask tests task removal
func TestTimeWheel_RemoveTask(t *testing.T) {
	tw := NewDynamicTimeWheel()

	task := &Task{
		ID:          "to-remove",
		NextRunTime: time.Now().UTC().Add(1 * time.Hour),
	}
	tw.AddTask(task)

	if !tw.TaskExist("to-remove") {
		t.Fatal("expected task to exist after add")
	}

	tw.RemoveTask(task)

	if tw.TaskExist("to-remove") {
		t.Error("expected task to not exist after remove")
	}
}

// TestTimeWheel_RemoveNonExistent tests removing non-existent task
func TestTimeWheel_RemoveNonExistent(t *testing.T) {
	tw := NewDynamicTimeWheel()

	task := &Task{
		ID:          "non-existent",
		NextRunTime: time.Now().UTC(),
	}

	// Should not panic
	tw.RemoveTask(task)
}

// TestTimeWheel_AddTaskUpdatesExisting tests that adding task with same ID updates it
func TestTimeWheel_AddTaskUpdatesExisting(t *testing.T) {
	tw := NewDynamicTimeWheel()

	now := time.Now().UTC()
	task1 := &Task{
		ID:          "update-task",
		NextRunTime: now.Add(1 * time.Hour),
	}
	tw.AddTask(task1)

	// Add same ID with different time
	task2 := &Task{
		ID:          "update-task",
		NextRunTime: now.Add(2 * time.Hour),
	}
	tw.AddTask(task2)

	// Should still only have one task
	tasks := tw.GetTasks()
	count := 0
	for _, task := range tasks {
		if task.ID == "update-task" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("expected 1 task with ID 'update-task', got %d", count)
	}
}

// TestTimeWheel_GetTasksReturnsCopy tests that GetTasks returns a copy
func TestTimeWheel_GetTasksReturnsCopy(t *testing.T) {
	tw := NewDynamicTimeWheel()

	task := &Task{
		ID:          "copy-test",
		NextRunTime: time.Now().UTC().Add(1 * time.Hour),
	}
	tw.AddTask(task)

	tasks1 := tw.GetTasks()
	tasks2 := tw.GetTasks()

	// Modifying one should not affect the other
	if len(tasks1) > 0 {
		tasks1[0] = nil
	}

	if len(tasks2) > 0 && tasks2[0] == nil {
		t.Error("expected GetTasks to return independent copies")
	}
}

// TestTimeWheel_TickWithNoTasks tests tick on empty wheel
func TestTimeWheel_TickWithNoTasks(t *testing.T) {
	tw := NewDynamicTimeWheel()

	ready := tw.Tick(time.Now().UTC())

	if len(ready) != 0 {
		t.Errorf("expected no ready tasks from empty wheel, got %d", len(ready))
	}
}

// TestTimeWheel_TickWithFutureTasks tests tick with only future tasks
func TestTimeWheel_TickWithFutureTasks(t *testing.T) {
	tw := NewDynamicTimeWheel()

	task := &Task{
		ID:          "future-only",
		NextRunTime: time.Now().UTC().Add(1 * time.Hour),
	}
	tw.AddTask(task)

	ready := tw.Tick(time.Now().UTC())

	if len(ready) != 0 {
		t.Errorf("expected no ready tasks for future-only, got %d", len(ready))
	}

	if !tw.TaskExist("future-only") {
		t.Error("expected future task to still exist")
	}
}

// TestTimeWheel_CustomTickDuration tests custom tick duration
func TestTimeWheel_CustomTickDuration(t *testing.T) {
	customTick := 500 * time.Millisecond
	tw := NewDynamicTimeWheel(customTick)

	if tw.baseTickDuration != customTick {
		t.Errorf("expected baseTickDuration %v, got %v", customTick, tw.baseTickDuration)
	}
}

// TestTimeWheel_TaskExistAfterMultipleTicks tests task existence across ticks
func TestTimeWheel_TaskExistAfterMultipleTicks(t *testing.T) {
	tw := NewDynamicTimeWheel(50 * time.Millisecond)

	task := &Task{
		ID:          "persist-task",
		NextRunTime: time.Now().UTC().Add(1 * time.Hour),
	}
	tw.AddTask(task)

	// Multiple ticks should not affect future task
	for i := 0; i < 10; i++ {
		tw.Tick(time.Now().UTC())
		time.Sleep(10 * time.Millisecond)
	}

	if !tw.TaskExist("persist-task") {
		t.Error("expected future task to persist across ticks")
	}
}

// TestTimeWheel_WithExpectedTasks tests preallocation with expected tasks option
func TestTimeWheel_WithExpectedTasks(t *testing.T) {
	tw := NewDynamicTimeWheel(WithExpectedTasks(1000))

	if tw.expectedTasks != 1000 {
		t.Errorf("expected expectedTasks to be 1000, got %d", tw.expectedTasks)
	}

	// Verify basic functionality still works
	now := time.Now().UTC()
	for i := 0; i < 100; i++ {
		task := &Task{
			ID:          "task-" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			NextRunTime: now.Add(time.Duration(i) * time.Second),
		}
		tw.AddTask(task)
	}

	tasks := tw.GetTasks()
	if len(tasks) < 50 {
		t.Errorf("expected at least 50 tasks, got %d", len(tasks))
	}
}

// TestTimeWheel_WithTickDuration tests tick duration option
func TestTimeWheel_WithTickDuration(t *testing.T) {
	customTick := 200 * time.Millisecond
	tw := NewDynamicTimeWheel(WithTickDuration(customTick))

	if tw.baseTickDuration != customTick {
		t.Errorf("expected baseTickDuration %v, got %v", customTick, tw.baseTickDuration)
	}
}

// TestTimeWheel_CombinedOptions tests multiple options together
func TestTimeWheel_CombinedOptions(t *testing.T) {
	customTick := 100 * time.Millisecond
	tw := NewDynamicTimeWheel(
		WithExpectedTasks(5000),
		WithTickDuration(customTick),
	)

	if tw.expectedTasks != 5000 {
		t.Errorf("expected expectedTasks to be 5000, got %d", tw.expectedTasks)
	}
	if tw.baseTickDuration != customTick {
		t.Errorf("expected baseTickDuration %v, got %v", customTick, tw.baseTickDuration)
	}
}

// TestTimeWheel_BackwardCompatibility tests backward compatibility with duration argument
func TestTimeWheel_BackwardCompatibility(t *testing.T) {
	// Old style: pass duration directly
	tw := NewDynamicTimeWheel(100 * time.Millisecond)

	if tw.baseTickDuration != 100*time.Millisecond {
		t.Errorf("expected baseTickDuration 100ms, got %v", tw.baseTickDuration)
	}

	// New style: use option function
	tw2 := NewDynamicTimeWheel(WithTickDuration(200 * time.Millisecond))

	if tw2.baseTickDuration != 200*time.Millisecond {
		t.Errorf("expected baseTickDuration 200ms, got %v", tw2.baseTickDuration)
	}
}

// TestTimeWheel_CalculateLevelCapacity tests capacity distribution across levels
func TestTimeWheel_CalculateLevelCapacity(t *testing.T) {
	tw := NewDynamicTimeWheel(WithExpectedTasks(1000))

	// Level 0 should get 70%
	level0Cap := tw.calculateLevelCapacity(0)
	if level0Cap != 700 {
		t.Errorf("expected level 0 capacity 700, got %d", level0Cap)
	}

	// Level 1 should get 20%
	level1Cap := tw.calculateLevelCapacity(1)
	if level1Cap != 200 {
		t.Errorf("expected level 1 capacity 200, got %d", level1Cap)
	}

	// Level 2+ should get 10%
	level2Cap := tw.calculateLevelCapacity(2)
	if level2Cap != 100 {
		t.Errorf("expected level 2 capacity 100, got %d", level2Cap)
	}
}

// TestTimeWheel_ZeroExpectedTasks tests behavior with zero expected tasks
func TestTimeWheel_ZeroExpectedTasks(t *testing.T) {
	tw := NewDynamicTimeWheel(WithExpectedTasks(0))

	// Should still work normally
	task := &Task{
		ID:          "zero-expected",
		NextRunTime: time.Now().UTC().Add(1 * time.Hour),
	}
	tw.AddTask(task)

	if !tw.TaskExist("zero-expected") {
		t.Error("expected task to exist")
	}
}
