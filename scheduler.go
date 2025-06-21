package golitecron

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	DefTickDuration = time.Minute
	DefWheelSize    = 60 // 1 hour in minutes
)

type Scheduler struct {
	taskTimeWheel *TimeWheel
	mu            sync.Mutex
	wg            sync.WaitGroup
	stopChan      chan struct{}
	running       bool
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		taskTimeWheel: NewTimeWheel(DefTickDuration, DefWheelSize),
		stopChan:      make(chan struct{}),
	}
	return s
}

func (s *Scheduler) AddTask(id string, job Job, expr CronParser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.taskTimeWheel.TaskExist(id) {
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
	s.taskTimeWheel.AddTask(task)

	return nil
}

func (s *Scheduler) RemoveTask(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.taskTimeWheel.TaskExist(id) {
		return false
	}

	s.taskTimeWheel.RemoveTask(&Task{ID: id})

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

	ticker := time.NewTicker(s.taskTimeWheel.tickDuration)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.taskTimeWheel.currentTick = (s.taskTimeWheel.currentTick + 1) % s.taskTimeWheel.wheelSize
		slot := s.taskTimeWheel.slots[s.taskTimeWheel.currentTick]
		if slot.Len() == 0 {
			s.mu.Unlock()
			continue
		}

		tasksToExecute := make([]*Task, 0, slot.Len())
		for e := slot.Front(); e != nil; e = e.Next() {
			task := e.Value.(*Task)
			tasksToExecute = append(tasksToExecute, task)
		}
		slot.Init()
		s.mu.Unlock()

		now := time.Now()
		for _, task := range tasksToExecute {
			s.wg.Add(1)
			go func(t *Task) {
				defer s.wg.Done()

				err := task.Job.Execute()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error executing task %s: %v\n", t.ID, err)
				}

				task.PreRunTime = now
				task.NextRunTime = task.Expr.Next(now)

				s.mu.Lock()
				s.taskTimeWheel.AddTask(task)
				s.mu.Unlock()

			}(task)
		}

	}

}
