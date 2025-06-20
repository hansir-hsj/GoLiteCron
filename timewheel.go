package golitecron

import (
	"container/list"
	"time"
)

type TimeWheel struct {
	tickDuration time.Duration
	wheelSize    int
	slots        []*list.List
	currentTick  int

	// taskId => slotIndex
	tasks map[string]int
}

func NewTimeWheel(tickDuration time.Duration, wheelSize int) *TimeWheel {
	slots := make([]*list.List, wheelSize)
	for i := 0; i < wheelSize; i++ {
		slots[i] = list.New()
	}

	return &TimeWheel{
		tickDuration: tickDuration,
		wheelSize:    wheelSize,
		slots:        slots,
		currentTick:  0,
		tasks:        make(map[string]int),
	}
}

func (tw *TimeWheel) TaskExist(taskID string) bool {
	_, exists := tw.tasks[taskID]
	return exists
}

func (tw *TimeWheel) AddTask(task *Task) {
	nextRunTime := task.NextRunTime
	delay := time.Until(nextRunTime)
	tw.scheduleTask(task, delay)
}

func (tw *TimeWheel) RemoveTask(task *Task) {
	slotIndex := tw.tasks[task.ID]
	tw.slots[slotIndex].Remove(&list.Element{Value: task})
}

func (tw *TimeWheel) scheduleTask(task *Task, delay time.Duration) {
	ticks := int(delay / tw.tickDuration)
	if ticks < 1 {
		ticks = 1
	}

	slotIndex := (tw.currentTick + ticks) % tw.wheelSize
	tw.slots[slotIndex].PushBack(task)
	tw.tasks[task.ID] = slotIndex
}
