package golitecron

import (
	"reflect"
	"testing"
	"time"
)

// helpers for tests
func makeTask(now time.Time, id string, offset time.Duration) *task {
	return &task{
		ID:          id,
		NextRunTime: now.Add(offset),
	}
}

func hasTask(tasks []*task, id string) bool {
	for _, t := range tasks {
		if t.ID == id {
			return true
		}
	}
	return false
}

func hasTaskInfo(tasks []TaskInfo, id string) bool {
	for _, t := range tasks {
		if t.ID == id {
			return true
		}
	}
	return false
}

func TestInternalTaskQueue_AddAndExist(t *testing.T) {
	now := time.Now().UTC()
	tq := newTaskQueue()

	if tq.Len() != 0 {
		t.Fatalf("expected empty queue, got len=%d", tq.Len())
	}

	t1 := makeTask(now, "t1", -time.Minute)
	tq.AddTask(t1)

	if !tq.TaskExist("t1") {
		t.Fatalf("expected task t1 to exist after AddTask")
	}

	tasks := tq.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task from GetTasks, got %d", len(tasks))
	}
	if tasks[0].ID != "t1" {
		t.Fatalf("expected task ID 't1', got %s", tasks[0].ID)
	}
}

func TestGetTasks_ReturnsIndependentTaskSnapshots(t *testing.T) {
	now := time.Now().UTC()
	tq := newTaskQueue()

	t1 := makeTask(now, "a", -time.Minute)
	t2 := makeTask(now, "b", time.Minute)
	tq.AddTask(t1)
	tq.AddTask(t2)

	got := tq.GetTasks()
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}

	got = append(got, TaskInfo{ID: "c"})
	if len(got) == len(tq.GetTasks()) {
		t.Fatalf("expected returned slice append not to change underlying storage length")
	}

	got[0].ID = "mutated"
	got[0].NextRunTime = now.Add(24 * time.Hour)

	if !tq.TaskExist("a") || !tq.TaskExist("b") {
		t.Fatalf("underlying queue lost tasks after modifying returned task snapshot")
	}

	due := tq.Tick(now)
	if !hasTask(due, "a") {
		t.Fatalf("expected original task a to remain due after mutating returned snapshot")
	}
}

func TestGetTasks_ReturnsSortedSnapshots(t *testing.T) {
	now := time.Now().UTC()
	tq := newTaskQueue()

	tq.AddTask(makeTask(now, "later", 3*time.Minute))
	tq.AddTask(makeTask(now, "same-b", time.Minute))
	tq.AddTask(makeTask(now, "same-a", time.Minute))
	tq.AddTask(makeTask(now, "earlier", -time.Minute))

	got := tq.GetTasks()
	ids := make([]string, 0, len(got))
	for _, task := range got {
		ids = append(ids, task.ID)
	}

	want := []string{"earlier", "same-a", "same-b", "later"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("expected sorted task IDs %v, got %v", want, ids)
	}
}

func TestRemoveTask(t *testing.T) {
	now := time.Now().UTC()
	tq := newTaskQueue()

	t1 := makeTask(now, "r1", -time.Minute)
	t2 := makeTask(now, "r2", time.Minute)
	tq.AddTask(t1)
	tq.AddTask(t2)

	if !tq.TaskExist("r1") || !tq.TaskExist("r2") {
		t.Fatalf("expected both tasks to exist after add")
	}

	tq.RemoveTaskByID(t1.ID)

	if tq.TaskExist("r1") {
		t.Fatalf("expected r1 to be removed")
	}
	if !tq.TaskExist("r2") {
		t.Fatalf("expected r2 to still exist")
	}
}

func TestTick_ReturnsDueTasksAndRemovesThem(t *testing.T) {
	now := time.Now().UTC()
	tq := newTaskQueue()

	past1 := makeTask(now, "past1", -2*time.Second)
	past2 := makeTask(now, "past2", -time.Second)
	future := makeTask(now, "future", time.Hour)

	tq.AddTask(past1)
	tq.AddTask(past2)
	tq.AddTask(future)

	due := tq.Tick(now)
	if len(due) == 0 {
		t.Fatalf("expected at least one due task, got 0")
	}

	if !hasTask(due, "past1") || !hasTask(due, "past2") {
		t.Fatalf("expected past1 and past2 to be returned by Tick, got %v", due)
	}
	if hasTask(due, "future") {
		t.Fatalf("did not expect future task in due list")
	}

	// ensure due tasks removed from queue, future remains
	if tq.TaskExist("past1") || tq.TaskExist("past2") {
		t.Fatalf("expected past tasks to be removed after Tick")
	}
	if !tq.TaskExist("future") {
		t.Fatalf("expected future task to remain after Tick")
	}
}
