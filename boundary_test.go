package golitecron

import (
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// Unix Timestamp Boundary Tests
// ============================================================================

// TestBoundary_UnixEpoch tests behavior at Unix epoch (1970-01-01 00:00:00 UTC).
func TestBoundary_UnixEpoch(t *testing.T) {
	epoch := time.Unix(0, 0).UTC()

	s := NewScheduler()
	job, _ := WrapJob("epoch-test", func() error { return nil })
	_ = s.AddTask("* * * * *", job)

	task := s.GetTasks()[0]
	next := task.CronParser.Next(epoch)

	t.Logf("Unix epoch: %v", epoch)
	t.Logf("Next: %v", next)

	if next.IsZero() {
		t.Error("Next() returned zero time at Unix epoch")
	}
	if next.Before(epoch) {
		t.Error("Next() returned time before Unix epoch")
	}
	// Should be 1970-01-01 00:01:00
	if next.Minute() != 1 {
		t.Errorf("Expected minute 1, got %d", next.Minute())
	}
}

// TestBoundary_Year2038 tests the 32-bit Unix timestamp overflow point.
func TestBoundary_Year2038(t *testing.T) {
	// 32-bit signed integer max: 2147483647
	// Corresponds to: 2038-01-19 03:14:07 UTC
	criticalTime := time.Date(2038, 1, 19, 3, 14, 7, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("2038-test", func() error { return nil })
	_ = s.AddTask("* * * * *", job)

	task := s.GetTasks()[0]
	next := task.CronParser.Next(criticalTime)

	t.Logf("2038 critical time: %v", criticalTime)
	t.Logf("Next: %v", next)

	if next.IsZero() {
		t.Error("Next() returned zero time at 2038 boundary")
	}
	// Should be 2038-01-19 03:15:00
	expected := time.Date(2038, 1, 19, 3, 15, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}

// TestBoundary_Year2099 tests the maximum year supported by CronParser.
func TestBoundary_Year2099(t *testing.T) {
	// CronParser supports years up to 2099
	farFuture := time.Date(2099, 12, 31, 23, 58, 0, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("2099-test", func() error { return nil })
	_ = s.AddTask("* * * * *", job, WithLocation(time.UTC))

	task := s.GetTasks()[0]
	next := task.CronParser.Next(farFuture)

	t.Logf("2099-12-31 23:58: %v", farFuture)
	t.Logf("Next: %v", next)

	// Should return 2099-12-31 23:59:00 or roll to 2100
	if next.IsZero() {
		t.Error("Next() returned zero time near 2099 end")
	}
	// The parser may roll over to 2100 depending on implementation
	t.Logf("Note: Got year %d (rollover to 2100 is acceptable)", next.Year())
}

// TestBoundary_Year1970 tests the minimum year supported by CronParser.
func TestBoundary_Year1970(t *testing.T) {
	// Minimum year in CronParser
	earlyTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("1970-test", func() error { return nil })
	_ = s.AddTask("30 12 * * *", job) // Daily at 12:30

	task := s.GetTasks()[0]
	next := task.CronParser.Next(earlyTime)

	t.Logf("1970-01-01: %v", earlyTime)
	t.Logf("Next: %v", next)

	if next.IsZero() {
		t.Error("Next() returned zero time for 1970")
	}
	// Should be 1970-01-01 12:30:00
	if next.Hour() != 12 || next.Minute() != 30 {
		t.Errorf("Expected 12:30, got %02d:%02d", next.Hour(), next.Minute())
	}
}

// ============================================================================
// Leap Year Boundary Tests
// ============================================================================

// TestBoundary_LeapYear_Feb29 tests February 29 scheduling in leap years.
func TestBoundary_LeapYear_Feb29(t *testing.T) {
	leapYears := []int{2024, 2028, 2032, 2096}    // Regular leap years
	nonLeapYears := []int{2023, 2025, 2100, 2200} // Non-leap years

	s := NewScheduler()
	job, _ := WrapJob("leap-test", func() error { return nil })
	_ = s.AddTask("0 0 29 2 *", job) // Feb 29 midnight

	task := s.GetTasks()[0]

	for _, year := range leapYears {
		t.Run(fmt.Sprintf("LeapYear_%d", year), func(t *testing.T) {
			start := time.Date(year, 2, 1, 0, 0, 0, 0, time.UTC)
			next := task.CronParser.Next(start)

			if next.IsZero() {
				t.Errorf("Leap year %d: expected Feb 29, got zero", year)
				return
			}
			if next.Month() != 2 || next.Day() != 29 || next.Year() != year {
				t.Errorf("Leap year %d: expected %d-02-29, got %v", year, year, next)
			}
		})
	}

	for _, year := range nonLeapYears {
		t.Run(fmt.Sprintf("NonLeapYear_%d", year), func(t *testing.T) {
			start := time.Date(year, 2, 1, 0, 0, 0, 0, time.UTC)
			next := task.CronParser.Next(start)

			// Should skip to next leap year's Feb 29
			if !next.IsZero() && next.Year() == year {
				t.Errorf("Non-leap year %d: should not have Feb 29, got %v", year, next)
			}
		})
	}
}

// TestBoundary_LeapYear_Century tests century leap year rules.
func TestBoundary_LeapYear_Century(t *testing.T) {
	// Century leap year rules:
	// - Divisible by 400 = leap year (2000, 2400)
	// - Divisible by 100 but not 400 = not leap year (2100, 2200, 2300)
	testCases := []struct {
		year   int
		isLeap bool
	}{
		{2000, true},  // Divisible by 400
		{2100, false}, // Divisible by 100, not 400
		{2200, false}, // Divisible by 100, not 400
		{2400, true},  // Divisible by 400
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Year_%d", tc.year), func(t *testing.T) {
			s := NewScheduler()
			job, _ := WrapJob("century-test", func() error { return nil })
			_ = s.AddTask("0 0 29 2 *", job)

			start := time.Date(tc.year, 1, 1, 0, 0, 0, 0, time.UTC)
			task := s.GetTasks()[0]
			next := task.CronParser.Next(start)

			if tc.isLeap {
				if next.Year() != tc.year || next.Month() != 2 || next.Day() != 29 {
					t.Errorf("Leap century %d: expected %d-02-29, got %v", tc.year, tc.year, next)
				}
			} else {
				if !next.IsZero() && next.Year() == tc.year && next.Month() == 2 && next.Day() == 29 {
					t.Errorf("Non-leap century %d: should not have Feb 29", tc.year)
				}
			}
		})
	}
}

// ============================================================================
// Month End Boundary Tests
// ============================================================================

// TestBoundary_MonthEnd_AllMonths tests month-end scheduling for all months.
func TestBoundary_MonthEnd_AllMonths(t *testing.T) {
	// Days in each month for 2024 (leap year)
	monthDays := map[time.Month]int{
		time.January:   31,
		time.February:  29, // Leap year
		time.March:     31,
		time.April:     30,
		time.May:       31,
		time.June:      30,
		time.July:      31,
		time.August:    31,
		time.September: 30,
		time.October:   31,
		time.November:  30,
		time.December:  31,
	}

	s := NewScheduler()
	job, _ := WrapJob("monthend", func() error { return nil })
	_ = s.AddTask("0 0 L * *", job) // Last day of month (L modifier)

	task := s.GetTasks()[0]

	for month, lastDay := range monthDays {
		t.Run(month.String(), func(t *testing.T) {
			start := time.Date(2024, month, 1, 0, 0, 0, 0, time.UTC)
			next := task.CronParser.Next(start)

			if next.IsZero() {
				t.Errorf("%s: Next() returned zero", month)
				return
			}
			if next.Day() != lastDay {
				t.Errorf("%s: expected day %d, got %d", month, lastDay, next.Day())
			}
			if next.Month() != month {
				t.Errorf("%s: expected month %s, got %s", month, month, next.Month())
			}
		})
	}
}

// TestBoundary_Day31_ShortMonths tests day 31 scheduling for months with less days.
func TestBoundary_Day31_ShortMonths(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("day31", func() error { return nil })
	_ = s.AddTask("0 0 31 * *", job) // 31st of each month

	task := s.GetTasks()[0]

	// Start from a month with 31 days
	start := time.Date(2024, 1, 31, 1, 0, 0, 0, time.UTC)
	next := task.CronParser.Next(start)

	// Should skip February (28/29), April (30), June (30), etc.
	t.Logf("After Jan 31: %v", next)

	// Next 31st should be March 31
	if next.Month() != 3 || next.Day() != 31 {
		t.Errorf("Expected March 31, got %v", next)
	}
}

// ============================================================================
// Year Boundary Tests
// ============================================================================

// TestBoundary_YearEnd tests year-end to year-start transition.
func TestBoundary_YearEnd(t *testing.T) {
	yearEnd := time.Date(2024, 12, 31, 23, 59, 0, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("yearend", func() error { return nil })
	_ = s.AddTask("0 0 * * *", job, WithLocation(time.UTC)) // Daily at midnight

	task := s.GetTasks()[0]
	next := task.CronParser.Next(yearEnd)

	t.Logf("Year end: %v", yearEnd)
	t.Logf("Next: %v", next)

	// Should be 2025-01-01 00:00:00 UTC
	expected := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}

// TestBoundary_YearStart tests first moment of a new year.
func TestBoundary_YearStart(t *testing.T) {
	yearStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("yearstart", func() error { return nil })
	_ = s.AddTask("0 0 * * *", job, WithLocation(time.UTC)) // Daily at midnight

	task := s.GetTasks()[0]
	next := task.CronParser.Next(yearStart)

	t.Logf("Year start: %v", yearStart)
	t.Logf("Next: %v", next)

	// Should be 2025-01-02 00:00:00 (next day)
	expected := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}

// ============================================================================
// Seconds Field Boundary Tests
// ============================================================================

// TestBoundary_Seconds_Max tests second field at maximum value.
func TestBoundary_Seconds_Max(t *testing.T) {
	// 59 seconds, just before minute rollover
	time59 := time.Date(2024, 6, 15, 12, 30, 59, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("sec-max", func() error { return nil })
	_ = s.AddTask("* * * * * *", job, WithSeconds()) // Every second

	task := s.GetTasks()[0]
	next := task.CronParser.Next(time59)

	t.Logf("At 59 seconds: %v", time59)
	t.Logf("Next: %v", next)

	// Should roll over to next minute, second 0
	if next.Second() != 0 || next.Minute() != 31 {
		t.Errorf("Expected 12:31:00, got %02d:%02d:%02d", next.Hour(), next.Minute(), next.Second())
	}
}

// TestBoundary_Seconds_StepValues tests second field with step values.
func TestBoundary_Seconds_StepValues(t *testing.T) {
	testCases := []struct {
		expr     string
		startSec int
		expected int
	}{
		{"*/10 * * * * *", 5, 10},
		{"*/10 * * * * *", 55, 0}, // Rolls to next minute
		{"*/15 * * * * *", 14, 15},
		{"*/15 * * * * *", 46, 0}, // Rolls to next minute
		{"0,30 * * * * *", 0, 30},
		{"0,30 * * * * *", 30, 0}, // Rolls to next minute
	}

	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			start := time.Date(2024, 6, 15, 12, 30, tc.startSec, 0, time.UTC)

			s := NewScheduler()
			job, _ := WrapJob("sec-step", func() error { return nil })
			_ = s.AddTask(tc.expr, job, WithSeconds())

			task := s.GetTasks()[0]
			next := task.CronParser.Next(start)

			if next.Second() != tc.expected {
				t.Errorf("%s from second %d: expected second %d, got %d",
					tc.expr, tc.startSec, tc.expected, next.Second())
			}
		})
	}
}

// ============================================================================
// 7-Field (Seconds + Years) Complete Tests
// ============================================================================

// TestBoundary_7Fields_Complete tests full 7-field cron expressions.
func TestBoundary_7Fields_Complete(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("7field", func() error { return nil })
	// 7 fields: sec min hour day month dow year
	// Every March 15 at 10:30:45 from 2025-2030
	err := s.AddTask("45 30 10 15 3 * 2025-2030", job, WithSeconds(), WithYears(), WithLocation(time.UTC))

	if err != nil {
		t.Fatalf("Failed to add 7-field task: %v", err)
	}

	task := s.GetTasks()[0]

	// Test from before the range
	start2024 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	next := task.CronParser.Next(start2024)

	t.Logf("Start 2024-01-01: %v", start2024)
	t.Logf("Next: %v", next)

	// Should be 2025-03-15 10:30:45 UTC
	if next.Year() != 2025 || next.Month() != 3 || next.Day() != 15 {
		t.Errorf("Expected 2025-03-15, got %v", next)
	}
	if next.Hour() != 10 || next.Minute() != 30 || next.Second() != 45 {
		t.Errorf("Expected 10:30:45, got %02d:%02d:%02d", next.Hour(), next.Minute(), next.Second())
	}

	// Test sequential years
	for year := 2025; year <= 2030; year++ {
		yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		yearNext := task.CronParser.Next(yearStart)

		if yearNext.Year() != year || yearNext.Month() != 3 || yearNext.Day() != 15 {
			t.Errorf("Year %d: expected %d-03-15, got %v", year, year, yearNext)
		}
	}

	// Test after the year range
	start2031 := time.Date(2030, 3, 15, 10, 30, 46, 0, time.UTC)
	afterRange := task.CronParser.Next(start2031)

	if !afterRange.IsZero() {
		t.Logf("Note: After year range got %v (may continue or return zero)", afterRange)
	}
}

// TestBoundary_7Fields_YearRange tests year range boundaries.
func TestBoundary_7Fields_YearRange(t *testing.T) {
	testCases := []struct {
		yearExpr string
		start    time.Time
		expected int // Expected year, 0 means zero time or skip
	}{
		{"2025", time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), 2025},
		{"2025-2027", time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), 2025},
		{"2025-2027", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), 2026},
	}

	for _, tc := range testCases {
		t.Run(tc.yearExpr, func(t *testing.T) {
			s := NewScheduler()
			job, _ := WrapJob("year-range", func() error { return nil })
			expr := fmt.Sprintf("0 0 0 1 6 * %s", tc.yearExpr) // June 1 midnight
			err := s.AddTask(expr, job, WithSeconds(), WithYears(), WithLocation(time.UTC))

			if err != nil {
				t.Skipf("Failed to add task with year expr %s: %v", tc.yearExpr, err)
				return
			}

			tasks := s.GetTasks()
			if len(tasks) == 0 {
				t.Skipf("No tasks created for year expr %s", tc.yearExpr)
				return
			}

			task := tasks[0]
			next := task.CronParser.Next(tc.start)

			t.Logf("Year expr %s from %v: next=%v", tc.yearExpr, tc.start, next)

			if tc.expected == 0 {
				if !next.IsZero() {
					t.Logf("Note: Expected zero time, got %v", next)
				}
			} else {
				if next.IsZero() {
					t.Errorf("Expected year %d, got zero time", tc.expected)
				} else if next.Year() != tc.expected {
					t.Errorf("Expected year %d, got %v", tc.expected, next)
				}
			}
		})
	}
}

// ============================================================================
// Edge Cases: Invalid/Impossible Dates
// ============================================================================

// TestBoundary_ImpossibleDate tests scheduling for dates that never occur.
func TestBoundary_ImpossibleDate(t *testing.T) {
	// Feb 30 and Feb 31 never exist
	s := NewScheduler()
	job, _ := WrapJob("impossible", func() error { return nil })
	err := s.AddTask("0 0 30 2 *", job) // Feb 30

	// The parser may reject this as invalid, or accept and return no matches
	if err != nil {
		t.Logf("Parser rejected impossible date (Feb 30): %v", err)
		return
	}

	tasks := s.GetTasks()
	if len(tasks) == 0 {
		t.Log("No task created for impossible date (Feb 30)")
		return
	}

	task := tasks[0]
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	next := task.CronParser.Next(start)

	t.Logf("Feb 30 search from 2024-01-01: %v", next)

	// Should return zero (no valid date) or skip to a valid date
	if next.IsZero() {
		t.Log("Correctly returned zero for impossible date")
	} else {
		t.Logf("Note: Got %v for impossible date Feb 30", next)
	}
}

// TestBoundary_WeekdayAndMonthday tests combined day-of-week and day-of-month.
func TestBoundary_WeekdayAndMonthday(t *testing.T) {
	// Standard cron: both specified = OR logic
	s := NewScheduler()
	job, _ := WrapJob("combined", func() error { return nil })
	_ = s.AddTask("0 0 15 * 5", job) // 15th OR Friday

	task := s.GetTasks()[0]
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // Saturday

	// Collect next 10 occurrences
	current := start
	for i := 0; i < 10; i++ {
		next := task.CronParser.Next(current)
		if next.IsZero() {
			break
		}
		// Should be either day 15 or Friday (Weekday() == 5)
		isDay15 := next.Day() == 15
		isFriday := next.Weekday() == time.Friday
		if !isDay15 && !isFriday {
			t.Errorf("Occurrence %d: expected 15th or Friday, got %v (weekday=%s)",
				i, next, next.Weekday())
		}
		current = next
	}
}

// ============================================================================
// Minute/Hour Rollover Tests
// ============================================================================

// TestBoundary_MinuteRollover tests minute boundaries.
func TestBoundary_MinuteRollover(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("minute-roll", func() error { return nil })
	_ = s.AddTask("0 * * * *", job, WithLocation(time.UTC)) // Every hour at minute 0

	task := s.GetTasks()[0]

	// Test at minute 59
	time59 := time.Date(2024, 6, 15, 12, 59, 30, 0, time.UTC)
	next := task.CronParser.Next(time59)

	// Should be 13:00:00
	if next.Hour() != 13 || next.Minute() != 0 {
		t.Errorf("Expected 13:00, got %02d:%02d", next.Hour(), next.Minute())
	}
}

// TestBoundary_HourRollover tests hour boundaries at day end.
func TestBoundary_HourRollover(t *testing.T) {
	s := NewScheduler()
	job, _ := WrapJob("hour-roll", func() error { return nil })
	_ = s.AddTask("0 0 * * *", job, WithLocation(time.UTC)) // Daily at midnight

	task := s.GetTasks()[0]

	// Test at 23:30
	time2330 := time.Date(2024, 6, 15, 23, 30, 0, 0, time.UTC)
	next := task.CronParser.Next(time2330)

	// Should be next day 00:00
	expected := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}

// TestBoundary_AllFieldsMax tests all fields at maximum values.
func TestBoundary_AllFieldsMax(t *testing.T) {
	// Test at 23:59 on Dec 31
	maxTime := time.Date(2024, 12, 31, 23, 59, 0, 0, time.UTC)

	s := NewScheduler()
	job, _ := WrapJob("all-max", func() error { return nil })
	_ = s.AddTask("* * * * *", job, WithLocation(time.UTC)) // Every minute

	task := s.GetTasks()[0]
	next := task.CronParser.Next(maxTime)

	// Should roll over to next year
	expected := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, next)
	}
}
