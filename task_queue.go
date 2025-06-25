package golitecron

import (
	"container/heap"
	"time"
)

type TaskQueue []*Task

func NewTaskQueue() *TaskQueue {
	tq := &TaskQueue{}
	heap.Init(tq)
	return tq
}

func (tq TaskQueue) Len() int {
	return len(tq)
}

func (tq TaskQueue) Less(i, j int) bool {
	return tq[i].NextRunTime.Before(tq[j].NextRunTime)
}

func (tq TaskQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
}

func (tq *TaskQueue) Push(task any) {
	t := task.(*Task)
	*tq = append(*tq, t)
}

func (tq *TaskQueue) Pop() any {
	if len(*tq) == 0 {
		return nil
	}
	task := (*tq)[len(*tq)-1]
	*tq = (*tq)[:len(*tq)-1]
	return task
}

func (tq *TaskQueue) TaskExist(taskID string) bool {
	for _, task := range *tq {
		if task.ID == taskID {
			return true
		}
	}
	return false
}

func (tq *TaskQueue) AddTask(task *Task) {
	heap.Push(tq, task)
}

func (tq *TaskQueue) RemoveTask(task *Task) {
	for i, t := range *tq {
		if t.ID == task.ID {
			heap.Remove(tq, i)
			break
		}
	}
}

func (tq *TaskQueue) Tick() []*Task {
	tasks := make([]*Task, 0, tq.Len())
	now := time.Now()
	if tq.Len() == 0 {
		return nil
	}

	for _, t := range *tq {
		if t.NextRunTime.After(now) {
			continue
		}
		task := heap.Pop(tq).(*Task)
		tasks = append(tasks, task)
	}

	return tasks
}
