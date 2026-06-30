package golitecron

import (
	"context"
	"errors"
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
	taskStorage *taskQueue
	logger      Logger
	loggerMu    sync.RWMutex
	jobs        map[string]any
	jobsMu      sync.RWMutex
	wg          sync.WaitGroup
	stopChan    chan struct{}
	runDone     chan struct{}
	runCancel   context.CancelFunc
	running     int32
	mu          sync.Mutex // protects Start/Stop
	taskMu      sync.Mutex // protects task operations
	lifecycle   *taskLifecycle
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		taskStorage: newTaskQueue(),
		logger:      &stdLogger{Logger: log.New(os.Stderr, "", log.LstdFlags)},
		jobs:        make(map[string]any),
		stopChan:    make(chan struct{}),
		runDone:     closedChan(),
		lifecycle:   newTaskLifecycle(),
	}
}

func closedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// WithLogger sets a custom logger. It must be called while the scheduler is stopped.
func (s *Scheduler) WithLogger(l Logger) error {
	if l == nil {
		return nil
	}
	if atomic.LoadInt32(&s.running) == 1 {
		return fmt.Errorf("cannot change logger while scheduler is running")
	}

	s.loggerMu.Lock()
	defer s.loggerMu.Unlock()
	s.logger = l
	return nil
}

func (s *Scheduler) logf(format string, args ...any) {
	s.loggerMu.RLock()
	logger := s.logger
	s.loggerMu.RUnlock()
	logger.Printf(format, args...)
}

func (s *Scheduler) RegisterJob(name string, fn any) error {
	if name == "" {
		return fmt.Errorf("job name is empty")
	}
	if _, err := WrapJob(name, fn); err != nil {
		return err
	}

	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	s.jobs[name] = fn
	return nil
}

func (s *Scheduler) GetJob(name string) (any, bool) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	fn, ok := s.jobs[name]
	return fn, ok
}

func (s *Scheduler) LoadTasksFromConfig(config *Config) error {
	tasks := make([]*task, 0, len(config.Tasks))
	for _, taskConfig := range config.Tasks {
		if taskConfig.ID == "" || taskConfig.CronExpr == "" || taskConfig.FuncName == "" {
			return fmt.Errorf("task config is missing required fields: ID, CronExpr, or FuncName")
		}

		fn, ok := s.GetJob(taskConfig.FuncName)
		if !ok {
			return fmt.Errorf("job function %s not found", taskConfig.FuncName)
		}

		var opts []TaskOption
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
		task, err := buildTask(taskConfig.CronExpr, taskConfig.ID, job, opts...)
		if err != nil {
			return fmt.Errorf("failed to build task %s: %w", taskConfig.ID, err)
		}
		tasks = append(tasks, task)
	}

	return s.commitTasks(tasks)
}

func (s *Scheduler) GetTasks() []TaskInfo {
	return s.taskStorage.GetTasks()
}

func (s *Scheduler) GetTask(taskID string) (TaskInfo, bool) {
	tasks := s.taskStorage.GetTasks()
	for _, task := range tasks {
		if task.ID == taskID {
			return task, true
		}
	}

	return TaskInfo{}, false
}

func (s *Scheduler) AddTask(expr string, job Job, opts ...TaskOption) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}
	return s.addTask(expr, job.ID(), job, false, opts...)
}

func (s *Scheduler) nextAvailableTaskIDLocked(baseID string, allowGeneratedIDSuffix bool) (string, error) {
	if baseID == "" {
		return "", fmt.Errorf("task ID is empty")
	}

	taskID := baseID
	if allowGeneratedIDSuffix {
		for suffix := 2; s.taskIDInUse(taskID); suffix++ {
			taskID = fmt.Sprintf("%s-%d", baseID, suffix)
		}
	}

	if s.taskIDInUse(taskID) {
		return "", fmt.Errorf("task with ID %s already exists", taskID)
	}

	return taskID, nil
}

func (s *Scheduler) addTask(expr string, taskID string, job Job, allowGeneratedIDSuffix bool, opts ...TaskOption) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}

	task, err := buildTask(expr, taskID, job, opts...)
	if err != nil {
		return err
	}

	return s.commitTask(task, allowGeneratedIDSuffix)
}

func buildTask(expr string, taskID string, job Job, opts ...TaskOption) (*task, error) {
	settings := newTaskSettings(opts...)
	parser, err := newCronParserFromSettings(expr, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	nowUTC := time.Now().UTC()
	nowInTaskZone := nowUTC.In(parser.location)
	nextRunTime := parser.Next(nowInTaskZone)

	if nextRunTime.IsZero() {
		return nil, fmt.Errorf("failed to calculate next run time for task %s: cron expression may be invalid or unsatisfiable", taskID)
	}

	return &task{
		ID:          taskID,
		Job:         job,
		CronParser:  parser,
		Options:     taskOptions{Timeout: settings.timeout, Retry: settings.retry},
		NextRunTime: nextRunTime,
		PreRunTime:  nowInTaskZone,
	}, nil
}

// addTaskLocked adds a fully built task while taskMu is already held.
func (s *Scheduler) addTaskLocked(task *task) {
	s.lifecycle.clearRemoved(task.ID)
	s.taskStorage.AddTask(task)
}

func (s *Scheduler) commitTask(task *task, allowGeneratedIDSuffix bool) error {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	finalID, err := s.nextAvailableTaskIDLocked(task.ID, allowGeneratedIDSuffix)
	if err != nil {
		return err
	}
	task.ID = finalID
	task.Job = setJobID(task.Job, finalID)
	s.addTaskLocked(task)
	return nil
}

func (s *Scheduler) commitTasks(tasks []*task) error {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	seen := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		if task.ID == "" {
			return fmt.Errorf("task ID is empty")
		}
		if _, ok := seen[task.ID]; ok {
			return fmt.Errorf("task with ID %s already exists", task.ID)
		}
		if s.taskIDInUse(task.ID) {
			return fmt.Errorf("task with ID %s already exists", task.ID)
		}
		seen[task.ID] = struct{}{}
	}

	for _, task := range tasks {
		s.addTaskLocked(task)
	}
	return nil
}

func (s *Scheduler) taskIDInUse(taskID string) bool {
	if s.lifecycle.isRunning(taskID) {
		return true
	}
	return s.taskStorage.TaskExist(taskID)
}

// RemoveTaskByID removes a task by ID and prevents a currently running task
// with the same ID from rescheduling itself after completion.
func (s *Scheduler) RemoveTaskByID(taskID string) bool {
	if taskID == "" {
		return false
	}
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if !s.taskIDInUse(taskID) {
		return false
	}

	s.lifecycle.markRemoved(taskID)

	if s.taskStorage.TaskExist(taskID) {
		s.taskStorage.RemoveTaskByID(taskID)
	}

	return true
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.running) == 1 {
		return
	}
	atomic.StoreInt32(&s.running, 1)
	stopChan := make(chan struct{})
	runDone := make(chan struct{})
	runCtx, runCancel := context.WithCancel(context.Background())
	s.stopChan = stopChan
	s.runDone = runDone
	s.runCancel = runCancel
	s.wg.Add(1)
	go s.run(runCtx, stopChan, runDone)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.running) == 0 {
		return
	}
	atomic.StoreInt32(&s.running, 0)
	if s.runCancel != nil {
		s.runCancel()
	}
	close(s.stopChan)
	runDone := s.runDone
	<-runDone
}

// Shutdown stops the scheduler and waits for the scheduler loop and running
// tasks to finish, or until ctx is canceled.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	s.Stop()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) run(runCtx context.Context, stopChan <-chan struct{}, runDone chan<- struct{}) {
	defer s.wg.Done()
	defer close(runDone)

	ticker := time.NewTicker(DefaultTickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			nowUTC := time.Now().UTC()
			s.taskMu.Lock()
			tasksToExecute := s.taskStorage.Tick(nowUTC)
			for _, task := range tasksToExecute {
				s.lifecycle.markRunning(task.ID)
			}
			s.taskMu.Unlock()
			if len(tasksToExecute) == 0 {
				continue
			}

			for _, scheduledTask := range tasksToExecute {
				s.wg.Add(1)
				go s.executeTask(runCtx, scheduledTask)
			}
		}
	}
}

func (s *Scheduler) executeTask(runCtx context.Context, t *task) {
	defer func() {
		s.taskMu.Lock()
		s.lifecycle.unmarkRunning(t.ID)
		s.taskMu.Unlock()
		s.wg.Done()
		if r := recover(); r != nil {
			s.logf("Recovered from panic in task %s: %v\n", t.ID, r)
		}
	}()

	if !atomic.CompareAndSwapInt32(&t.Running, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&t.Running, 0)

	s.runTaskWithRetry(runCtx, t)
	s.rescheduleTask(t)
}

func (s *Scheduler) runTaskWithRetry(runCtx context.Context, t *task) {
	for attempt := 0; attempt < t.Options.Retry+1; attempt++ {
		err, timedOut := s.runTaskAttempt(runCtx, t)
		if errors.Is(err, context.Canceled) {
			return
		}
		if timedOut {
			s.logf("Task %s timed out, skipping retries to prevent goroutine accumulation\n", t.ID)
			return
		}
		if err != nil {
			s.logf("Error executing task %s: %v (retry %d)\n", t.ID, err, attempt)
			continue
		}
		return
	}
}

func (s *Scheduler) runTaskAttempt(runCtx context.Context, t *task) (error, bool) {
	timeout := t.Options.Timeout
	if timeout <= 0 {
		return t.Job.Execute(runCtx), false
	}

	ctx, cancel := context.WithTimeout(runCtx, timeout)
	defer cancel()

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
	case err := <-done:
		return err, false
	case <-ctx.Done():
		return fmt.Errorf("task %s timed out after %s", t.ID, timeout), true
	}
}

func (s *Scheduler) rescheduleTask(t *task) {
	nowUTC := time.Now().UTC()
	nowInTaskZone := nowUTC.In(t.CronParser.location)
	nextRunTime := t.CronParser.Next(nowInTaskZone)

	if nextRunTime.IsZero() {
		s.logf("Task %s: failed to calculate next run time, task will not be rescheduled\n", t.ID)
		return
	}

	if atomic.LoadInt32(&s.running) == 0 {
		return
	}

	updateTask := &task{
		ID:          t.ID,
		Job:         t.Job,
		CronParser:  t.CronParser,
		Options:     t.Options,
		NextRunTime: nextRunTime,
		PreRunTime:  nowInTaskZone,
		Running:     0,
	}

	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if atomic.LoadInt32(&t.Removed) == 1 {
		return
	}
	if s.lifecycle.isRemoved(t.ID) {
		return
	}
	s.taskStorage.AddTask(updateTask)
}
