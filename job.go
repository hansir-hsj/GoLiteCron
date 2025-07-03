package golitecron

type Job interface {
	Execute() error
	ID() string
}
