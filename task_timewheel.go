package golitecron

import (
	"container/list"
	"time"
)

const (
	MinTimeWheelDuration = time.Minute
	DefaultWheelSize     = 60
)

type MultiLevelTimeWheel struct {
	timeWheels []*TaskTimeWheel
}

func NewMultiLevelTimeWheel() *MultiLevelTimeWheel {
	timeWheels := []*TaskTimeWheel{
		newTaskTimeWheel(time.Second, 60),
		newTaskTimeWheel(time.Minute, 60),
		newTaskTimeWheel(time.Hour, 24),
		newTaskTimeWheel(24*time.Hour, 365),
	}
	return &MultiLevelTimeWheel{
		timeWheels: timeWheels,
	}
}

func (mltw *MultiLevelTimeWheel) TaskExist(taskID string) bool {
	for _, tw := range mltw.timeWheels {
		if tw.taskExist(taskID) {
			return true
		}
	}
	return false
}

func (mltw *MultiLevelTimeWheel) AddTask(task *Task) {
	now := time.Now()
	duration := task.NextRunTime.Sub(now)
	switch {
	case duration < time.Second:
		mltw.timeWheels[0].addTask(task)
	case duration < time.Hour:
		mltw.timeWheels[1].addTask(task)
	case duration < 24*time.Hour:
		mltw.timeWheels[2].addTask(task)
	default:
		mltw.timeWheels[3].addTask(task)
	}
}

func (mltw *MultiLevelTimeWheel) RemoveTask(task *Task) {
	for _, tw := range mltw.timeWheels {
		if tw.taskExist(task.ID) {
			tw.removeTask(task)
			break
		}
	}
}

func (mltw *MultiLevelTimeWheel) Tick() []*Task {
	now := time.Now()
	var tasks []*Task
	for _, tw := range mltw.timeWheels {
		twTasks := tw.tick(now)
		tasks = append(tasks, twTasks...)
	}
	return tasks
}

type entry struct {
	SlotIndex int
	Element   *list.Element
}

type TaskTimeWheel struct {
	tickDuration time.Duration
	wheelSize    int
	slots        []*list.List
	preTickTime  time.Time

	tasks map[string]entry
}

func newTaskTimeWheel(tickDuration time.Duration, wheelSize int) *TaskTimeWheel {
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
		tasks:        make(map[string]entry),
	}
}

func (tw *TaskTimeWheel) taskExist(taskID string) bool {
	_, exists := tw.tasks[taskID]
	return exists
}

func (tw *TaskTimeWheel) addTask(task *Task) {
	nextRunTime := task.NextRunTime
	slotIndex := int(nextRunTime.Sub(tw.preTickTime)/tw.tickDuration) % tw.wheelSize

	// remove task if it already exists in the wheel
	if _, exist := tw.tasks[task.ID]; exist {
		tw.removeTask(task)
	}

	e := tw.slots[slotIndex].PushBack(task)
	tw.tasks[task.ID] = entry{
		SlotIndex: slotIndex,
		Element:   e,
	}
}

func (tw *TaskTimeWheel) removeTask(task *Task) {
	entry, ok := tw.tasks[task.ID]
	if !ok {
		return
	}
	if entry.Element == nil {
		return
	}
	tw.slots[entry.SlotIndex].Remove(entry.Element)
	delete(tw.tasks, task.ID)
}

func (tw *TaskTimeWheel) tick(now time.Time) []*Task {
	slotIndex := int(now.Sub(tw.preTickTime)/tw.tickDuration) % tw.wheelSize

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
