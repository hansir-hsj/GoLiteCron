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
	DefaultTickDuration = time.Millisecond * 200
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

		job := WrapJob(taskConfig.ID, fn)
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
			return fmt.Sprintf("Task ID: %s, Next Run Time: %s", task.ID, task.NextRunTime.Format(time.RFC3339))
		}
	}

	return fmt.Sprintf("Task with ID %s not found", taskID)
}

func (s *Scheduler) AddTask(expr string, job Job, opts ...Option) error {
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
	if !s.taskStorage.TaskExist(task.ID) {
		return false
	}

	s.taskStorage.RemoveTask(task)

	return false
}

func (s *Scheduler) Start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler) Stop() {
	if !atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		return
	}
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
			tasksToExecute := s.taskStorage.Tick()
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

					if !atomic.CompareAndSwapInt32(&t.Running, 0, 1) {
						return
					}
					defer atomic.StoreInt32(&t.Running, 0)

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

					task.PreRunTime = nowInLocation
					task.NextRunTime = task.CronParser.Next(nowInLocation)
					s.taskStorage.AddTask(task)
				}(task)
			}
		}
	}
}
