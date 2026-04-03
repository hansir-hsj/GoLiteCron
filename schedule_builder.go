package golitecron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ScheduleBuilder provides a fluent API for building schedule expressions
type ScheduleBuilder struct {
	scheduler      *Scheduler
	interval       int
	unit           string
	timeSpec       string
	weekday        time.Weekday
	options        []Option
	secondsEnabled bool // Track if WithSeconds was already added
}

// Every starts a new schedule builder with an optional interval.
func (s *Scheduler) Every(intervals ...int) *ScheduleBuilder {
	interval := 1
	if len(intervals) > 0 {
		interval = intervals[0]
	}
	return &ScheduleBuilder{
		scheduler: s,
		interval:  interval,
	}
}

func (sb *ScheduleBuilder) Second() *ScheduleBuilder {
	sb.unit = "second"
	return sb
}

func (sb *ScheduleBuilder) Seconds() *ScheduleBuilder {
	sb.unit = "second"
	return sb
}

func (sb *ScheduleBuilder) Minute() *ScheduleBuilder {
	sb.unit = "minute"
	return sb
}

func (sb *ScheduleBuilder) Minutes() *ScheduleBuilder {
	sb.unit = "minute"
	return sb
}

func (sb *ScheduleBuilder) Hour() *ScheduleBuilder {
	sb.unit = "hour"
	return sb
}

func (sb *ScheduleBuilder) Hours() *ScheduleBuilder {
	sb.unit = "hour"
	return sb
}

func (sb *ScheduleBuilder) Day() *ScheduleBuilder {
	sb.unit = "day"
	return sb
}

func (sb *ScheduleBuilder) Days() *ScheduleBuilder {
	sb.unit = "day"
	return sb
}

func (sb *ScheduleBuilder) Week() *ScheduleBuilder {
	sb.unit = "week"
	return sb
}

func (sb *ScheduleBuilder) Weeks() *ScheduleBuilder {
	sb.unit = "week"
	return sb
}

func (sb *ScheduleBuilder) Month() *ScheduleBuilder {
	sb.unit = "month"
	return sb
}

func (sb *ScheduleBuilder) Months() *ScheduleBuilder {
	sb.unit = "month"
	return sb
}

func (sb *ScheduleBuilder) Monday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Monday
	return sb
}

func (sb *ScheduleBuilder) Tuesday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Tuesday
	return sb
}

func (sb *ScheduleBuilder) Wednesday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Wednesday
	return sb
}

func (sb *ScheduleBuilder) Thursday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Thursday
	return sb
}

func (sb *ScheduleBuilder) Friday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Friday
	return sb
}

func (sb *ScheduleBuilder) Saturday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Saturday
	return sb
}

func (sb *ScheduleBuilder) Sunday() *ScheduleBuilder {
	sb.unit = "weekday"
	sb.weekday = time.Sunday
	return sb
}

// At specifies the time of day (format: "HH:MM" or "HH:MM:SS").
func (sb *ScheduleBuilder) At(timeStr string) *ScheduleBuilder {
	sb.timeSpec = timeStr
	return sb
}

func (sb *ScheduleBuilder) WithTimeout(timeout time.Duration) *ScheduleBuilder {
	sb.options = append(sb.options, WithTimeout(timeout))
	return sb
}

func (sb *ScheduleBuilder) WithRetry(retry int) *ScheduleBuilder {
	sb.options = append(sb.options, WithRetry(retry))
	return sb
}

func (sb *ScheduleBuilder) WithLocation(loc *time.Location) *ScheduleBuilder {
	sb.options = append(sb.options, WithLocation(loc))
	return sb
}

func (sb *ScheduleBuilder) WithSeconds() *ScheduleBuilder {
	if !sb.secondsEnabled {
		sb.options = append(sb.options, WithSeconds())
		sb.secondsEnabled = true
	}
	return sb
}

func (sb *ScheduleBuilder) WithYears() *ScheduleBuilder {
	sb.options = append(sb.options, WithYears())
	return sb
}

// Do registers the job. Accepts func(), func() error, or Job interface.
func (sb *ScheduleBuilder) Do(job any, taskID ...string) error {
	cronExpr, err := sb.buildCronExpression()
	if err != nil {
		return fmt.Errorf("failed to build cron expression: %w", err)
	}

	var wrappedJob Job
	var id string

	if len(taskID) > 0 && taskID[0] != "" {
		id = taskID[0]
	} else {
		id = sb.generateTaskID()
	}

	var wrapErr error
	switch j := job.(type) {
	case func() error:
		wrappedJob, wrapErr = WrapJob(id, j)
	case func():
		wrappedJob, wrapErr = WrapJob(id, func() error {
			j()
			return nil
		})
	case Job:
		wrappedJob, wrapErr = WrapJob(id, j.Execute)
	default:
		return fmt.Errorf("unsupported job type: %T", job)
	}

	if wrapErr != nil {
		return fmt.Errorf("failed to wrap job: %w", wrapErr)
	}

	return sb.scheduler.AddTask(cronExpr, wrappedJob, sb.options...)
}

// buildCronExpression builds a cron expression from the builder configuration.
func (sb *ScheduleBuilder) buildCronExpression() (string, error) {
	var cronExpr string

	switch sb.unit {
	case "second":
		if sb.interval == 1 {
			cronExpr = "* * * * * *"
		} else {
			cronExpr = fmt.Sprintf("*/%d * * * * *", sb.interval)
		}
		sb.WithSeconds() // Use method to avoid duplicate

	case "minute":
		if sb.interval == 1 {
			cronExpr = "* * * * *"
		} else {
			cronExpr = fmt.Sprintf("*/%d * * * *", sb.interval)
		}

	case "hour":
		if sb.interval == 1 {
			cronExpr = "0 * * * *"
		} else {
			cronExpr = fmt.Sprintf("0 */%d * * *", sb.interval)
		}

	case "day":
		if sb.interval > 1 {
			return "", fmt.Errorf("interval > 1 not supported for Day unit (cron cannot express 'every N days')")
		}
		if sb.timeSpec != "" {
			hour, minute, second, err := sb.parseTimeSpec()
			if err != nil {
				return "", err
			}
			if second >= 0 {
				cronExpr = fmt.Sprintf("%d %d %d * * *", second, minute, hour)
				sb.WithSeconds() // Use method to avoid duplicate
			} else {
				cronExpr = fmt.Sprintf("%d %d * * *", minute, hour)
			}
		} else {
			cronExpr = "0 0 * * *"
		}

	case "week":
		if sb.interval > 1 {
			return "", fmt.Errorf("interval > 1 not supported for Week unit (cron cannot express 'every N weeks')")
		}
		if sb.timeSpec != "" {
			hour, minute, second, err := sb.parseTimeSpec()
			if err != nil {
				return "", err
			}
			if second >= 0 {
				cronExpr = fmt.Sprintf("%d %d %d * * 0", second, minute, hour)
				sb.WithSeconds() // Use method to avoid duplicate
			} else {
				cronExpr = fmt.Sprintf("%d %d * * 0", minute, hour)
			}
		} else {
			cronExpr = "0 0 * * 0"
		}

	case "month":
		if sb.interval > 1 {
			return "", fmt.Errorf("interval > 1 not supported for Month unit (cron cannot express 'every N months')")
		}
		if sb.timeSpec != "" {
			hour, minute, second, err := sb.parseTimeSpec()
			if err != nil {
				return "", err
			}
			if second >= 0 {
				cronExpr = fmt.Sprintf("%d %d %d 1 * *", second, minute, hour)
				sb.WithSeconds() // Use method to avoid duplicate
			} else {
				cronExpr = fmt.Sprintf("%d %d 1 * *", minute, hour)
			}
		} else {
			cronExpr = "0 0 1 * *"
		}

	case "weekday":
		weekdayNum := int(sb.weekday)

		if sb.timeSpec != "" {
			hour, minute, second, err := sb.parseTimeSpec()
			if err != nil {
				return "", err
			}
			if second >= 0 {
				cronExpr = fmt.Sprintf("%d %d %d * * %d", second, minute, hour, weekdayNum)
				sb.WithSeconds() // Use method to avoid duplicate
			} else {
				cronExpr = fmt.Sprintf("%d %d * * %d", minute, hour, weekdayNum)
			}
		} else {
			cronExpr = fmt.Sprintf("0 0 * * %d", weekdayNum)
		}

	default:
		return "", fmt.Errorf("unsupported time unit: %s", sb.unit)
	}

	return cronExpr, nil
}

// parseTimeSpec parses "HH:MM" or "HH:MM:SS" format.
func (sb *ScheduleBuilder) parseTimeSpec() (hour, minute, second int, err error) {
	parts := strings.Split(sb.timeSpec, ":")
	second = -1

	if len(parts) < 2 || len(parts) > 3 {
		return 0, 0, -1, fmt.Errorf("invalid time format: %s, expected HH:MM or HH:MM:SS", sb.timeSpec)
	}

	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, -1, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err = strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, -1, fmt.Errorf("invalid minute: %s", parts[1])
	}

	if len(parts) == 3 {
		second, err = strconv.Atoi(parts[2])
		if err != nil || second < 0 || second > 59 {
			return 0, 0, -1, fmt.Errorf("invalid second: %s", parts[2])
		}
	}

	return hour, minute, second, nil
}

// generateTaskID generates a default task ID based on the schedule.
func (sb *ScheduleBuilder) generateTaskID() string {
	switch sb.unit {
	case "second":
		if sb.interval == 1 {
			return "every-second"
		}
		return fmt.Sprintf("every-%d-seconds", sb.interval)
	case "minute":
		if sb.interval == 1 {
			return "every-minute"
		}
		return fmt.Sprintf("every-%d-minutes", sb.interval)
	case "hour":
		if sb.interval == 1 {
			return "every-hour"
		}
		return fmt.Sprintf("every-%d-hours", sb.interval)
	case "day":
		if sb.timeSpec != "" {
			return fmt.Sprintf("daily-at-%s", strings.ReplaceAll(sb.timeSpec, ":", "-"))
		}
		return "daily"
	case "week":
		if sb.timeSpec != "" {
			return fmt.Sprintf("weekly-at-%s", strings.ReplaceAll(sb.timeSpec, ":", "-"))
		}
		return "weekly"
	case "month":
		if sb.timeSpec != "" {
			return fmt.Sprintf("monthly-at-%s", strings.ReplaceAll(sb.timeSpec, ":", "-"))
		}
		return "monthly"
	case "weekday":
		weekdayName := sb.weekday.String()
		if sb.timeSpec != "" {
			return fmt.Sprintf("%s-at-%s", strings.ToLower(weekdayName), strings.ReplaceAll(sb.timeSpec, ":", "-"))
		}
		return strings.ToLower(weekdayName)
	default:
		return "unknown-task"
	}
}
