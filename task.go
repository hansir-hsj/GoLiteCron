package golitecron

import (
	"time"
)

// TaskInfo is a read-only snapshot of a scheduled task.
// Mutating a TaskInfo value never changes scheduler state.
type TaskInfo struct {
	ID          string
	NextRunTime time.Time
	PreRunTime  time.Time
	Running     bool
	// Deprecated: always false; removed tasks are not returned in snapshots.
	Removed  bool
	Location *time.Location
	Timeout  time.Duration
	Retry    int
}

type taskOptions struct {
	Timeout       time.Duration
	Retry         int
	RetryInterval time.Duration
}

type task struct {
	ID          string
	Job         Job
	CronParser  *CronParser
	Options     taskOptions
	NextRunTime time.Time
	PreRunTime  time.Time

	Removed int32 // Set to 1 when task is explicitly removed by user
}

func newTaskInfo(task *task, running bool) TaskInfo {
	info := TaskInfo{
		ID:          task.ID,
		NextRunTime: task.NextRunTime,
		PreRunTime:  task.PreRunTime,
		Running:     running,
	}

	if task.CronParser != nil {
		info.Location = task.CronParser.location
	}
	info.Timeout = task.Options.Timeout
	info.Retry = task.Options.Retry

	return info
}
