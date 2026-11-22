package golitecron

type taskLifecycle struct {
	running map[string]*task
}

func newTaskLifecycle() *taskLifecycle {
	return &taskLifecycle{
		running: make(map[string]*task),
	}
}

func (tl *taskLifecycle) markRunning(task *task) {
	tl.running[task.ID] = task
}

func (tl *taskLifecycle) unmarkRunning(taskID string) {
	delete(tl.running, taskID)
}

func (tl *taskLifecycle) isRunning(taskID string) bool {
	_, ok := tl.running[taskID]
	return ok
}

func (tl *taskLifecycle) getRunning(taskID string) (*task, bool) {
	task, ok := tl.running[taskID]
	return task, ok
}
