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
