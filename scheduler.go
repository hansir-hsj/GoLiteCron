package golitecron

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultTickDuration = time.Millisecond * 500
)

var ErrShutdownFromJob = errors.New("shutdown cannot be called from a scheduler job")
var ErrShutdownDeadlineRequired = errors.New("shutdown requires a deadline while tasks are running")

type schedulerContextKey struct{}

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
	taskStorage  *taskQueue
	logger       Logger
	loggerMu     sync.RWMutex
	jobs         map[string]any
	jobsMu       sync.RWMutex
	tasksWG      *sync.WaitGroup   // tracks tasks started since the last Shutdown
	drainingWGs  []*sync.WaitGroup // groups a Shutdown is still draining
	stopChan     chan struct{}
	runDone      chan struct{}
	runCancel    context.CancelFunc
	tickDuration time.Duration
	running      int32
	mu           sync.Mutex // protects lifecycle state; acquired before taskMu
	taskMu       sync.Mutex // protects task operations
	lifecycle    *taskLifecycle
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		taskStorage:  newTaskQueue(),
		logger:       &stdLogger{Logger: log.New(os.Stderr, "", log.LstdFlags)},
		jobs:         make(map[string]any),
		tasksWG:      &sync.WaitGroup{},
		stopChan:     make(chan struct{}),
		runDone:      closedChan(),
		tickDuration: DefaultTickDuration,
		lifecycle:    newTaskLifecycle(),
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
		return errors.New("logger is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if atomic.LoadInt32(&s.running) == 1 {
		return errors.New("cannot change logger while scheduler is running")
	}

	s.loggerMu.Lock()
	defer s.loggerMu.Unlock()
	s.logger = l
	return nil
}

// WithTickDuration sets how often the scheduler checks for due tasks.
// It must be called while the scheduler is stopped.
func (s *Scheduler) WithTickDuration(duration time.Duration) error {
	if duration <= 0 {
		return errors.New("tick duration must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if atomic.LoadInt32(&s.running) == 1 {
		return errors.New("cannot change tick duration while scheduler is running")
	}
	s.tickDuration = duration
	return nil
}

func (s *Scheduler) logf(format string, args ...any) {
	s.loggerMu.RLock()
	logger := s.logger
	s.loggerMu.RUnlock()
	// A panicking logger must not break task execution; fall back to stderr.
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "golitecron: logger panicked: %v; message: "+format, append([]any{r}, args...)...)
		}
	}()
	logger.Printf(format, args...)
}

func (s *Scheduler) RegisterJob(name string, fn any) error {
	if name == "" {
		return errors.New("job name is empty")
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
	if config == nil {
		return errors.New("config is nil")
	}
	tasks := make([]*task, 0, len(config.Tasks))
	for _, taskConfig := range config.Tasks {
		if taskConfig.ID == "" || taskConfig.CronExpr == "" || taskConfig.FuncName == "" {
			return errors.New("task config is missing required fields: ID, CronExpr, or FuncName")
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
	s.taskMu.Lock()
	tasks := s.taskStorage.taskInfos()
	for _, runningTask := range s.lifecycle.running {
		if atomic.LoadInt32(&runningTask.Removed) == 1 {
			continue
		}
		tasks = append(tasks, newTaskInfo(runningTask, true))
	}
	s.taskMu.Unlock()

	sortTaskInfos(tasks)
	return tasks
}

func (s *Scheduler) GetTask(taskID string) (TaskInfo, bool) {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if runningTask, ok := s.lifecycle.getRunning(taskID); ok {
		if atomic.LoadInt32(&runningTask.Removed) == 1 {
			return TaskInfo{}, false
		}
		return newTaskInfo(runningTask, true), true
	}
	return s.taskStorage.TaskInfoByID(taskID)
}

func (s *Scheduler) AddTask(expr string, job Job, opts ...TaskOption) error {
	if job == nil {
		return errors.New("job is nil")
	}
	return s.addTask(expr, job.ID(), job, false, opts...)
}

func (s *Scheduler) nextAvailableTaskIDLocked(baseID string, allowGeneratedIDSuffix bool) (string, error) {
	if baseID == "" {
		return "", errors.New("task ID is empty")
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
		return errors.New("job is nil")
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
			return errors.New("task ID is empty")
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

// RemoveTaskByID removes a task; a running instance is not rescheduled.
func (s *Scheduler) RemoveTaskByID(taskID string) bool {
	if taskID == "" {
		return false
	}
	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	if !s.taskIDInUse(taskID) {
		return false
	}

	if runningTask, ok := s.lifecycle.getRunning(taskID); ok {
		atomic.StoreInt32(&runningTask.Removed, 1)
	}
	if queuedTask, ok := s.taskStorage.TaskByID(taskID); ok {
		atomic.StoreInt32(&queuedTask.Removed, 1)
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
	go s.run(runCtx, s.tasksWG, stopChan, runDone)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

// stopLocked stops the run loop; s.mu must be held.
func (s *Scheduler) stopLocked() {
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

// Shutdown stops the scheduler and waits for tasks started before it, or for ctx.
// With running tasks and no ctx deadline it stops and returns ErrShutdownDeadlineRequired.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	if owner, _ := ctx.Value(schedulerContextKey{}).(*Scheduler); owner == s {
		return ErrShutdownFromJob
	}
	s.mu.Lock()
	s.stopLocked()
	s.taskMu.Lock()
	hasRunningTasks := len(s.lifecycle.running) > 0
	s.taskMu.Unlock()
	if _, hasDeadline := ctx.Deadline(); hasRunningTasks && !hasDeadline {
		s.mu.Unlock()
		return ErrShutdownDeadlineRequired
	}
	// Swap groups so a later Start cannot race this Wait.
	s.drainingWGs = append(s.drainingWGs, s.tasksWG)
	s.tasksWG = &sync.WaitGroup{}
	waitFor := append([]*sync.WaitGroup(nil), s.drainingWGs...)
	s.mu.Unlock()

	drained := make(chan struct{})
	go func() {
		for _, wg := range waitFor {
			wg.Wait()
		}
		s.mu.Lock()
		s.drainingWGs = withoutWaitGroups(s.drainingWGs, waitFor)
		s.mu.Unlock()
		close(drained)
	}()

	select {
	case <-drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func withoutWaitGroups(groups, drained []*sync.WaitGroup) []*sync.WaitGroup {
	kept := groups[:0]
	for _, wg := range groups {
		if !slices.Contains(drained, wg) {
			kept = append(kept, wg)
		}
	}
	return kept
}

func (s *Scheduler) run(runCtx context.Context, tasksWG *sync.WaitGroup, stopChan <-chan struct{}, runDone chan<- struct{}) {
	defer close(runDone)

	ticker := time.NewTicker(s.tickDuration)
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
				s.lifecycle.markRunning(task)
			}
			s.taskMu.Unlock()
			if len(tasksToExecute) == 0 {
				continue
			}

			for _, scheduledTask := range tasksToExecute {
				tasksWG.Add(1)
				go s.executeTask(runCtx, tasksWG, scheduledTask)
			}
		}
	}
}

func (s *Scheduler) executeTask(runCtx context.Context, tasksWG *sync.WaitGroup, t *task) {
	defer tasksWG.Done()
	defer func() {
		if r := recover(); r != nil {
			// Release the running slot that rescheduleTask did not reach.
			s.taskMu.Lock()
			if current, ok := s.lifecycle.getRunning(t.ID); ok && current == t {
				s.lifecycle.unmarkRunning(t.ID)
			}
			s.taskMu.Unlock()
			s.logf("Recovered from panic in task %s: %v\n", t.ID, r)
		}
	}()

	jobCtx := context.WithValue(runCtx, schedulerContextKey{}, s)
	pending := s.runTaskWithRetry(jobCtx, t)
	if pending != nil {
		s.waitTimedOutTask(t, pending)
	}
	s.rescheduleTask(t)
}

// waitTimedOutTask waits for a timed-out or canceled job to return, warning once if it lingers.
func (s *Scheduler) waitTimedOutTask(t *task, pending <-chan error) {
	timer := time.NewTimer(t.Options.Timeout)
	defer timer.Stop()
	var err error
	select {
	case err = <-pending:
	case <-timer.C:
		s.logf("Task %s is still running after its context ended; it will not be rescheduled until it returns\n", t.ID)
		err = <-pending
	}
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		s.logf("Task %s failed after its context ended: %v\n", t.ID, err)
	}
}

// runTaskWithRetry retries immediately; timeouts are not retried because the job may still run.
func (s *Scheduler) runTaskWithRetry(runCtx context.Context, t *task) <-chan error {
	for attempt := 0; attempt < t.Options.Retry+1; attempt++ {
		timedOut, pending, err := s.runTaskAttempt(runCtx, t)
		if errors.Is(err, context.Canceled) {
			return pending
		}
		if timedOut {
			s.logf("Task %s timed out, skipping retries to prevent goroutine accumulation\n", t.ID)
			return pending
		}
		if err != nil {
			s.logf("Error executing task %s: %v (retry %d)\n", t.ID, err, attempt)
			continue
		}
		return nil
	}
	return nil
}

func (s *Scheduler) runTaskAttempt(runCtx context.Context, t *task) (timedOut bool, pending <-chan error, err error) {
	timeout := t.Options.Timeout
	if timeout <= 0 {
		// Turn panics into errors, as the timeout path does.
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in task %s: %v", t.ID, r)
			}
		}()
		return false, nil, t.Job.Execute(runCtx)
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
		return false, nil, err
	case <-ctx.Done():
		// Stop/Shutdown canceled the run; report cancellation, not a timeout.
		if err := runCtx.Err(); err != nil {
			return false, done, err
		}
		return true, done, fmt.Errorf("task %s timed out after %s", t.ID, timeout)
	}
}

func (s *Scheduler) rescheduleTask(t *task) {
	nowUTC := time.Now().UTC()
	nowInTaskZone := nowUTC.In(t.CronParser.location)
	nextRunTime := t.CronParser.Next(nowInTaskZone)

	if nextRunTime.IsZero() {
		s.taskMu.Lock()
		s.lifecycle.unmarkRunning(t.ID)
		s.taskMu.Unlock()
		s.logf("Task %s: failed to calculate next run time, task will not be rescheduled\n", t.ID)
		return
	}

	updateTask := &task{
		ID:          t.ID,
		Job:         t.Job,
		CronParser:  t.CronParser,
		Options:     t.Options,
		NextRunTime: nextRunTime,
		PreRunTime:  nowInTaskZone,
	}

	s.taskMu.Lock()
	defer s.taskMu.Unlock()
	s.lifecycle.unmarkRunning(t.ID)

	if atomic.LoadInt32(&t.Removed) == 1 {
		return
	}
	s.taskStorage.AddTask(updateTask)
}
