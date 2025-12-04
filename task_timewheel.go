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
			level.addTask(task)
			return
		}
	}

}

func (dtw *DynamicTimeWheel) Tick(now time.Time) []*Task {
	nowUTC := now.UTC()
	var readyTasks []*Task

	dtw.mu.RLock()
	levelCnt := len(dtw.levels)
	dtw.mu.RUnlock()

	for i := levelCnt - 1; i >= 0; i-- {
		dtw.mu.RLock()
		level := dtw.levels[i]
		dtw.mu.RUnlock()

		level.mu.Lock()

		// Calculate ticks and update time/slot
		elapsed := nowUTC.Sub(level.lastTickTime)
		ticks := int(elapsed / level.tickDuration)

		// Collect expired tasks from current slot
		expiredTasks := level.collectExpiredTasksFromCurrentSlot(nowUTC)

		if ticks > 0 {
			level.lastTickTime = level.lastTickTime.Add(time.Duration(ticks) * level.tickDuration)
			level.currentSlot = (level.currentSlot + ticks) % level.wheelSize

			// Collect expired tasks from new slot and remaining tasks
			newExpiredTasks, remainingTasks := level.processSlotTick(nowUTC)
			expiredTasks = append(expiredTasks, newExpiredTasks...)

			// Redistribute remaining tasks
			level.redistributeRemainingTasks(remainingTasks, i, dtw)
		}

		level.mu.Unlock()

		// Move expired tasks to appropriate level
		dtw.moveExpiredTasks(expiredTasks, i, &readyTasks)
	}

	return readyTasks
}

// Helper method to collect expired tasks from current slot
func (ltw *LevelTimeWheel) collectExpiredTasksFromCurrentSlot(now time.Time) []*Task {
	var expiredTasks []*Task
	slot := ltw.slots[ltw.currentSlot]

	var elementsToRemove []*list.Element
	for e := slot.Front(); e != nil; e = e.Next() {
		task := e.Value.(*Task)
		if !task.NextRunTime.UTC().After(now) {
			expiredTasks = append(expiredTasks, task)
			elementsToRemove = append(elementsToRemove, e)
			delete(ltw.tasks, task.ID)
		}
	}

	for _, el := range elementsToRemove {
		slot.Remove(el)
	}

	return expiredTasks
}

// Helper method to process slot tick
func (ltw *LevelTimeWheel) processSlotTick(now time.Time) ([]*Task, []*Task) {
	var expiredTasks []*Task
	var remainingTasks []*Task

	slot := ltw.slots[ltw.currentSlot]

	// Collect expired tasks
	var elementsToRemove []*list.Element
	for e := slot.Front(); e != nil; e = e.Next() {
		task := e.Value.(*Task)
		if !task.NextRunTime.UTC().After(now) {
			expiredTasks = append(expiredTasks, task)
			elementsToRemove = append(elementsToRemove, e)
			delete(ltw.tasks, task.ID)
		}
	}

	// Remove expired tasks
	for _, el := range elementsToRemove {
		slot.Remove(el)
	}

	// Collect remaining tasks
	for e := slot.Front(); e != nil; e = e.Next() {
		task := e.Value.(*Task)
		remainingTasks = append(remainingTasks, task)
		delete(ltw.tasks, task.ID)
	}

	// Clear the slot
	slot.Init()

	return expiredTasks, remainingTasks
}

// Helper method to redistribute remaining tasks
func (ltw *LevelTimeWheel) redistributeRemainingTasks(remainingTasks []*Task, levelIndex int, dtw *DynamicTimeWheel) {
	for _, task := range remainingTasks {
		if levelIndex == 0 {
			// For level 0, re-add to the same level with updated timing
			ltw.addTask(task)
		} else {
			// For higher levels, move to lower level
			dtw.mu.RLock()
			lowerLevel := dtw.levels[levelIndex-1]
			dtw.mu.RUnlock()
			lowerLevel.addTask(task)
		}
	}
}

// Helper method to move expired tasks to appropriate level
func (dtw *DynamicTimeWheel) moveExpiredTasks(expiredTasks []*Task, levelIndex int, readyTasks *[]*Task) {
	if len(expiredTasks) == 0 {
		return
	}

	if levelIndex == 0 {
		*readyTasks = append(*readyTasks, expiredTasks...)
	} else {
		dtw.mu.RLock()
		lowerLevel := dtw.levels[levelIndex-1]
		dtw.mu.RUnlock()

		for _, task := range expiredTasks {
			lowerLevel.addTask(task)
		}
	}
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

func (ltw *LevelTimeWheel) addTask(task *Task) {
	ltw.mu.Lock()
	defer ltw.mu.Unlock()

	taskTimeUTC := task.NextRunTime.UTC()
	offset := max(taskTimeUTC.Sub(ltw.lastTickTime), 0)

	ticks := int((offset + ltw.tickDuration - 1) / ltw.tickDuration)
	slotIndex := (ltw.currentSlot + ticks) % ltw.wheelSize

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
