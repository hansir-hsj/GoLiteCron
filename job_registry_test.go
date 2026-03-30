package golitecron

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegisterJob_Single(t *testing.T) {
	called := false
	RegisterJob("test-single-job", func() error {
		called = true
		return nil
	})

	fn, ok := GetJob("test-single-job")
	if !ok {
		t.Fatal("expected to find registered job")
	}

	if err := fn.(func() error)(); err != nil {
		t.Fatalf("job execution failed: %v", err)
	}

	if !called {
		t.Fatal("job function was not called")
	}
}

func TestRegisterJob_Multiple(t *testing.T) {
	RegisterJob("job-a", func() error { return nil })
	RegisterJob("job-b", func() error { return nil })
	RegisterJob("job-c", func() error { return nil })

	for _, name := range []string{"job-a", "job-b", "job-c"} {
		if _, ok := GetJob(name); !ok {
			t.Errorf("expected to find job %s", name)
		}
	}
}

func TestRegisterJob_Override(t *testing.T) {
	counter := int32(0)

	RegisterJob("override-job", func() error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	RegisterJob("override-job", func() error {
		atomic.AddInt32(&counter, 10)
		return nil
	})

	fn, ok := GetJob("override-job")
	if !ok {
		t.Fatal("expected to find registered job")
	}

	if err := fn.(func() error)(); err != nil {
		t.Fatalf("job execution failed: %v", err)
	}

	// Should use the second (overridden) function
	if atomic.LoadInt32(&counter) != 10 {
		t.Fatalf("expected counter to be 10 (from override), got %d", counter)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	_, ok := GetJob("non-existent-job-xyz")
	if ok {
		t.Fatal("expected not to find non-existent job")
	}
}

func TestFuncJob_Execute_NilFn(t *testing.T) {
	job := &FuncJob{id: "nil-fn-job", fn: nil}
	err := job.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error when fn is nil")
	}
}

func TestFuncJob_ID(t *testing.T) {
	job := &FuncJob{id: "test-id", fn: func(ctx context.Context) error { return nil }}
	if job.ID() != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", job.ID())
	}
}

// LoadTasksFromConfig tests

func TestLoadTasksFromConfig_Success(t *testing.T) {
	// Register job first
	executed := int32(0)
	RegisterJob("config-test-job", func() error {
		atomic.AddInt32(&executed, 1)
		return nil
	})

	content := `tasks:
  - id: "config-task"
    cron_expr: "*/1 * * * * *"
    func_name: "config-test-job"
    enable_seconds: true
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, err := LoadFromYaml(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromYaml failed: %v", err)
	}

	s := NewScheduler()
	if err := s.LoadTasksFromConfig(config); err != nil {
		t.Fatalf("LoadTasksFromConfig failed: %v", err)
	}

	tasks := s.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].ID != "config-task" {
		t.Errorf("expected task ID 'config-task', got '%s'", tasks[0].ID)
	}

	// Test execution
	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	if atomic.LoadInt32(&executed) == 0 {
		t.Fatal("expected task to execute at least once")
	}
}

func TestLoadTasksFromConfig_MissingID(t *testing.T) {
	RegisterJob("missing-id-job", func() error { return nil })

	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "", // Missing ID
				CronExpr: "* * * * *",
				FuncName: "missing-id-job",
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}

func TestLoadTasksFromConfig_MissingCronExpr(t *testing.T) {
	RegisterJob("missing-cron-job", func() error { return nil })

	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "task-no-cron",
				CronExpr: "", // Missing CronExpr
				FuncName: "missing-cron-job",
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err == nil {
		t.Fatal("expected error for missing CronExpr")
	}
}

func TestLoadTasksFromConfig_MissingFuncName(t *testing.T) {
	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "task-no-func",
				CronExpr: "* * * * *",
				FuncName: "", // Missing FuncName
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err == nil {
		t.Fatal("expected error for missing FuncName")
	}
}

func TestLoadTasksFromConfig_JobNotFound(t *testing.T) {
	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "task-unknown-func",
				CronExpr: "* * * * *",
				FuncName: "non-existent-function-xyz",
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err == nil {
		t.Fatal("expected error for non-existent job function")
	}
}

func TestLoadTasksFromConfig_InvalidCronExpr(t *testing.T) {
	RegisterJob("invalid-cron-job", func() error { return nil })

	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "task-invalid-cron",
				CronExpr: "invalid cron expression",
				FuncName: "invalid-cron-job",
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

func TestLoadTasksFromConfig_InvalidLocation(t *testing.T) {
	RegisterJob("invalid-loc-job", func() error { return nil })

	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "task-invalid-loc",
				CronExpr: "* * * * *",
				FuncName: "invalid-loc-job",
				Location: "Invalid/Timezone",
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err == nil {
		t.Fatal("expected error for invalid location")
	}
}

func TestLoadTasksFromConfig_WithAllOptions(t *testing.T) {
	RegisterJob("full-options-job", func() error { return nil })

	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:            "full-options-task",
				CronExpr:      "0 0 0 * * * 2027", // 7 fields: sec min hour dom month dow year
				FuncName:      "full-options-job",
				Timeout:       5000,
				Retry:         3,
				Location:      "UTC",
				EnableSeconds: true,
				EnableYears:   true,
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err != nil {
		t.Fatalf("LoadTasksFromConfig failed: %v", err)
	}

	tasks := s.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.CronParser.timeout != 5000*time.Millisecond {
		t.Errorf("expected timeout 5000ms, got %v", task.CronParser.timeout)
	}
	if task.CronParser.retry != 3 {
		t.Errorf("expected retry 3, got %d", task.CronParser.retry)
	}
}

func TestLoadTasksFromConfig_MultipleTasks(t *testing.T) {
	RegisterJob("multi-job-1", func() error { return nil })
	RegisterJob("multi-job-2", func() error { return nil })

	config := &Config{
		Tasks: []TaskConfig{
			{
				ID:       "multi-task-1",
				CronExpr: "* * * * *",
				FuncName: "multi-job-1",
			},
			{
				ID:       "multi-task-2",
				CronExpr: "0 * * * *",
				FuncName: "multi-job-2",
			},
		},
	}

	s := NewScheduler()
	err := s.LoadTasksFromConfig(config)
	if err != nil {
		t.Fatalf("LoadTasksFromConfig failed: %v", err)
	}

	tasks := s.GetTasks()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}
