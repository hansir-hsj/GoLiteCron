package golitecron

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	DefaultTickDuration = time.Millisecond * 200
)

type StorageType int

const (
	StorageTypeHeap StorageType = iota
	StorageTypeTimeWheel
)

type Scheduler struct {
	taskStorage TaskStorage
	mu          sync.Mutex
	wg          sync.WaitGroup
	stopChan    chan struct{}
	running     bool
}

func NewScheduler(storageType ...StorageType) *Scheduler {
	var taskStorage TaskStorage
	if len(storageType) == 0 {
		storageType = append(storageType, StorageTypeHeap)
	}
	switch storageType[0] {
	case StorageTypeTimeWheel:
		taskStorage = NewMultiLevelTimeWheel()
	case StorageTypeHeap:
		fallthrough
	default:
		taskStorage = NewTaskQueue()
	}
	return &Scheduler{
		taskStorage: taskStorage,
		stopChan:    make(chan struct{}),
	}
}

func (s *Scheduler) AddTask(expr CronParser, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := job.GetID()

	if s.taskStorage.TaskExist(id) {
		return fmt.Errorf("task with ID %s already exists", id)
	}

	now := time.Now()
	task := &Task{
		ID:          id,
		Job:         job,
		Expr:        expr,
		NextRunTime: expr.Next(now),
		PreRunTime:  now,
	}
	s.taskStorage.AddTask(task)

	return nil
}

func (s *Scheduler) RemoveTask(task *Task) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.taskStorage.TaskExist(task.ID) {
		return false
	}

	s.taskStorage.RemoveTask(task)

	return false
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(DefaultTickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.Lock()
			tasksToExecute := s.taskStorage.Tick()
			s.mu.Unlock()
			if len(tasksToExecute) == 0 {
				continue
			}

			now := time.Now()
			for _, task := range tasksToExecute {
				s.wg.Add(1)
				go func(t *Task) {
					defer s.wg.Done()

					s.mu.Lock()
					if t.Running {
						s.mu.Unlock()
						return
					}
					t.Running = true
					s.mu.Unlock()

					err := t.Job.Execute()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error executing task %s: %v\n", t.ID, err)
					}

					s.mu.Lock()
					task.PreRunTime = now
					task.NextRunTime = task.Expr.Next(now)
					task.Running = false
					s.taskStorage.AddTask(task)
					s.mu.Unlock()
				}(task)
			}
		}
	}
}
