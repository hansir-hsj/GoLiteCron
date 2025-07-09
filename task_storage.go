package golitecron

type TaskStorage interface {
	TaskExist(taskID string) bool
	AddTask(task *Task)
	RemoveTask(task *Task)
	Tick() []*Task
	GetTasks() []*Task
}
