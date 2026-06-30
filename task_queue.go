package golitecron

import (
	"container/heap"
	"sort"
	"sync"
	"time"
)

type taskQueue struct {
	tasks   []*task
	taskIdx map[string]int // task ID -> index in heap
	mu      sync.RWMutex
}

func newTaskQueue() *taskQueue {
	tq := &taskQueue{taskIdx: make(map[string]int)}
	heap.Init(tq)
	return tq
}

func (tq *taskQueue) Len() int {
	return len(tq.tasks)
}

func (tq *taskQueue) Less(i, j int) bool {
	return tq.tasks[i].NextRunTime.Before(tq.tasks[j].NextRunTime)
}

func (tq *taskQueue) Swap(i, j int) {
	tq.tasks[i], tq.tasks[j] = tq.tasks[j], tq.tasks[i]
	tq.taskIdx[tq.tasks[i].ID] = i
	tq.taskIdx[tq.tasks[j].ID] = j
}

func (tq *taskQueue) Push(value any) {
	t := value.(*task)
	tq.taskIdx[t.ID] = len(tq.tasks)
	tq.tasks = append(tq.tasks, t)
}

func (tq *taskQueue) Pop() any {
	if len(tq.tasks) == 0 {
		return nil
	}
	task := tq.tasks[len(tq.tasks)-1]
	tq.tasks = tq.tasks[:len(tq.tasks)-1]
	delete(tq.taskIdx, task.ID)
	return task
}

func (tq *taskQueue) TaskExist(taskID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	_, exists := tq.taskIdx[taskID]
	return exists
}

func (tq *taskQueue) AddTask(task *task) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	heap.Push(tq, task)
}

func (tq *taskQueue) GetTasks() []TaskInfo {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	tasks := make([]TaskInfo, 0, len(tq.tasks))
	for _, task := range tq.tasks {
		tasks = append(tasks, newTaskInfo(task))
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].NextRunTime.Equal(tasks[j].NextRunTime) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].NextRunTime.Before(tasks[j].NextRunTime)
	})
	return tasks
}

func (tq *taskQueue) RemoveTaskByID(taskID string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if idx, ok := tq.taskIdx[taskID]; ok {
		heap.Remove(tq, idx)
	}
}

func (tq *taskQueue) Tick(now time.Time) []*task {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.Len() == 0 {
		return nil
	}

	nowUTC := now.UTC()
	tasks := make([]*task, 0)

	for tq.Len() > 0 {
		top := tq.tasks[0]
		if top.NextRunTime.UTC().After(nowUTC) {
			break
		}
		tasks = append(tasks, heap.Pop(tq).(*task))
	}

	return tasks
}
