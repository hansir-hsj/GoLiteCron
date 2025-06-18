package golitecron

import (
	"container/heap"
	"fmt"
	"os"
	"sync"
	"time"
)

type Scheduler struct {
	taskQueue TaskQueue
	mu        sync.Mutex
	wg        sync.WaitGroup
	stopChan  chan struct{}
	running   bool
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		taskQueue: make(TaskQueue, 0),
		stopChan:  make(chan struct{}),
		running:   false,
	}
	heap.Init(&s.taskQueue)
	return s
}

func (s *Scheduler) AddTask(id string, job Job, expr CronParser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.taskQueue {
		if task.ID == id {
			return fmt.Errorf("task with ID %s already exists", id)
		}
	}

	now := time.Now()
	task := &Task{
		ID:          id,
		Job:         job,
		Expr:        expr,
		NextRunTime: expr.Next(now),
		PreRunTime:  now,
	}

	heap.Push(&s.taskQueue, task)

	return nil
}

func (s *Scheduler) RemoveTask(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.taskQueue {
		if task.ID == id {
			heap.Remove(&s.taskQueue, i)
			return true
		}
	}
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

	for {
		s.mu.Lock()
		if len(s.taskQueue) == 0 {
			s.mu.Unlock()
			select {
			case <-time.After(1 * time.Second):
				continue
			case <-s.stopChan:
				return
			}
		}

		task := s.taskQueue[0]
		now := time.Now()

		if task.NextRunTime.After(now) {
			waitTime := task.NextRunTime.Sub(now)
			s.mu.Unlock()

			select {
			case <-time.After(waitTime):
			case <-s.stopChan:
				return
			}
			continue
		}

		heap.Pop(&s.taskQueue)
		s.mu.Unlock()

		task.PreRunTime = now
		task.NextRunTime = task.Expr.Next(now)

		// run asynchronously
		s.wg.Add(1)
		go func(t *Task) {
			defer s.wg.Done()
			err := t.Job.Execute()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing task %s: %v\n", t.ID, err)
			}
		}(task)

		s.mu.Lock()
		heap.Push(&s.taskQueue, task)
		s.mu.Unlock()
	}
}
