package golitecron

import (
	"testing"
	"time"
)

func TestNewDynamicTimeWheel_AddAndExist(t *testing.T) {
	now := time.Now()
	tw := NewDynamicTimeWheel()

	if tw == nil {
		t.Fatalf("expected NewDynamicTimeWheel to return non-nil")
	}

	if tw.TaskExist("nope") {
		t.Fatalf("expected no tasks initially")
	}

	t1 := makeTask(now, "tw1", -time.Minute)
	tw.AddTask(t1)

	if !tw.TaskExist("tw1") {
		t.Fatalf("expected task 'tw1' to exist after AddTask")
	}

	tasks := tw.GetTasks()
	if len(tasks) == 0 {
		t.Fatalf("expected GetTasks to return at least one task")
	}
	if tasks[0].ID != "tw1" && !hasTask(tasks, "tw1") {
		t.Fatalf("expected returned tasks to include 'tw1'")
	}
}

func TestTimeWheel_Tick_ReturnsDueTasksAndRemovesThem(t *testing.T) {
	now := time.Now()
	tw := NewDynamicTimeWheel()

	past1 := makeTask(now, "past1", -2*time.Second)
	past2 := makeTask(now, "past2", -time.Second)
	future := makeTask(now, "future", time.Hour)

	tw.AddTask(past1)
	tw.AddTask(past2)
	tw.AddTask(future)

	due := tw.Tick()
	if len(due) == 0 {
		t.Fatalf("expected due tasks from Tick, got 0")
	}

	if !hasTask(due, "past1") || !hasTask(due, "past2") {
		t.Fatalf("expected past1 and past2 to be returned by Tick, got %+v", due)
	}
	if hasTask(due, "future") {
		t.Fatalf("did not expect future task in due list")
	}

	// ensure due tasks removed from wheel, future remains
	if tw.TaskExist("past1") || tw.TaskExist("past2") {
		t.Fatalf("expected past tasks to be removed after Tick")
	}
	if !tw.TaskExist("future") {
		t.Fatalf("expected future task to remain after Tick")
	}
}

func TestTimeWheel_RemoveTaskAndGetTasksCopy(t *testing.T) {
	now := time.Now()
	tw := NewDynamicTimeWheel()

	a := makeTask(now, "a", -time.Minute)
	b := makeTask(now, "b", time.Minute)
	tw.AddTask(a)
	tw.AddTask(b)

	if !tw.TaskExist("a") || !tw.TaskExist("b") {
		t.Fatalf("expected both tasks to exist after add")
	}

	// remove a
	tw.RemoveTask(a)
	if tw.TaskExist("a") {
		t.Fatalf("expected 'a' to be removed")
	}
	if !tw.TaskExist("b") {
		t.Fatalf("expected 'b' to still exist")
	}

	// GetTasks should return a copy (mutating returned slice shouldn't affect storage)
	got := tw.GetTasks()
	got = append(got, makeTask(now, "c", time.Hour))
	if len(got) == len(tw.GetTasks()) {
		t.Fatalf("expected returned slice append not to change underlying storage length")
	}
	// underlying still has 'b'
	if !tw.TaskExist("b") {
		t.Fatalf("underlying wheel lost task 'b' after mutating returned slice")
	}
}
