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

// unmarkRunning clears the slot only if it still holds t, so a finishing removed
// task cannot clear a newer task that reused its ID.
func (tl *taskLifecycle) unmarkRunning(t *task) {
	if current, ok := tl.running[t.ID]; ok && current == t {
		delete(tl.running, t.ID)
	}
}

func (tl *taskLifecycle) getRunning(taskID string) (*task, bool) {
	task, ok := tl.running[taskID]
	return task, ok
}
