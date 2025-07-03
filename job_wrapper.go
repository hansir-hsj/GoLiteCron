package golitecron

import "errors"

type FuncJob struct {
	id string
	fn func() error
}

func (fj *FuncJob) Execute() error {
	if fj.fn == nil {
		return errors.New("function is not set")
	}
	return fj.fn()
}

func (fj *FuncJob) ID() string {
	return fj.id
}

func WrapJob(id string, fn func() error) Job {
	return &FuncJob{
		id: id,
		fn: fn,
	}
}
