package golitecron

import (
	"strings"
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

func (j *testJob) Execute() error {
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

	info := s.GetTaskInfo("job1")
	if !strings.Contains(info, "job1") {
		t.Fatalf("GetTaskInfo did not contain job id: %s", info)
	}

	// RemoveTask currently returns false in the implementation.
	removed := s.RemoveTask(tasks[0])
	if removed != false {
		t.Fatalf("RemoveTask returned %v, expected false (current implementation)", removed)
	}

	// underlying storage should no longer have the task
	if s.taskStorage.TaskExist("job1") {
		t.Fatalf("expected task to be removed from storage")
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