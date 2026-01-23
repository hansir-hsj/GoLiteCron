package golitecron

import (
	"testing"
	"time"
)

// TestCron_DayOfMonthORDayOfWeek tests OR logic when both are specified
func TestCron_DayOfMonthORDayOfWeek(t *testing.T) {
	// "0 0 15 * 1" means: at midnight on the 15th OR on Monday
	parser, err := newCronParser("0 0 15 * 1", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Start from Jan 1, 2024 (Monday)
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	// Jan 1, 2024 is a Monday, so next should be Jan 2, 2024 (first minute after start on a Monday)
	// Actually, Jan 1 is Monday at 00:00, so next Monday time would be Jan 8 00:00
	// But 15th is Jan 15, and Monday comes first at Jan 8
	// So next should be Jan 8, 2024 (Monday) at 00:00
	expected := time.Date(2024, time.January, 8, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}

	// From Jan 8, next should be Jan 15 (the 15th) at 00:00
	next2 := parser.Next(next)
	expected2 := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	if !next2.Equal(expected2) {
		t.Errorf("expected %v, got %v", expected2, next2)
	}
}

// TestCron_OnlyDayOfMonth tests when only dayOfMonth is specified
func TestCron_OnlyDayOfMonth(t *testing.T) {
	// "0 0 15 * *" means: at midnight on the 15th of every month
	parser, err := newCronParser("0 0 15 * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	start := time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// TestCron_OnlyDayOfWeek tests when only dayOfWeek is specified
func TestCron_OnlyDayOfWeek(t *testing.T) {
	// "0 0 * * 5" means: at midnight every Friday
	parser, err := newCronParser("0 0 * * 5", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Jan 1, 2024 is Monday
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	// First Friday is Jan 5, 2024
	expected := time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// TestCron_BothWildcards tests when both dayOfMonth and dayOfWeek are wildcards
func TestCron_BothWildcards(t *testing.T) {
	// "0 0 * * *" means: at midnight every day
	parser, err := newCronParser("0 0 * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	// Should be next day
	expected := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// TestCron_LeapYearFeb29 tests scheduling for Feb 29 in leap year
func TestCron_LeapYearFeb29(t *testing.T) {
	// "0 0 29 2 *" means: at midnight on Feb 29
	parser, err := newCronParser("0 0 29 2 *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// From March 2024 (leap year), next Feb 29 is 2028
	start := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// TestCron_WithYearsField tests year field parsing
func TestCron_WithYearsField(t *testing.T) {
	// "0 0 1 1 * 2025" means: at midnight on Jan 1, 2025
	parser, err := newCronParser("0 0 1 1 * 2025", WithYears(), WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// TestCron_WithYearsRange tests year range in expression
func TestCron_WithYearsRange(t *testing.T) {
	// "0 0 1 1 * 2025-2027" means: Jan 1 in years 2025, 2026, 2027
	parser, err := newCronParser("0 0 1 1 * 2025-2027", WithYears(), WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)
	expected := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("first: expected %v, got %v", expected, next)
	}

	next2 := parser.Next(next)
	expected2 := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !next2.Equal(expected2) {
		t.Errorf("second: expected %v, got %v", expected2, next2)
	}
}

// TestCron_InvalidExpressionReturnsZero tests that impossible expressions return zero
func TestCron_InvalidExpressionReturnsZero(t *testing.T) {
	// Feb 30 never exists
	parser, err := newCronParser("0 0 30 2 *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	if !next.IsZero() {
		t.Errorf("expected zero time for impossible date, got %v", next)
	}
}

// TestCron_NearestWeekday_Saturday tests W modifier for Saturday
func TestCron_NearestWeekday_Saturday(t *testing.T) {
	// "0 0 6W * *" means: nearest weekday to the 6th
	parser, err := newCronParser("0 0 6W * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Jan 6, 2024 is Saturday, nearest weekday is Friday Jan 5
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v (Friday), got %v", expected, next)
	}
}

// TestCron_NearestWeekday_Sunday tests W modifier for Sunday
func TestCron_NearestWeekday_Sunday(t *testing.T) {
	// "0 0 7W * *" means: nearest weekday to the 7th
	parser, err := newCronParser("0 0 7W * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Jan 7, 2024 is Sunday, nearest weekday is Monday Jan 8
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2024, time.January, 8, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v (Monday), got %v", expected, next)
	}
}

// TestCron_NearestWeekday_FirstDaySunday tests W modifier at month start
func TestCron_NearestWeekday_FirstDaySunday(t *testing.T) {
	// "0 0 1W * *" means: nearest weekday to the 1st
	parser, err := newCronParser("0 0 1W * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Sep 1, 2024 is Sunday, nearest weekday is Monday Sep 2 (can't go to previous month)
	start := time.Date(2024, time.August, 31, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2024, time.September, 2, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v (Monday), got %v", expected, next)
	}
}

// TestCron_Macros tests @yearly, @monthly, etc.
func TestCron_Macros(t *testing.T) {
	testCases := []struct {
		macro    string
		expected string
	}{
		{"@yearly", Yearly},
		{"@annually", Yearly},
		{"@monthly", Monthly},
		{"@weekly", Weekly},
		{"@daily", Daily},
		{"@midnight", Daily},
		{"@hourly", Hourly},
		{"@minutely", Minutely},
	}

	for _, tc := range testCases {
		parser, err := newCronParser(tc.macro, WithLocation(time.UTC))
		if err != nil {
			t.Errorf("failed to create parser for %s: %v", tc.macro, err)
			continue
		}

		// Just verify it parses successfully
		start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
		next := parser.Next(start)
		if next.IsZero() {
			t.Errorf("macro %s returned zero time", tc.macro)
		}
	}
}

// TestCron_CommaList tests comma-separated values
func TestCron_CommaList(t *testing.T) {
	// "0 0,30 * * *" means: at minute 0 and 30 of every hour
	parser, err := newCronParser("0,30 * * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	if !mapHas(parser.minutes, 0, 30) {
		t.Errorf("expected minutes to contain 0 and 30, got %v", parser.minutes)
	}
}

// TestCron_RangeExpression tests range expressions
func TestCron_RangeExpression(t *testing.T) {
	// "0 9-17 * * *" means: at minute 0 of hours 9-17
	parser, err := newCronParser("0 9-17 * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	expected := []int{9, 10, 11, 12, 13, 14, 15, 16, 17}
	if !mapHas(parser.hours, expected...) {
		t.Errorf("expected hours %v, got %v", expected, parser.hours)
	}
}

// TestCron_QuestionMark tests ? as wildcard
func TestCron_QuestionMark(t *testing.T) {
	// "0 0 ? * ?" means: at midnight (same as "0 0 * * *")
	parser, err := newCronParser("0 0 ? * ?", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// TestCron_Normalization tests that seconds default to 0 when enabled but not specified
func TestCron_Normalization(t *testing.T) {
	parser := &CronParser{
		enableSeconds: true,
		seconds:       nil,
	}
	parser.normalization()

	if !mapHas(parser.seconds, 0) {
		t.Error("expected seconds to be normalized to {0}")
	}
}

// TestCron_LocationAware tests timezone handling
func TestCron_LocationAware(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skipf("Tokyo timezone not available: %v", err)
	}

	parser, err := newCronParser("0 9 * * *", WithLocation(tokyo))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Start at midnight UTC (9 AM Tokyo)
	startUTC := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(startUTC.In(tokyo))

	// Should be 9 AM Tokyo time
	if next.Hour() != 9 {
		t.Errorf("expected hour 9 in Tokyo timezone, got %d", next.Hour())
	}
}