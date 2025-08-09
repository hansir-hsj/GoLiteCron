package golitecron

import (
	"container/list"
	"sync"
	"time"
)

const (
	BaseTickDuration = time.Second
	DefaultWheelSize = 60
)

type entry struct {
	SlotIndex int
	Element   *list.Element
}

type LevelTimeWheel struct {
	tickDuration time.Duration
	wheelSize    int
	slots        []*list.List
	tasks        map[string]entry
	currentSlot  int
	lastTickTime time.Time
	mu           sync.RWMutex
}

type DynamicTimeWheel struct {
	baseTickDuration time.Duration
	levels           []*LevelTimeWheel
	mu               sync.RWMutex
}

func NewDynamicTimeWheel(baseTickDuration ...time.Duration) *DynamicTimeWheel {
	tickDuration := BaseTickDuration
	if len(baseTickDuration) > 0 {
		tickDuration = baseTickDuration[0]
	}
	return &DynamicTimeWheel{
		baseTickDuration: tickDuration,
		levels:           []*LevelTimeWheel{newLevelTimeWheel(tickDuration, DefaultWheelSize)},
	}
}

func newLevelTimeWheel(tick time.Duration, size int) *LevelTimeWheel {
	slots := make([]*list.List, size)
	for i := range slots {
		slots[i] = list.New()
	}
	return &LevelTimeWheel{
		tickDuration: tick,
		wheelSize:    size,
		slots:        slots,
		tasks:        make(map[string]entry),
		currentSlot:  0,
		lastTickTime: time.Now(),
	}
}

func (dtw *DynamicTimeWheel) maxCoverage() time.Duration {
	dtw.mu.RLock()
	defer dtw.mu.RUnlock()

	max := time.Duration(0)
	for _, level := range dtw.levels {
		max += level.tickDuration * time.Duration(level.wheelSize)
	}
	return max
}

func (dtw *DynamicTimeWheel) expandLevel() {
	dtw.mu.Lock()
	defer dtw.mu.Unlock()

	lastLevel := dtw.levels[len(dtw.levels)-1]
	newTick := lastLevel.tickDuration * time.Duration(lastLevel.wheelSize)
	newLevel := newLevelTimeWheel(newTick, DefaultWheelSize)
	dtw.levels = append(dtw.levels, newLevel)
}

func (dtw *DynamicTimeWheel) AddTask(task *Task) {
	now := time.Now()
	duration := max(task.NextRunTime.Sub(now), 0)

	for dtw.maxCoverage() < duration {
		dtw.expandLevel()
	}

	dtw.mu.RLock()
	levels := dtw.levels
	dtw.mu.RUnlock()

	for _, level := range levels {
		levelCoverage := level.tickDuration * time.Duration(level.wheelSize)
		if duration <= levelCoverage {
			level.addTask(task, now)
			return
		}
	}

	dtw.levels[len(dtw.levels)-1].addTask(task, now)
}

func (dtw *DynamicTimeWheel) Tick() []*Task {
	now := time.Now()
	var readyTasks []*Task

	dtw.mu.RLock()
	levels := dtw.levels
	dtw.mu.RUnlock()

	for i := len(levels) - 1; i >= 0; i-- {
		level := dtw.levels[i]
		level.mu.Lock()

		elapsed := now.Sub(level.lastTickTime)
		ticks := int(elapsed / level.tickDuration)

		if ticks > 0 {
			level.lastTickTime = level.lastTickTime.Add(time.Duration(ticks) * level.tickDuration)
			level.currentSlot = (level.currentSlot + ticks) % level.wheelSize

			slot := level.slots[level.currentSlot]
			if slot.Len() > 0 {
				tasks := make([]*Task, 0, slot.Len())
				for e := slot.Front(); e != nil; e = e.Next() {
					task := e.Value.(*Task)
					tasks = append(tasks, task)
					delete(level.tasks, task.ID)
				}
				// Clear the slot
				slot.Init()

				if i == 0 {
					readyTasks = append(readyTasks, tasks...)
				} else {
					level.mu.Unlock()
					for _, task := range tasks {
						dtw.levels[i-1].addTask(task, now)
					}
					continue
				}
			}
		}
		level.mu.Unlock()
	}

	return readyTasks
}

func (dtw *DynamicTimeWheel) TaskExist(taskID string) bool {
	dtw.mu.RLock()
	defer dtw.mu.RUnlock()

	var exists bool
	for _, level := range dtw.levels {
		level.mu.RLock()
		if _, ok := level.tasks[taskID]; ok {
			exists = true
		}
		level.mu.RUnlock()
		if exists {
			break
		}
	}
	return exists
}

func (dtw *DynamicTimeWheel) RemoveTask(task *Task) {
	dtw.mu.RLock()
	defer dtw.mu.RUnlock()

	for _, level := range dtw.levels {
		level.mu.Lock()
		if _, exists := level.tasks[task.ID]; exists {
			level.removeTask(task.ID)
			level.mu.Unlock()
			break
		}
		level.mu.Unlock()
	}
}

func (dtw *DynamicTimeWheel) GetTasks() []*Task {
	dtw.mu.RLock()
	defer dtw.mu.RUnlock()

	var tasks []*Task
	for _, level := range dtw.levels {
		level.mu.RLock()
		for _, slot := range level.slots {
			for e := slot.Front(); e != nil; e = e.Next() {
				tasks = append(tasks, e.Value.(*Task))
			}
		}
		level.mu.RUnlock()
	}
	return tasks
}

func (ltw *LevelTimeWheel) addTask(task *Task, now time.Time) {
	ltw.mu.Lock()
	defer ltw.mu.Unlock()

	offset := max(task.NextRunTime.Sub(now), 0)
	slotIndex := int(offset/ltw.tickDuration) % ltw.wheelSize

	if _, exists := ltw.tasks[task.ID]; exists {
		ltw.removeTask(task.ID)
	}

	e := ltw.slots[slotIndex].PushBack(task)
	ltw.tasks[task.ID] = entry{
		SlotIndex: slotIndex,
		Element:   e,
	}
}

func (ltw *LevelTimeWheel) removeTask(taskID string) {
	entry, ok := ltw.tasks[taskID]
	if !ok {
		return
	}
	ltw.slots[entry.SlotIndex].Remove(entry.Element)
	delete(ltw.tasks, taskID)
}
