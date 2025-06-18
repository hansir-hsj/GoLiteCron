package golitecron

import "time"

type TaskQueue []*Task

func (tq TaskQueue) Len() int {
	return len(tq)
}

func (tq TaskQueue) Less(i, j int) bool {
	return tq[i].NextRunTime.Before(tq[j].NextRunTime)
}

func (tq TaskQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
}

func (tq *TaskQueue) Push(task any) {
	t := task.(*Task)
	*tq = append(*tq, t)
}

func (tq *TaskQueue) Pop() any {
	if len(*tq) == 0 {
		return nil
	}
	task := (*tq)[len(*tq)-1]
	*tq = (*tq)[:len(*tq)-1]
	return task
}

func (tq TaskQueue) NextWaitTime() time.Duration {
	if len(tq) == 0 {
		return 0
	}

	now := time.Now()
	if tq[0].NextRunTime.Before(now) {
		return 0
	}
	return tq[0].NextRunTime.Sub(now)
}
