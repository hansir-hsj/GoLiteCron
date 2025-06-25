package golitecron

import (
	"container/list"
	"time"
)

const (
	MinTimeWheelDuration = time.Minute
	DefaultWheelSize     = 60
)

type TaskTimeWheel struct {
	tickDuration time.Duration
	wheelSize    int
	slots        []*list.List
	preTickTime  time.Time

	// taskId => slotIndex
	tasks map[string]int
}

func NewTaskTimeWheel(tickDuration time.Duration, wheelSize int) *TaskTimeWheel {
	if tickDuration <= 0 {
		tickDuration = MinTimeWheelDuration
	}
	if wheelSize <= 0 {
		wheelSize = DefaultWheelSize
	}

	slots := make([]*list.List, wheelSize)
	for i := range wheelSize {
		slots[i] = list.New()
	}

	return &TaskTimeWheel{
		tickDuration: tickDuration,
		wheelSize:    wheelSize,
		slots:        slots,
		preTickTime:  time.Now(),
		tasks:        make(map[string]int),
	}
}

func (tw *TaskTimeWheel) TaskExist(taskID string) bool {
	_, exists := tw.tasks[taskID]
	return exists
}

func (tw *TaskTimeWheel) AddTask(task *Task) {
	nextRunTime := task.NextRunTime

	minuteOfHour := nextRunTime.Minute()

	slotIndex := minuteOfHour % tw.wheelSize

	// remove task if it already exists in the wheel
	if _, exist := tw.tasks[task.ID]; exist {
		tw.RemoveTask(task)
	}

	tw.slots[slotIndex].PushBack(task)
	tw.tasks[task.ID] = slotIndex
}

func (tw *TaskTimeWheel) RemoveTask(task *Task) {
	slotIndex := tw.tasks[task.ID]
	for e := tw.slots[slotIndex].Front(); e != nil; e = e.Next() {
		if e.Value.(*Task).ID == task.ID {
			tw.slots[slotIndex].Remove(e)
			delete(tw.tasks, task.ID)
			break
		}
	}
}

func (tw *TaskTimeWheel) Tick() []*Task {
	now := time.Now()

	currentMinute := now.Minute()
	slotIndex := currentMinute % tw.wheelSize

	if tw.slots[slotIndex].Len() == 0 {
		return nil
	}

	tasks := make([]*Task, 0, tw.slots[slotIndex].Len())
	for e := tw.slots[slotIndex].Front(); e != nil; e = e.Next() {
		t := e.Value.(*Task)
		if t.NextRunTime.After(now) {
			continue
		}
		tasks = append(tasks, t)
		tw.slots[slotIndex].Remove(e)
		delete(tw.tasks, t.ID)
	}

	return tasks
}
