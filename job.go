package golitecron

import "context"

type Job interface {
	Execute(ctx context.Context) error
	ID() string
}
