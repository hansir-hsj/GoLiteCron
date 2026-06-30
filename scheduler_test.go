package golitecron

import (
	"context"
	"testing"
	"time"
)

// testJob implements the Job interface used by Scheduler.
type testJob struct {
	id    string
	runCh chan struct{}
}

func (j *testJob) ID() string {
	return j.id
}

func (j *testJob) Execute(ctx context.Context) error {
	// non-blocking send to avoid hanging the test
	select {
	case j.runCh <- struct{}{}:
	default:
	}
	return nil
}

// helper to wait for a run signal with timeout
func waitForRun(ch <-chan struct{}, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ch:
		return true
	case <-timer.C:
		return false
	}
}

func TestNewScheduler_AddGetRemove(t *testing.T) {
	s := NewScheduler()
	if s == nil {
		t.Fatalf("expected scheduler, got nil")
	}

	runCh := make(chan struct{}, 1)
	job := &testJob{id: "job1", runCh: runCh}

	// add task that runs every second (seconds enabled)
	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	tasks := s.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "job1" {
		t.Fatalf("expected task ID job1, got %s", tasks[0].ID)
	}

	info, ok := s.GetTask("job1")
	if !ok {
		t.Fatal("expected GetTask to find job1")
	}
	if info.ID != "job1" {
		t.Fatalf("expected task ID job1, got %s", info.ID)
	}

	removed := s.RemoveTaskByID(tasks[0].ID)
	if removed != true {
		t.Fatalf("RemoveTaskByID returned %v, expected true", removed)
	}

	// underlying storage should no longer have the task
	if s.taskStorage.TaskExist("job1") {
		t.Fatalf("expected task to be removed from storage")
	}
}

func TestScheduler_GetTasksReturnsReadOnlySnapshots(t *testing.T) {
	s := NewScheduler()
	job := &testJob{id: "snapshot-job", runCh: make(chan struct{}, 1)}

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	tasks := s.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	tasks[0].ID = "mutated"
	tasks[0].NextRunTime = time.Now().UTC().Add(24 * time.Hour)

	if !s.RemoveTaskByID("snapshot-job") {
		t.Fatal("expected original task ID to remain removable after mutating returned task snapshot")
	}
	if s.taskStorage.TaskExist("snapshot-job") {
		t.Fatal("expected original task to be removed")
	}
}

func TestScheduler_AddTaskNilJobReturnsError(t *testing.T) {
	s := NewScheduler()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AddTask should return an error for nil job, not panic: %v", r)
		}
	}()

	if err := s.AddTask("* * * * *", nil); err == nil {
		t.Fatal("expected error for nil job")
	}
}

func TestScheduler_AddTaskWithNilLocationUsesLocal(t *testing.T) {
	s := NewScheduler()
	job := &testJob{id: "nil-location", runCh: make(chan struct{}, 1)}

	if err := s.AddTask("* * * * *", job, WithLocation(nil)); err != nil {
		t.Fatalf("AddTask should accept nil location by falling back to local: %v", err)
	}

	tasks := s.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	if tasks[0].Location != time.Local {
		t.Fatalf("expected local location, got %v", tasks[0].Location)
	}
}

func TestScheduler_StartRunsTask(t *testing.T) {
	s := NewScheduler()
	runCh := make(chan struct{}, 4)
	job := &testJob{id: "job2", runCh: runCh}

	if err := s.AddTask("*/1 * * * * *", job, WithSeconds(), WithLocation(time.UTC)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// start scheduler
	s.Start()

	// wait up to 3 seconds for the job to be executed at least once
	ok := waitForRun(runCh, 3*time.Second)

	// stop scheduler
	s.Stop()

	if !ok {
		t.Fatalf("expected job to be executed at least once within timeout")
	}
}
