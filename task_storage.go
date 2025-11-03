package golitecron

import "time"

type TaskStorage interface {
	TaskExist(taskID string) bool
	AddTask(task *Task)
	RemoveTask(task *Task)
	Tick(now time.Time) []*Task
	GetTasks() []*Task
}
