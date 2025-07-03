package golitecron

import (
	"fmt"
	"time"
)

type Task struct {
	ID          string
	Job         Job
	CronParser  *CronParser
	NextRunTime time.Time
	PreRunTime  time.Time

	Running bool
}

func (t *Task) String() string {
	return fmt.Sprintf("Task(ID: %s, NextRunTime: %s, PreRunTime: %s, Running: %t)", t.ID, t.NextRunTime.Format(time.RFC3339), t.PreRunTime.Format(time.RFC3339), t.Running)
}
