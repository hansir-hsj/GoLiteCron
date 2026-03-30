package golitecron

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWrapJob(t *testing.T) {
	// Test case 1: Function without context
	executed1 := false
	jobFn1 := func() error {
		executed1 = true
		return nil
	}
	job1, _ := WrapJob("test-id-1", jobFn1)

	if job1.ID() != "test-id-1" {
		t.Errorf("expected ID 'test-id-1', got '%s'", job1.ID())
	}

	if err := job1.Execute(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !executed1 {
		t.Error("job function 1 was not executed")
	}

	// Test case 2: Function with context
	executed2 := false
	jobFn2 := func(ctx context.Context) error {
		executed2 = true
		return nil
	}
	job2, _ := WrapJob("test-id-2", jobFn2)

	if job2.ID() != "test-id-2" {
		t.Errorf("expected ID 'test-id-2', got '%s'", job2.ID())
	}

	if err := job2.Execute(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !executed2 {
		t.Error("job function 2 was not executed")
	}
}

func TestJob_Cancellation(t *testing.T) {
	jobFn := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}

	job, _ := WrapJob("cancel-test", jobFn)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := job.Execute(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}
