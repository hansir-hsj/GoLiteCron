package golitecron

import (
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
