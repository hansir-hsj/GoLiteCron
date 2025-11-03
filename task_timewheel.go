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
		lastTickTime: time.Now().UTC(),
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
	now := time.Now().UTC()
	duration := max(task.NextRunTime.UTC().Sub(now), 0)

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

func (dtw *DynamicTimeWheel) Tick(now time.Time) []*Task {
	nowUTC := now.UTC()
	var readyTasks []*Task

	dtw.mu.RLock()
	levelCnt := len(dtw.levels)
	dtw.mu.RUnlock()

	for i := levelCnt - 1; i >= 0; i-- {
		var tasksToMove []*Task

		// step 1: collect tasks to move
		dtw.mu.RLock()
		level := dtw.levels[i]
		dtw.mu.RUnlock()

		level.mu.Lock()

		elapsed := nowUTC.Sub(level.lastTickTime)
		ticks := int(elapsed / level.tickDuration)

		if ticks > 0 {
			level.lastTickTime = level.lastTickTime.Add(time.Duration(ticks) * level.tickDuration)
			level.currentSlot = (level.currentSlot + ticks) % level.wheelSize

			slot := level.slots[level.currentSlot]
			for e := slot.Front(); e != nil; e = e.Next() {
				task := e.Value.(*Task)
				tasksToMove = append(tasksToMove, task)
				delete(level.tasks, task.ID)
			}
			// Clear the slot
			slot.Init()
		} else {
			// Even if there is no complete tick to proceed, check the tasks that have expired in the current slot
			slot := level.slots[level.currentSlot]
			var removeEls []*list.Element
			for e := slot.Front(); e != nil; e = e.Next() {
				task := e.Value.(*Task)
				if !task.NextRunTime.UTC().After(nowUTC) {
					tasksToMove = append(tasksToMove, task)
					removeEls = append(removeEls, e)
					delete(level.tasks, task.ID)
				}
			}
			for _, el := range removeEls {
				slot.Remove(el)
			}
		}

		// step 2: move tasks
		if len(tasksToMove) > 0 {
			if i == 0 {
				readyTasks = append(readyTasks, tasksToMove...)
			} else {
				dtw.mu.RLock()
				lowerLevel := dtw.levels[i-1]
				dtw.mu.RUnlock()

				for _, task := range tasksToMove {
					lowerLevel.addTask(task, nowUTC)
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

	nowUTC := now.UTC()
	taskTimeUTC := task.NextRunTime.UTC()
	offset := max(taskTimeUTC.Sub(nowUTC), 0)
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
