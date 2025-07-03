package golitecron

import "time"

type Parser interface {
	Next(time.Time) time.Time
}
