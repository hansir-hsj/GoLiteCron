package golitecron

import (
	"container/heap"
	"sync"
	"time"
)

type TaskQueue struct {
	tasks   []*Task
	taskIdx map[string]int // task ID -> index in heap
	mu      sync.RWMutex
}

func NewTaskQueue() *TaskQueue {
	tq := &TaskQueue{taskIdx: make(map[string]int)}
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
	tq.taskIdx[tq.tasks[i].ID] = i
	tq.taskIdx[tq.tasks[j].ID] = j
}

func (tq *TaskQueue) Push(task any) {
	t := task.(*Task)
	tq.taskIdx[t.ID] = len(tq.tasks)
	tq.tasks = append(tq.tasks, t)
}

func (tq *TaskQueue) Pop() any {
	if len(tq.tasks) == 0 {
		return nil
	}
	task := tq.tasks[len(tq.tasks)-1]
	tq.tasks = tq.tasks[:len(tq.tasks)-1]
	delete(tq.taskIdx, task.ID)
	return task
}

func (tq *TaskQueue) TaskExist(taskID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	_, exists := tq.taskIdx[taskID]
	return exists
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

	if idx, ok := tq.taskIdx[task.ID]; ok {
		heap.Remove(tq, idx)
	}
}

func (tq *TaskQueue) Tick(now time.Time) []*Task {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.Len() == 0 {
		return nil
	}

	nowUTC := now.UTC()
	tasks := make([]*Task, 0)

	for tq.Len() > 0 {
		top := tq.tasks[0]
		if top.NextRunTime.UTC().After(nowUTC) {
			break
		}
		tasks = append(tasks, heap.Pop(tq).(*Task))
	}

	return tasks
}
