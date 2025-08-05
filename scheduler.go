package golitecron

import (
	"context"
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
	mu          sync.RWMutex
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

func (s *Scheduler) GetTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.taskStorage.GetTasks()
}

func (s *Scheduler) GetTaskInfo(taskID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := s.taskStorage.GetTasks()
	for _, task := range tasks {
		if task.ID == taskID {
			return fmt.Sprintf("Task ID: %s, Next Run Time: %s", task.ID, task.NextRunTime.Format(time.RFC3339))
		}
	}

	return fmt.Sprintf("Task with ID %s not found", taskID)
}

func (s *Scheduler) AddTask(expr string, job Job, opts ...Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := job.ID()
	if s.taskStorage.TaskExist(id) {
		return fmt.Errorf("task with ID %s already exists", id)
	}

	parser, err := newCronParser(expr, opts...)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression: %w", err)
	}

	now := time.Now()
	task := &Task{
		ID:          id,
		Job:         job,
		CronParser:  parser,
		NextRunTime: parser.Next(now),
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
					defer func() {
						s.wg.Done()
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stderr, "Recovered from panic in task %s: %v\n", t.ID, r)
						}
					}()

					s.mu.Lock()
					if t.Running {
						s.mu.Unlock()
						return
					}
					t.Running = true
					s.mu.Unlock()

					// timeout control
					var err error
					timeout := t.CronParser.timeout

					for i := 0; i < t.CronParser.retry+1; i++ {
						if timeout > 0 {
							ctx, cancel := context.WithTimeout(context.Background(), timeout)
							defer cancel()

							done := make(chan error, 1)
							go func() {
								done <- t.Job.Execute()
							}()

							select {
							case err = <-done:
							case <-ctx.Done():
								err = fmt.Errorf("task %s timed out after %s", t.ID, timeout)
							}
						} else {
							err = t.Job.Execute()
						}

						if err != nil {
							fmt.Fprintf(os.Stderr, "Error executing task %s: %v (retry %d)\n", t.ID, err, i)
						} else {
							break
						}
					}

					// Convert the current time to the time zone of the task
					nowInLocation := now.In(t.CronParser.location)

					s.mu.Lock()
					task.PreRunTime = nowInLocation
					task.NextRunTime = task.CronParser.Next(nowInLocation)
					task.Running = false
					s.taskStorage.AddTask(task)
					s.mu.Unlock()
				}(task)
			}
		}
	}
}
