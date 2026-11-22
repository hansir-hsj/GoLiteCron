package golitecron

import (
	"sort"
	"time"
)

// Next returns the next time after t that matches the cron expression.
// Uses field-jumping: O(F × V) where F=fields, V=max values per field.
func (p *CronParser) Next(t time.Time) time.Time {
	if len(p.sortedMonths) == 0 || len(p.sortedHours) == 0 || len(p.sortedMinutes) == 0 ||
		(p.enableSeconds && len(p.sortedSeconds) == 0) {
		return time.Time{}
	}
	t = t.In(p.location)
	after := t
	// Advance by wall clock so the first pass of a repeated hour is not skipped.
	if p.enableSeconds {
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()+1, 0, p.location)
	} else {
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, p.location)
	}

	year := t.Year()
	month := int(t.Month())
	day := t.Day()
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()

	maxYear := year + 400
	if p.enableYears && len(p.sortedYears) > 0 {
		maxYear = p.sortedYears[len(p.sortedYears)-1]
	}

	for year <= maxYear {
		// Year
		if p.enableYears {
			y, found := nextInSorted(p.sortedYears, year)
			if !found || y > maxYear {
				return time.Time{}
			}
			if y != year {
				year = y
				month = p.sortedMonths[0]
				day = 1
				hour = p.sortedHours[0]
				minute = p.sortedMinutes[0]
				second = p.firstSecond()
			}
		}

		// Month
		m, found := nextInSorted(p.sortedMonths, month)
		if !found {
			year++
			month = p.sortedMonths[0]
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if m != month {
			month = m
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
		}

		// Day (handles L/W and OR logic)
		lastDay := daysInMonth(year, time.Month(month))
		if day > lastDay {
			month++
			if month > 12 {
				month = 1
				year++
			}
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		d, dayFound := p.nextValidDay(year, time.Month(month), day)
		if !dayFound {
			month++
			if month > 12 {
				month = 1
				year++
			}
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if d != day {
			day = d
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
		}

		// Hour
		h, found := nextInSorted(p.sortedHours, hour)
		if !found {
			day++
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if h != hour {
			hour = h
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
		}

		// Minute
		mi, found := nextInSorted(p.sortedMinutes, minute)
		if !found {
			hour++
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if mi != minute {
			minute = mi
			second = p.firstSecond()
		}

		// Second
		if p.enableSeconds {
			s, found := nextInSorted(p.sortedSeconds, second)
			if !found {
				minute++
				second = p.sortedSeconds[0]
				continue
			}
			second = s
		}

		// time.Date normalizes invalid dates and DST gaps, so a field mismatch means no such time.
		result := time.Date(year, time.Month(month), day, hour, minute, second, 0, p.location)
		if result.Year() == year && result.Month() == time.Month(month) && result.Day() == day &&
			result.Hour() == hour && result.Minute() == minute && result.Second() == second {
			if result.After(after) {
				// time.Date may pick the second pass; prefer the first.
				if earlier := earlierWallTimeAfter(result, after); !earlier.IsZero() {
					return earlier
				}
				return result
			}
			if repeated := repeatedWallTimeAfter(result, after); !repeated.IsZero() {
				return repeated
			}
		}
		// Try the next candidate.
		if p.enableSeconds {
			second++
			if second > 59 {
				second = 0
				minute++
			}
		} else {
			minute++
		}
		if minute > 59 {
			minute = 0
			hour++
		}
		if hour > 23 {
			hour = 0
			day++
		}
	}

	return time.Time{}
}

// repeatedWallTimeAfter returns the second pass of first's wall time if it is after `after`.
func repeatedWallTimeAfter(first, after time.Time) time.Time {
	for candidate := first.Add(time.Minute); !candidate.After(first.Add(3 * time.Hour)); candidate = candidate.Add(time.Minute) {
		local := candidate.In(first.Location())
		if candidate.After(after) && sameWallClock(local, first) {
			return local
		}
	}
	return time.Time{}
}

// earlierWallTimeAfter returns the first pass of result's wall time if it is after `after`.
func earlierWallTimeAfter(result, after time.Time) time.Time {
	start := result.Add(-3 * time.Hour)
	_, offStart := start.Zone()
	_, offResult := result.Zone()
	if offStart <= offResult {
		return time.Time{}
	}
	for candidate := start; candidate.Before(result); candidate = candidate.Add(time.Minute) {
		local := candidate.In(result.Location())
		if candidate.After(after) && sameWallClock(local, result) {
			return local
		}
	}
	return time.Time{}
}

func sameWallClock(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day() &&
		a.Hour() == b.Hour() && a.Minute() == b.Minute() && a.Second() == b.Second()
}

// nextInSorted finds the smallest value >= val in a sorted slice.
func nextInSorted(sorted []int, val int) (int, bool) {
	idx := sort.SearchInts(sorted, val)
	if idx < len(sorted) {
		return sorted[idx], true
	}
	return 0, false
}

// firstSecond returns the first valid second, or 0 if seconds not enabled.
func (p *CronParser) firstSecond() int {
	if p.enableSeconds && len(p.sortedSeconds) > 0 {
		return p.sortedSeconds[0]
	}
	return 0
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// nextValidDay finds the next valid day >= startDay in the given year/month.
func (p *CronParser) nextValidDay(year int, month time.Month, startDay int) (int, bool) {
	lastDay := daysInMonth(year, month)
	for day := startDay; day <= lastDay; day++ {
		if p.isDayValid(year, month, day) {
			return day, true
		}
	}
	return 0, false
}

// isDayValid checks dayOfMonth/dayOfWeek constraints with OR logic.
func (p *CronParser) isDayValid(year int, month time.Month, day int) bool {
	if p.dayOfMonthWildcard && p.dayOfWeekWildcard {
		return true
	}

	domValid := p.isDayOfMonthMatch(year, month, day)
	dowValid := p.isDayOfWeekMatch(year, month, day)

	if p.dayOfMonthWildcard {
		return dowValid
	}
	if p.dayOfWeekWildcard {
		return domValid
	}
	// Both specified: OR logic (standard cron behavior)
	return domValid || dowValid
}

// isDayOfMonthMatch handles L (last day) and W (nearest weekday).
func (p *CronParser) isDayOfMonthMatch(year int, month time.Month, day int) bool {
	for d := range p.dayOfMonth {
		switch {
		case d == 0: // L: last day of month
			if day == daysInMonth(year, month) {
				return true
			}
		case d < 0: // W: nearest weekday to day -d
			if day == findNearestWeekday(year, month, -d, p.location) {
				return true
			}
		default:
			if day == d {
				return true
			}
		}
	}
	return false
}

// isDayOfWeekMatch handles nL (last nth weekday of month).
func (p *CronParser) isDayOfWeekMatch(year int, month time.Month, day int) bool {
	weekday := int(time.Date(year, month, day, 0, 0, 0, 0, p.location).Weekday())
	for w := range p.dayOfWeek {
		if w < 0 {
			targetWeekday := -w
			if targetWeekday == 7 {
				targetWeekday = 0
			}
			lastDay := findLastWeekdayOfMonth(year, month, targetWeekday, p.location)
			if day == lastDay {
				return true
			}
		} else if weekday == w {
			return true
		}
	}
	return false
}

// findNearestWeekday returns the weekday nearest to targetDay without leaving the month.
func findNearestWeekday(year int, month time.Month, targetDay int, loc *time.Location) int {
	lastDay := daysInMonth(year, month)
	if targetDay < 1 || targetDay > lastDay {
		return -1
	}

	t := time.Date(year, month, targetDay, 0, 0, 0, 0, loc)
	wd := t.Weekday()

	if wd >= time.Monday && wd <= time.Friday {
		return targetDay
	}

	if wd == time.Saturday {
		if targetDay-1 >= 1 {
			return targetDay - 1
		}
		return targetDay + 2
	}

	if wd == time.Sunday {
		if targetDay+1 <= lastDay {
			return targetDay + 1
		}
		return targetDay - 2
	}

	return targetDay
}

// findLastWeekdayOfMonth returns the last occurrence of targetWeekday (0=Sun, 6=Sat).
func findLastWeekdayOfMonth(year int, month time.Month, targetWeekday int, loc *time.Location) int {
	lastDay := daysInMonth(year, month)
	for day := lastDay; day >= 1; day-- {
		t := time.Date(year, month, day, 0, 0, 0, 0, loc)
		if int(t.Weekday()) == targetWeekday {
			return day
		}
	}
	return -1
}
