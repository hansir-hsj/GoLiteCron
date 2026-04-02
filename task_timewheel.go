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
	slots        []*list.List // Lazily initialized: nil until first task added to slot
	tasks        map[string]entry
	currentSlot  int
	lastTickTime time.Time
	mu           sync.RWMutex
}

type DynamicTimeWheel struct {
	baseTickDuration time.Duration
	expectedTasks    int // Expected number of tasks for map preallocation
	levels           []*LevelTimeWheel
	mu               sync.RWMutex
}

// TimeWheelOption is a function that configures a DynamicTimeWheel.
type TimeWheelOption func(*DynamicTimeWheel)

// WithExpectedTasks sets the expected number of tasks for map preallocation.
// This helps reduce map resizing overhead when adding many tasks.
// The capacity is distributed across levels, with lower levels getting more capacity.
func WithExpectedTasks(n int) TimeWheelOption {
	return func(dtw *DynamicTimeWheel) {
		if n > 0 {
			dtw.expectedTasks = n
		}
	}
}

// WithTickDuration sets the base tick duration for the time wheel.
// Default is 1 second.
func WithTickDuration(d time.Duration) TimeWheelOption {
	return func(dtw *DynamicTimeWheel) {
		if d > 0 {
			dtw.baseTickDuration = d
		}
	}
}

// NewDynamicTimeWheel creates a new dynamic time wheel.
// It accepts optional TimeWheelOption functions or a single time.Duration for backward compatibility.
func NewDynamicTimeWheel(args ...any) *DynamicTimeWheel {
	dtw := &DynamicTimeWheel{
		baseTickDuration: BaseTickDuration,
		expectedTasks:    0,
	}

	// Parse arguments for backward compatibility and new option pattern
	for _, arg := range args {
		switch v := arg.(type) {
		case time.Duration:
			dtw.baseTickDuration = v
		case TimeWheelOption:
			v(dtw)
		}
	}

	// Calculate initial map capacity for level 0
	// Level 0 typically holds most tasks (short-term tasks)
	initialCap := dtw.calculateLevelCapacity(0)
	dtw.levels = []*LevelTimeWheel{newLevelTimeWheel(dtw.baseTickDuration, DefaultWheelSize, initialCap)}

	return dtw
}

// calculateLevelCapacity returns the map preallocation capacity for a given level.
// Lower levels get more capacity as they hold more tasks.
// Level 0: 70% of expected tasks
// Level 1: 20% of expected tasks
// Level 2+: 10% of expected tasks (split among remaining levels)
func (dtw *DynamicTimeWheel) calculateLevelCapacity(level int) int {
	if dtw.expectedTasks <= 0 {
		return 0 // Let Go use default map capacity
	}

	switch level {
	case 0:
		return dtw.expectedTasks * 70 / 100
	case 1:
		return dtw.expectedTasks * 20 / 100
	default:
		return dtw.expectedTasks * 10 / 100
	}
}

// newLevelTimeWheel creates a new level time wheel.
// Slots are lazily initialized - only allocate the pointer array, not the lists.
// mapCapacity specifies the initial capacity for the tasks map (0 means use default).
func newLevelTimeWheel(tick time.Duration, size int, mapCapacity int) *LevelTimeWheel {
	var tasks map[string]entry
	if mapCapacity > 0 {
		tasks = make(map[string]entry, mapCapacity)
	} else {
		tasks = make(map[string]entry)
	}

	return &LevelTimeWheel{
		tickDuration: tick,
		wheelSize:    size,
		slots:        make([]*list.List, size), // Only allocate pointer array, slots are nil
		tasks:        tasks,
		currentSlot:  0,
		lastTickTime: time.Now().UTC(),
	}
}

// getOrCreateSlot returns the slot at the given index, creating it if necessary.
// Caller must hold ltw.mu (write lock) when calling this method.
func (ltw *LevelTimeWheel) getOrCreateSlot(index int) *list.List {
	if ltw.slots[index] == nil {
		ltw.slots[index] = list.New()
	}
	return ltw.slots[index]
}

// getSlot returns the slot at the given index, or nil if not initialized.
// Safe to call with read lock.
func (ltw *LevelTimeWheel) getSlot(index int) *list.List {
	return ltw.slots[index]
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
	newLevelIndex := len(dtw.levels)
	mapCapacity := dtw.calculateLevelCapacity(newLevelIndex)
	newLevel := newLevelTimeWheel(newTick, DefaultWheelSize, mapCapacity)
	dtw.levels = append(dtw.levels, newLevel)
}

func (dtw *DynamicTimeWheel) Tick(now time.Time) []*Task {
	nowUTC := now.UTC()
	var readyTasks []*Task

	// Take a consistent snapshot of levels under a single lock acquisition.
	// If expandLevelLocked adds new levels during this tick, they are skipped
	// until the next tick, which is safe: new levels only contain far-future tasks.
	dtw.mu.RLock()
	levels := make([]*LevelTimeWheel, len(dtw.levels))
	copy(levels, dtw.levels)
	dtw.mu.RUnlock()

	for i := len(levels) - 1; i >= 0; i-- {
		level := levels[i]
		level.mu.Lock()

		// Calculate ticks and update time/slot
		elapsed := nowUTC.Sub(level.lastTickTime)
		ticks := int(elapsed / level.tickDuration)

		// Collect expired tasks from current slot
		expiredTasks := level.collectExpiredTasksFromCurrentSlot(nowUTC)

		if ticks > 0 {
			oldSlot := level.currentSlot
			level.lastTickTime = level.lastTickTime.Add(time.Duration(ticks) * level.tickDuration)
			level.currentSlot = (level.currentSlot + ticks) % level.wheelSize
			newSlot := level.currentSlot

			// Collect ALL tasks from skipped intermediate slots.
			// These slots' time range has fully elapsed, so all their tasks are overdue.
			steps := ticks - 1
			if steps >= level.wheelSize {
				steps = level.wheelSize - 1 // cap: at most all other slots
			}
			for step := 1; step <= steps; step++ {
				intermediateSlot := (oldSlot + step) % level.wheelSize
				if intermediateSlot == newSlot {
					continue // processSlotTick will handle the new current slot
				}
				skippedTasks := level.collectAllTasksFromSlot(intermediateSlot)
				expiredTasks = append(expiredTasks, skippedTasks...)
			}

			// Process the new current slot (split expired vs remaining for redistribution)
			newExpiredTasks, remainingTasks := level.processSlotTick(nowUTC)
			expiredTasks = append(expiredTasks, newExpiredTasks...)

			// Temporarily store remaining tasks to redistribute after unlocking
			// We cannot call redistributeRemainingTasks here because it might acquire dtw.mu or lowerLevel.mu
			// violating lock ordering (Level -> DTW -> LowerLevel vs DTW -> Level)
			expiredTasks = append(expiredTasks, remainingTasks...)
		}

		level.mu.Unlock()

		// Move all collected tasks (expired and remaining-but-needs-move) to appropriate level.
		// Uses the pre-snapshotted levels to avoid re-acquiring dtw.mu.
		dtw.redistributeTasks(expiredTasks, i, levels, &readyTasks)
	}

	return readyTasks
}

// collectExpiredTasksFromCurrentSlot collects expired tasks from the current slot.
// Caller must hold ltw.mu.
func (ltw *LevelTimeWheel) collectExpiredTasksFromCurrentSlot(now time.Time) []*Task {
	slot := ltw.getSlot(ltw.currentSlot)
	if slot == nil {
		return nil // Slot not initialized, no tasks
	}

	var expiredTasks []*Task
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

// processSlotTick processes the current slot and returns expired and remaining tasks.
// Caller must hold ltw.mu.
func (ltw *LevelTimeWheel) processSlotTick(now time.Time) ([]*Task, []*Task) {
	slot := ltw.getSlot(ltw.currentSlot)
	if slot == nil {
		return nil, nil // Slot not initialized, no tasks
	}

	var expiredTasks []*Task
	var remainingTasks []*Task

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

	// Clear the slot (reinitialize to empty list)
	slot.Init()

	return expiredTasks, remainingTasks
}

// collectAllTasksFromSlot removes and returns ALL tasks from the given slot.
// Used for intermediate slots that have been completely skipped during tick advancement;
// their time window has fully elapsed so all contained tasks are overdue.
// Caller must hold ltw.mu.
func (ltw *LevelTimeWheel) collectAllTasksFromSlot(slotIndex int) []*Task {
	slot := ltw.getSlot(slotIndex)
	if slot == nil {
		return nil // Slot not initialized, no tasks
	}

	var tasks []*Task
	for e := slot.Front(); e != nil; e = e.Next() {
		task := e.Value.(*Task)
		tasks = append(tasks, task)
		delete(ltw.tasks, task.ID)
	}
	slot.Init()
	return tasks
}

// redistributeTasks handles tasks popped from a slot, placing them into the correct level.
// It uses the pre-snapshotted levels slice to avoid re-acquiring dtw.mu during Tick().
//
// For levelIndex == 0:
// - Tasks that are expired (<= now) are ready to run.
// - Tasks that are not expired are re-added to level 0.
//
// For levelIndex > 0:
// - Tasks are cascaded down to levelIndex - 1.
func (dtw *DynamicTimeWheel) redistributeTasks(tasks []*Task, levelIndex int, levels []*LevelTimeWheel, readyTasks *[]*Task) {
	if len(tasks) == 0 {
		return
	}

	now := time.Now().UTC()

	if levelIndex == 0 {
		level0 := levels[0]
		// For Level 0, split tasks: ready vs re-queue.
		// We are NOT holding level0.mu here, so we can safely call level0.addTask which acquires the lock.
		for _, task := range tasks {
			if !task.NextRunTime.UTC().After(now) {
				*readyTasks = append(*readyTasks, task)
			} else {
				level0.addTask(task)
			}
		}
	} else {
		// For higher levels, cascade everything down to the next level.
		lowerLevel := levels[levelIndex-1]
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
			// Skip uninitialized slots
			if slot == nil {
				continue
			}
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
// Used internally by addTask to avoid recursive locking.
func (ltw *LevelTimeWheel) addTaskLocked(task *Task) {
	taskTimeUTC := task.NextRunTime.UTC()
	offset := max(taskTimeUTC.Sub(ltw.lastTickTime), 0)

	ticks := int((offset + ltw.tickDuration - 1) / ltw.tickDuration)
	slotIndex := (ltw.currentSlot + ticks) % ltw.wheelSize

	if _, exists := ltw.tasks[task.ID]; exists {
		ltw.removeTask(task.ID)
	}

	// Use getOrCreateSlot to lazily initialize the slot
	slot := ltw.getOrCreateSlot(slotIndex)
	e := slot.PushBack(task)
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
	// Slot must exist if task exists in tasks map
	slot := ltw.slots[entry.SlotIndex]
	if slot != nil {
		slot.Remove(entry.Element)
	}
	delete(ltw.tasks, taskID)
}
