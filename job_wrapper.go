package golitecron

import (
	"context"
	"errors"
)

type FuncJob struct {
	id string
	fn func(context.Context) error
}

func (fj *FuncJob) Execute(ctx context.Context) error {
	if fj.fn == nil {
		return errors.New("function is not set")
	}
	return fj.fn(ctx)
}

func (fj *FuncJob) ID() string {
	return fj.id
}

// WrapJob wraps a function as a Job.
// The function can either accept no arguments (func() error) or a context (func(context.Context) error).
// If the function does not accept a context, the context passed to Execute is ignored.
func WrapJob(id string, fn interface{}) Job {
	switch f := fn.(type) {
	case func() error:
		return &FuncJob{
			id: id,
			fn: func(_ context.Context) error { return f() },
		}
	case func(context.Context) error:
		return &FuncJob{
			id: id,
			fn: f,
		}
	default:
		// Panicking here might be too harsh for a library function without returning error,
		// but since WrapJob signature returns only Job, we'll return a job that errors.
		// Alternatively, we could change WrapJob to return (Job, error).
		// For now, let's assume usage correctness or return a dummy job that fails.
		return &FuncJob{
			id: id,
			fn: func(_ context.Context) error { return errors.New("invalid function signature") },
		}
	}
}
