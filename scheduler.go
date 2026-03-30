package golitecron

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultTickDuration = time.Millisecond * 500
)

type StorageType int

const (
	StorageTypeHeap StorageType = iota
	StorageTypeTimeWheel
)

type Scheduler struct {
	taskStorage TaskStorage
	wg          sync.WaitGroup
	stopChan    chan struct{}
	running     int32
	mu          sync.Mutex // protects Start/Stop lifecycle
	taskMu      sync.Mutex // protects task check+add atomicity
}

func NewScheduler(storageType ...StorageType) *Scheduler {
	var taskStorage TaskStorage
	if len(storageType) == 0 {
		storageType = append(storageType, StorageTypeHeap)
	}
	switch storageType[0] {
	case StorageTypeTimeWheel:
		taskStorage = NewDynamicTimeWheel()
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

func (s *Scheduler) LoadTasksFromConfig(config *Config) error {
	for _, taskConfig := range config.Tasks {
		if taskConfig.ID == "" || taskConfig.CronExpr == "" || taskConfig.FuncName == "" {
			return fmt.Errorf("task config is missing required fields: ID, CronExpr, or FuncName")
		}

		fn, ok := GetJob(taskConfig.FuncName)
		if !ok {
			return fmt.Errorf("job function %s not found", taskConfig.FuncName)
		}

		var opts []Option
		if taskConfig.EnableSeconds {
			opts = append(opts, WithSeconds())
		}
		if taskConfig.EnableYears {
			opts = append(opts, WithYears())
		}
		if taskConfig.Timeout > 0 {
			opts = append(opts, WithTimeout(time.Duration(taskConfig.Timeout)*time.Millisecond))
		}
		if taskConfig.Retry > 0 {
			opts = append(opts, WithRetry(taskConfig.Retry))
		}
		if taskConfig.Location != "" {
			loc, err := time.LoadLocation(taskConfig.Location)
			if err != nil {
				return fmt.Errorf("failed to load location %s: %w", taskConfig.Location, err)
			}
			opts = append(opts, WithLocation(loc))
		}

		job, err := WrapJob(taskConfig.ID, fn)
		if err != nil {
			return fmt.Errorf("failed to wrap job %s: %w", taskConfig.FuncName, err)
		}
		if err := s.AddTask(taskConfig.CronExpr, job, opts...); err != nil {
			return fmt.Errorf("failed to add task %s: %w", taskConfig.ID, err)
		}
	}

	return nil
}

func (s *Scheduler) GetTasks() []*Task {
	return s.taskStorage.GetTasks()
}

func (s *Scheduler) GetTaskInfo(taskID string) string {
	tasks := s.taskStorage.GetTasks()
	for _, task := range tasks {
		if task.ID == taskID {
			return fmt.Sprintf("Task ID: %s, Pre Run Time: %s, Next Run Time: %s",
				task.ID, task.PreRunTime.Format(time.RFC3339), task.NextRunTime.Format(time.RFC3339))
		}
	}

	return fmt.Sprintf("Task with ID %s not found", taskID)
}

func (s *Scheduler) AddTask(expr string, job Job, opts ...Option) error {
	// Expensive operations outside the lock: cron parsing and Next() calculation
	// do not require mutual exclusion
	parser, err := newCronParser(expr, opts...)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression: %w", err)
	}

	nowUTC := time.Now().UTC()
	nowInTaskZone := nowUTC.In(parser.location)
	nextRunTime := parser.Next(nowInTaskZone)

	// Check if a valid next run time was found
	if nextRunTime.IsZero() {
		return fmt.Errorf("failed to calculate next run time for task %s: cron expression may be invalid or unsatisfiable", job.ID())
	}

	task := &Task{
		ID:          job.ID(),
		Job:         job,
		CronParser:  parser,
		NextRunTime: nextRunTime,
		PreRunTime:  nowInTaskZone,
	}

	// Lock only for the check+add to guarantee atomicity
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if s.taskStorage.TaskExist(task.ID) {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}
	s.taskStorage.AddTask(task)

	return nil
}

func (s *Scheduler) RemoveTask(task *Task) bool {
	// Always mark task as removed to prevent rescheduling after execution
	// This is important even if task is currently executing (not in storage)
	atomic.StoreInt32(&task.Removed, 1)

	if !s.taskStorage.TaskExist(task.ID) {
		return false
	}

	s.taskStorage.RemoveTask(task)

	return true
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.running) == 1 {
		return
	}
	atomic.StoreInt32(&s.running, 1)
	// Recreate stopChan to allow restart after Stop
	s.stopChan = make(chan struct{})
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.running) == 0 {
		return
	}
	atomic.StoreInt32(&s.running, 0)
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
			nowUTC := time.Now().UTC()
			tasksToExecute := s.taskStorage.Tick(nowUTC)
			if len(tasksToExecute) == 0 {
				continue
			}

			for _, task := range tasksToExecute {
				s.wg.Add(1)
				go func(t *Task) {
					defer func() {
						s.wg.Done()
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stderr, "Recovered from panic in task %s: %v\n", t.ID, r)
						}
					}()

					if !atomic.CompareAndSwapInt32(&t.Running, 0, 1) {
						return
					}
					defer atomic.StoreInt32(&t.Running, 0)

					// timeout control
					var err error
					timeout := t.CronParser.timeout
					timedOut := false

					for i := 0; i < t.CronParser.retry+1; i++ {
						if timeout > 0 {
							ctx, cancel := context.WithTimeout(context.Background(), timeout)

						done := make(chan error, 1)
						go func() {
							defer func() {
								if r := recover(); r != nil {
									done <- fmt.Errorf("panic in task %s: %v", t.ID, r)
								}
							}()
							done <- t.Job.Execute(ctx)
						}()

							select {
							case err = <-done:
								// Task completed successfully or failed within timeout
							case <-ctx.Done():
								err = fmt.Errorf("task %s timed out after %s", t.ID, timeout)
								timedOut = true
							}

							cancel() // Release context resources immediately after each iteration

							// Skip retries on timeout to prevent goroutine accumulation.
							// Note: The timed-out goroutine may still be running (Go limitation).
							// Recommendation: Job implementations should support context cancellation.
							if timedOut {
								fmt.Fprintf(os.Stderr, "Task %s timed out, skipping retries to prevent goroutine accumulation\n", t.ID)
								break
							}
						} else {
							err = t.Job.Execute(context.Background())
						}

						if err != nil {
							fmt.Fprintf(os.Stderr, "Error executing task %s: %v (retry %d)\n", t.ID, err, i)
						} else {
							break
						}
					}
					// Convert the current time to the time zone of the task
					nowUTC := time.Now().UTC()
					nowInTaskZone := nowUTC.In(t.CronParser.location)
					nextRunTime := t.CronParser.Next(nowInTaskZone)

					// Check if a valid next run time was found
					if nextRunTime.IsZero() {
						fmt.Fprintf(os.Stderr, "Task %s: failed to calculate next run time, task will not be rescheduled\n", t.ID)
						return
					}

					// Check if scheduler has been stopped
					if atomic.LoadInt32(&s.running) == 0 {
						return
					}

					// Check if task was explicitly removed during execution
					if atomic.LoadInt32(&t.Removed) == 1 {
						return
					}

					updateTask := &Task{
						ID:          t.ID,
						Job:         t.Job,
						CronParser:  t.CronParser,
						NextRunTime: nextRunTime,
						PreRunTime:  nowInTaskZone,
						Running:     0,
					}

					s.taskStorage.AddTask(updateTask)
				}(task)
			}
		}
	}
}
