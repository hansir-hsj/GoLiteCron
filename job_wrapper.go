package golitecron

import (
	"context"
	"fmt"
)

type FuncJob struct {
	id string
	fn func(context.Context) error
}

func (fj *FuncJob) Execute(ctx context.Context) error {
	if fj.fn == nil {
		return fmt.Errorf("function is not set")
	}
	return fj.fn(ctx)
}

func (fj *FuncJob) ID() string {
	return fj.id
}

// WrapJob wraps a function as a Job. Supports func() error or func(context.Context) error.
func WrapJob(id string, fn any) (Job, error) {
	switch f := fn.(type) {
	case func() error:
		return &FuncJob{
			id: id,
			fn: func(_ context.Context) error { return f() },
		}, nil
	case func(context.Context) error:
		return &FuncJob{
			id: id,
			fn: f,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported function signature: %T (expected func() error or func(context.Context) error)", fn)
	}
}
