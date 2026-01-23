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

	// Use write lock for the entire operation to prevent race conditions
	dtw.mu.Lock()
	defer dtw.mu.Unlock()

	// Remove existing task with same ID from any level first
	for _, level := range dtw.levels {
		level.mu.Lock()
		if _, exists := level.tasks[task.ID]; exists {
			level.removeTask(task.ID)
		}
		level.mu.Unlock()
	}

	// Expand levels while holding the lock
	for dtw.maxCoverageLocked() < duration {
		dtw.expandLevelLocked()
	}

	// Find appropriate level and add task
	for _, level := range dtw.levels {
		levelCoverage := level.tickDuration * time.Duration(level.wheelSize)
		if duration <= levelCoverage {
			level.addTask(task)
			return
		}
	}
}

// maxCoverageLocked returns max coverage without acquiring lock.
// Caller must hold dtw.mu.
func (dtw *DynamicTimeWheel) maxCoverageLocked() time.Duration {
	max := time.Duration(0)
	for _, level := range dtw.levels {
		max += level.tickDuration * time.Duration(level.wheelSize)
	}
	return max
}

// expandLevelLocked expands levels without acquiring lock.
// Caller must hold dtw.mu.
func (dtw *DynamicTimeWheel) expandLevelLocked() {
	lastLevel := dtw.levels[len(dtw.levels)-1]
	newTick := lastLevel.tickDuration * time.Duration(lastLevel.wheelSize)
	newLevel := newLevelTimeWheel(newTick, DefaultWheelSize)
	dtw.levels = append(dtw.levels, newLevel)
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

			// Temporarily store remaining tasks to redistribute after unlocking
			// We cannot call redistributeRemainingTasks here because it might acquire dtw.mu or lowerLevel.mu
			// violating lock ordering (Level -> DTW -> LowerLevel vs DTW -> Level)
			expiredTasks = append(expiredTasks, remainingTasks...)
		}

		level.mu.Unlock()

		// Move all collected tasks (expired and remaining-but-needs-move) to appropriate level
		// Note: "expiredTasks" here effectively contains all tasks that were popped from the current slot.
		// Some are truly expired (ready to run), some are just cascading down (remaining).
		dtw.redistributeTasks(expiredTasks, i, &readyTasks)
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

// Helper method to redistribute tasks (both remaining and expired from higher levels)
// This method handles tasks that were popped from a slot and need to be placed back
// into the correct level (either current level or lower level).
//
// For levelIndex == 0:
// - Tasks that are expired (<= now) are ready to run.
// - Tasks that are not expired are re-added to level 0.
//
// For levelIndex > 0:
// - Tasks are moved to levelIndex - 1.
func (dtw *DynamicTimeWheel) redistributeTasks(tasks []*Task, levelIndex int, readyTasks *[]*Task) {
	if len(tasks) == 0 {
		return
	}

	now := time.Now().UTC()

	if levelIndex == 0 {
		dtw.mu.RLock()
		level0 := dtw.levels[0]
		dtw.mu.RUnlock()
		
		// For Level 0, we must split tasks: ready vs re-queue
		// We are NOT holding level0.mu here, so we can safely call level0.addTask which acquires the lock.
		for _, task := range tasks {
			if !task.NextRunTime.UTC().After(now) {
				*readyTasks = append(*readyTasks, task)
			} else {
				level0.addTask(task)
			}
		}
	} else {
		// For higher levels, push everything down to the next level
		dtw.mu.RLock()
		lowerLevel := dtw.levels[levelIndex-1]
		dtw.mu.RUnlock()

		for _, task := range tasks {
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

	ltw.addTaskLocked(task)
}

// addTaskLocked adds a task without acquiring the lock.
// The caller must hold ltw.mu before calling this method.
// This is used internally to avoid deadlock when redistributing tasks.
func (ltw *LevelTimeWheel) addTaskLocked(task *Task) {
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
