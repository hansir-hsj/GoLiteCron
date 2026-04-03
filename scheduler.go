package golitecron

import (
	"context"
	"fmt"
	"log"
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

// Logger defines the logging interface used by the scheduler.
type Logger interface {
	Printf(format string, args ...any)
}

// stdLogger wraps Go's standard log.Logger.
type stdLogger struct {
	*log.Logger
}

func (l *stdLogger) Printf(format string, args ...any) {
	l.Logger.Printf(format, args...)
}

type Scheduler struct {
	taskStorage TaskStorage
	logger      Logger
	wg          sync.WaitGroup
	stopChan    chan struct{}
	running     int32
	mu          sync.Mutex // protects Start/Stop
	taskMu      sync.Mutex // protects task operations
}

func NewScheduler(storageType ...StorageType) *Scheduler {
	var taskStorage TaskStorage
	st := StorageTypeHeap
	if len(storageType) > 0 {
		st = storageType[0]
	}
	switch st {
	case StorageTypeTimeWheel:
		taskStorage = NewDynamicTimeWheel()
	default:
		taskStorage = NewTaskQueue()
	}
	return &Scheduler{
		taskStorage: taskStorage,
		logger:      &stdLogger{Logger: log.New(os.Stderr, "", log.LstdFlags)},
		stopChan:    make(chan struct{}),
	}
}

// WithLogger sets a custom logger. Must be called before Start().
func (s *Scheduler) WithLogger(l Logger) {
	if l != nil {
		s.logger = l
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
		if taskConfig.Timeout != "" {
			timeout, err := time.ParseDuration(taskConfig.Timeout)
			if err != nil {
				return fmt.Errorf("invalid timeout duration %q for task %s: %w", taskConfig.Timeout, taskConfig.ID, err)
			}
			opts = append(opts, WithTimeout(timeout))
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
	parser, err := newCronParser(expr, opts...)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression: %w", err)
	}

	nowUTC := time.Now().UTC()
	nowInTaskZone := nowUTC.In(parser.location)
	nextRunTime := parser.Next(nowInTaskZone)

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

	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if s.taskStorage.TaskExist(task.ID) {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}
	s.taskStorage.AddTask(task)

	return nil
}

func (s *Scheduler) RemoveTask(task *Task) bool {
	atomic.StoreInt32(&task.Removed, 1)

	s.taskMu.Lock()
	defer s.taskMu.Unlock()

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
							s.logger.Printf("Recovered from panic in task %s: %v\n", t.ID, r)
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
							case <-ctx.Done():
								err = fmt.Errorf("task %s timed out after %s", t.ID, timeout)
								timedOut = true
							}

							cancel()

							if timedOut {
								s.logger.Printf("Task %s timed out, skipping retries to prevent goroutine accumulation\n", t.ID)
								break
							}
						} else {
							err = t.Job.Execute(context.Background())
						}

						if err != nil {
							s.logger.Printf("Error executing task %s: %v (retry %d)\n", t.ID, err, i)
						} else {
							break
						}
					}
					// Calculate next run time in task's timezone
					nowUTC := time.Now().UTC()
					nowInTaskZone := nowUTC.In(t.CronParser.location)
					nextRunTime := t.CronParser.Next(nowInTaskZone)

					if nextRunTime.IsZero() {
						s.logger.Printf("Task %s: failed to calculate next run time, task will not be rescheduled\n", t.ID)
						return
					}

					if atomic.LoadInt32(&s.running) == 0 {
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

					s.taskMu.Lock()
					if atomic.LoadInt32(&t.Removed) == 1 {
						s.taskMu.Unlock()
						return
					}
					s.taskStorage.AddTask(updateTask)
					s.taskMu.Unlock()
				}(task)
			}
		}
	}
}
