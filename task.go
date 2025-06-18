package golitecron

import "time"

type Task struct {
	ID          string
	Job         Job
	Expr        CronParser
	NextRunTime time.Time
	PreRunTime  time.Time
}
