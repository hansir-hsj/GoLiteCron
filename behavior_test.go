package golitecron

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestParse_NthWeekday(t *testing.T) {
	p, err := Parse("0 0 * * MON#2", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	got := p.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	want := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
	for _, expr := range []string{"0 0 * * MON#6", "0 0 * * MON#0", "0 0 * * 8#1", "0 0 #1 * *"} {
		if _, err := Parse(expr); err == nil {
			t.Errorf("Parse(%q) should fail", expr)
		}
	}
}

func TestParse_StarStepDayUsesAnd(t *testing.T) {
	p, err := Parse("0 0 */2 * MON", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	// Jan 3 is an odd day but a Saturday; the first odd Monday is Jan 5.
	got := p.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	want := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
}

func TestScheduleBuilder_IntervalMustDivideCycle(t *testing.T) {
	s := NewScheduler()
	for _, err := range []error{
		s.Every(45).Seconds().Do(func() {}),
		s.Every(7).Minutes().Do(func() {}),
		s.Every(5).Hours().Do(func() {}),
	} {
		if err == nil {
			t.Error("expected an error for an interval that does not divide its cycle")
		}
	}
	if err := s.Every(30).Seconds().Do(func(context.Context) error { return nil }); err != nil {
		t.Errorf("context-aware job should be accepted: %v", err)
	}
}

func TestScheduler_RemoveRunningTaskFreesID(t *testing.T) {
	s := NewScheduler()
	if err := s.WithTickDuration(10 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	oldJob, _ := WrapJob("job", func() error {
		once.Do(func() { close(started) })
		<-release
		return nil
	})
	if err := s.AddTask("* * * * * *", oldJob, WithSeconds()); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	s.Start()
	defer s.Stop()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start")
	}
	if !s.RemoveTaskByID("job") {
		t.Fatal("RemoveTaskByID returned false")
	}
	newJob, _ := WrapJob("job", func() error { return nil })
	if err := s.AddTask("0 0 1 1 *", newJob); err != nil {
		t.Fatalf("re-adding the ID of a removed running task failed: %v", err)
	}
	close(release)
	time.Sleep(200 * time.Millisecond)

	if tasks := s.GetTasks(); len(tasks) != 1 || tasks[0].ID != "job" {
		t.Fatalf("expected only the new task, got %+v", tasks)
	}
}

func TestScheduler_RetryInterval(t *testing.T) {
	s := NewScheduler()
	if err := s.WithTickDuration(10 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var stamps []time.Time
	job, _ := WrapJob("retry", func() error {
		mu.Lock()
		stamps = append(stamps, time.Now())
		mu.Unlock()
		return errors.New("fail")
	})
	if err := s.AddTask("* * * * * *", job, WithSeconds(), WithRetry(2), WithRetryInterval(80*time.Millisecond)); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	s.Start()
	defer s.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(stamps)
		mu.Unlock()
		if n >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(stamps) < 3 {
		t.Fatalf("expected 3 attempts, got %d", len(stamps))
	}
	if gap := stamps[1].Sub(stamps[0]); gap < 70*time.Millisecond {
		t.Errorf("gap between attempts = %v, want at least the retry interval", gap)
	}
}

func TestLoadConfig_RejectsUnknownFieldsAndNegativeValues(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(yamlPath, []byte("tasks:\n  - id: a\n    timout: 1s\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromYAML(yamlPath); err == nil {
		t.Error("LoadFromYAML should reject unknown keys")
	}
	jsonPath := filepath.Join(dir, "c.json")
	if err := os.WriteFile(jsonPath, []byte(`{"tasks":[{"id":"a","timout":"1s"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromJSON(jsonPath); err == nil {
		t.Error("LoadFromJSON should reject unknown keys")
	}

	s := NewScheduler()
	if err := s.RegisterJob("f", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Tasks: []TaskConfig{{ID: "a", CronExpr: "* * * * *", FuncName: "f", Retry: -1}}}
	if err := s.LoadTasksFromConfig(cfg); err == nil {
		t.Error("negative retry should be rejected")
	}
}
