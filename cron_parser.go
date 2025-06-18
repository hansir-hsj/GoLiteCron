package golitecron

import "time"

type CronParser interface {
	Next(time.Time) time.Time
}
