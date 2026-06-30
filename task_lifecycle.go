package golitecron

type taskLifecycle struct {
	removed map[string]struct{}
	running map[string]struct{}
}

func newTaskLifecycle() *taskLifecycle {
	return &taskLifecycle{
		removed: make(map[string]struct{}),
		running: make(map[string]struct{}),
	}
}

func (tl *taskLifecycle) markRunning(taskID string) {
	tl.running[taskID] = struct{}{}
}

func (tl *taskLifecycle) unmarkRunning(taskID string) {
	delete(tl.running, taskID)
}

func (tl *taskLifecycle) markRemoved(taskID string) {
	tl.removed[taskID] = struct{}{}
}

func (tl *taskLifecycle) clearRemoved(taskID string) {
	delete(tl.removed, taskID)
}

func (tl *taskLifecycle) isRunning(taskID string) bool {
	_, ok := tl.running[taskID]
	return ok
}

func (tl *taskLifecycle) isRemoved(taskID string) bool {
	_, ok := tl.removed[taskID]
	return ok
}
