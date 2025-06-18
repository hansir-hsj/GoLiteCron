package golitecron

type Job interface {
	Execute() error
	GetID() string
}
