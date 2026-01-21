package golitecron

import (
	"testing"
	"time"
)

// helpers
func mapHas(m map[int]struct{}, vals ...int) bool {
	for _, v := range vals {
		if _, ok := m[v]; !ok {
			return false
		}
	}
	return true
}

func TestNewCronParser_InvalidLength(t *testing.T) {
	// 6 fields but seconds not enabled -> should error
	_, err := newCronParser("0 0 0 * * *")
	if err == nil {
		t.Fatalf("expected error for invalid cron expression length, got nil")
	}
}

func TestNewCronParser_ParseFields(t *testing.T) {
	parser, err := newCronParser("*/15 1-3 1,15 * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	// minutes should contain 0,15,30,45
	if !mapHas(parser.minutes, 0, 15, 30, 45) {
		t.Fatalf("minutes did not contain expected values, got %v", parser.minutes)
	}

	// hours should contain 1,2,3
	if !mapHas(parser.hours, 1, 2, 3) {
		t.Fatalf("hours did not contain expected values, got %v", parser.hours)
	}

	// dayOfMonth should contain 1 and 15
	if !mapHas(parser.dayOfMonth, 1, 15) {
		t.Fatalf("dayOfMonth did not contain expected values, got %v", parser.dayOfMonth)
	}

	// months and dayOfWeek should be non-empty (parsed from "*")
	if len(parser.months) == 0 {
		t.Fatalf("months should not be empty")
	}
	if len(parser.dayOfWeek) == 0 {
		t.Fatalf("dayOfWeek should not be empty")
	}
}

func TestNext_NoSeconds(t *testing.T) {
	parser, err := newCronParser("0 0 * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	start := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	next := parser.Next(start)

	expected := time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next returned %v, expected %v", next, expected)
	}
}

func TestWithSeconds_Next(t *testing.T) {
	parser, err := newCronParser("*/30 * * * * *", WithSeconds(), WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	// confirm seconds contains 0 and 30
	if !mapHas(parser.seconds, 0, 30) {
		t.Fatalf("seconds did not contain expected values, got %v", parser.seconds)
	}

	start := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)
	next := parser.Next(start)
	expected := time.Date(2023, time.January, 1, 12, 0, 30, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next returned %v, expected %v", next, expected)
	}
}

func TestL_LastDayOfMonth_Next(t *testing.T) {
	parser, err := newCronParser("0 0 L * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	// Case A: Jan 30 -> should pick Jan 31
	startA := time.Date(2023, time.January, 30, 0, 0, 0, 0, time.UTC)
	nextA := parser.Next(startA)
	expectedA := time.Date(2023, time.January, 31, 0, 0, 0, 0, time.UTC)
	if !nextA.Equal(expectedA) {
		t.Fatalf("Next for Jan30 returned %v, expected %v", nextA, expectedA)
	}

	// Case B: Jan 31 -> should pick Feb 28 (last day of Feb 2023)
	startB := time.Date(2023, time.January, 31, 0, 0, 0, 0, time.UTC)
	nextB := parser.Next(startB)
	expectedB := time.Date(2023, time.February, 28, 0, 0, 0, 0, time.UTC)
	if !nextB.Equal(expectedB) {
		t.Fatalf("Next for Jan31 returned %v, expected %v", nextB, expectedB)
	}
}

func TestL_LastWeekDayOfMonth_Next(t *testing.T) {
	parser, err := newCronParser("0 0 * * 5L", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	// Case A: Jan 20, 2023 (Saturday) -> should pick Jan 27 (Friday)
	startA := time.Date(2023, time.January, 20, 0, 0, 0, 0, time.UTC)
	nextA := parser.Next(startA)
	expectedA := time.Date(2023, time.January, 27, 0, 0, 0, 0, time.UTC)
	if !nextA.Equal(expectedA) {
		t.Fatalf("Next for Jan28 returned %v, expected %v", nextA, expectedA)
	}
}

func TestW_NearstWeekDay_Next(t *testing.T) {
	parser, err := newCronParser("0 0 22W * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	startA := time.Date(2023, time.January, 20, 0, 0, 0, 0, time.UTC)
	nextA := parser.Next(startA)
	expectedA := time.Date(2023, time.January, 23, 0, 0, 0, 0, time.UTC)
	if !nextA.Equal(expectedA) {
		t.Fatalf("Next for Jan28 returned %v, expected %v", nextA, expectedA)
	}
}

// ============================================================================
// Step Expression Tests (*/n, start-end/n, start/n)
// ============================================================================

func TestStep_AnyStep(t *testing.T) {
	// Test */7 which was previously rejected because 60%7 != 0
	// Standard cron allows any valid step value
	parser, err := newCronParser("*/7 * * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser with */7: %v", err)
	}

	// minutes should contain 0,7,14,21,28,35,42,49,56
	expected := []int{0, 7, 14, 21, 28, 35, 42, 49, 56}
	if !mapHas(parser.minutes, expected...) {
		t.Fatalf("*/7 minutes did not contain expected values, got %v", parser.minutes)
	}
	if len(parser.minutes) != len(expected) {
		t.Fatalf("*/7 minutes has wrong length: got %d, expected %d", len(parser.minutes), len(expected))
	}
}

func TestStep_RangeWithStep(t *testing.T) {
	// Test 10-30/5 which should produce 10,15,20,25,30
	parser, err := newCronParser("10-30/5 * * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser with 10-30/5: %v", err)
	}

	expected := []int{10, 15, 20, 25, 30}
	if !mapHas(parser.minutes, expected...) {
		t.Fatalf("10-30/5 minutes did not contain expected values, got %v", parser.minutes)
	}
	if len(parser.minutes) != len(expected) {
		t.Fatalf("10-30/5 minutes has wrong length: got %d, expected %d", len(parser.minutes), len(expected))
	}
}

func TestStep_StartWithStep(t *testing.T) {
	// Test 15/10 which should produce 15,25,35,45,55 (from 15 to 59 with step 10)
	parser, err := newCronParser("15/10 * * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser with 15/10: %v", err)
	}

	expected := []int{15, 25, 35, 45, 55}
	if !mapHas(parser.minutes, expected...) {
		t.Fatalf("15/10 minutes did not contain expected values, got %v", parser.minutes)
	}
	if len(parser.minutes) != len(expected) {
		t.Fatalf("15/10 minutes has wrong length: got %d, expected %d", len(parser.minutes), len(expected))
	}
}

func TestStep_HoursField(t *testing.T) {
	// Test */8 for hours (0-23), should produce 0,8,16
	parser, err := newCronParser("0 */8 * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser with */8 hours: %v", err)
	}

	expected := []int{0, 8, 16}
	if !mapHas(parser.hours, expected...) {
		t.Fatalf("*/8 hours did not contain expected values, got %v", parser.hours)
	}
	if len(parser.hours) != len(expected) {
		t.Fatalf("*/8 hours has wrong length: got %d, expected %d", len(parser.hours), len(expected))
	}
}

func TestStep_SecondsField(t *testing.T) {
	// Test */7 for seconds with seconds enabled
	parser, err := newCronParser("*/7 * * * * *", WithSeconds(), WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser with */7 seconds: %v", err)
	}

	expected := []int{0, 7, 14, 21, 28, 35, 42, 49, 56}
	if !mapHas(parser.seconds, expected...) {
		t.Fatalf("*/7 seconds did not contain expected values, got %v", parser.seconds)
	}
	if len(parser.seconds) != len(expected) {
		t.Fatalf("*/7 seconds has wrong length: got %d, expected %d", len(parser.seconds), len(expected))
	}
}

func TestStep_RangeWithStep_Hours(t *testing.T) {
	// Test 9-17/2 for hours (business hours, every 2 hours): 9,11,13,15,17
	parser, err := newCronParser("0 9-17/2 * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser with 9-17/2 hours: %v", err)
	}

	expected := []int{9, 11, 13, 15, 17}
	if !mapHas(parser.hours, expected...) {
		t.Fatalf("9-17/2 hours did not contain expected values, got %v", parser.hours)
	}
	if len(parser.hours) != len(expected) {
		t.Fatalf("9-17/2 hours has wrong length: got %d, expected %d", len(parser.hours), len(expected))
	}
}

func TestStep_InvalidStep(t *testing.T) {
	// Step value 0 should be rejected
	_, err := newCronParser("*/0 * * * *", WithLocation(time.UTC))
	if err == nil {
		t.Fatalf("expected error for step value 0, got nil")
	}

	// Negative step should be rejected
	_, err = newCronParser("*/-5 * * * *", WithLocation(time.UTC))
	if err == nil {
		t.Fatalf("expected error for negative step value, got nil")
	}
}

func TestStep_InvalidRange(t *testing.T) {
	// Range start > end should be rejected
	_, err := newCronParser("30-10/5 * * * *", WithLocation(time.UTC))
	if err == nil {
		t.Fatalf("expected error for range start > end, got nil")
	}

	// Out of bounds range should be rejected
	_, err = newCronParser("50-70/5 * * * *", WithLocation(time.UTC))
	if err == nil {
		t.Fatalf("expected error for out of bounds range, got nil")
	}
}

func TestStep_Next_WithAnyStep(t *testing.T) {
	// Test that Next() works correctly with */7 minutes
	parser, err := newCronParser("*/7 * * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	// Start at 12:05, next should be 12:07
	start := time.Date(2023, time.January, 1, 12, 5, 0, 0, time.UTC)
	next := parser.Next(start)
	expected := time.Date(2023, time.January, 1, 12, 7, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next returned %v, expected %v", next, expected)
	}

	// Start at 12:56, next should be 13:00 (next hour, minute 0)
	start2 := time.Date(2023, time.January, 1, 12, 56, 0, 0, time.UTC)
	next2 := parser.Next(start2)
	expected2 := time.Date(2023, time.January, 1, 13, 0, 0, 0, time.UTC)
	if !next2.Equal(expected2) {
		t.Fatalf("Next returned %v, expected %v", next2, expected2)
	}
}

func TestStep_Next_WithRangeStep(t *testing.T) {
	// Test that Next() works correctly with 10-30/5 minutes
	parser, err := newCronParser("10-30/5 * * * *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("unexpected error creating parser: %v", err)
	}

	// Start at 12:05, next should be 12:10
	start := time.Date(2023, time.January, 1, 12, 5, 0, 0, time.UTC)
	next := parser.Next(start)
	expected := time.Date(2023, time.January, 1, 12, 10, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("Next returned %v, expected %v", next, expected)
	}

	// Start at 12:30, next should be 13:10 (next valid time in range)
	start2 := time.Date(2023, time.January, 1, 12, 30, 0, 0, time.UTC)
	next2 := parser.Next(start2)
	expected2 := time.Date(2023, time.January, 1, 13, 10, 0, 0, time.UTC)
	if !next2.Equal(expected2) {
		t.Fatalf("Next returned %v, expected %v", next2, expected2)
	}
}
