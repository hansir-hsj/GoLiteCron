package golitecron

import (
	"container/heap"
	"sync"
	"time"
)

type TaskQueue struct {
	tasks []*Task
	mu    sync.RWMutex
}

func NewTaskQueue() *TaskQueue {
	tq := &TaskQueue{}
	heap.Init(tq)
	return tq
}

func (tq *TaskQueue) Len() int {
	return len(tq.tasks)
}

func (tq *TaskQueue) Less(i, j int) bool {
	return tq.tasks[i].NextRunTime.Before(tq.tasks[j].NextRunTime)
}

func (tq *TaskQueue) Swap(i, j int) {
	tq.tasks[i], tq.tasks[j] = tq.tasks[j], tq.tasks[i]
}

func (tq *TaskQueue) Push(task any) {
	tq.tasks = append(tq.tasks, task.(*Task))
}

func (tq *TaskQueue) Pop() any {
	if len(tq.tasks) == 0 {
		return nil
	}
	task := tq.tasks[len(tq.tasks)-1]
	tq.tasks = tq.tasks[:len(tq.tasks)-1]
	return task
}

func (tq *TaskQueue) TaskExist(taskID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	for _, task := range tq.tasks {
		if task.ID == taskID {
			return true
		}
	}
	return false
}

func (tq *TaskQueue) AddTask(task *Task) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	heap.Push(tq, task)
}

func (tq *TaskQueue) GetTasks() []*Task {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	tasks := make([]*Task, len(tq.tasks))
	copy(tasks, tq.tasks)
	return tasks
}

func (tq *TaskQueue) RemoveTask(task *Task) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for i, t := range tq.tasks {
		if t.ID == task.ID {
			heap.Remove(tq, i)
			break
		}
	}
}

func (tq *TaskQueue) Tick() []*Task {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tasks := make([]*Task, 0)
	now := time.Now()
	if tq.Len() == 0 {
		return nil
	}

	toRemove := make([]int, 0)
	for i, t := range tq.tasks {
		if t.NextRunTime.After(now) {
			continue
		}
		tasks = append(tasks, t)
		toRemove = append(toRemove, i)
	}

	for i := len(toRemove) - 1; i >= 0; i-- {
		heap.Remove(tq, toRemove[i])
	}

	return tasks
}
