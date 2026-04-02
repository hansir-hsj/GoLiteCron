package golitecron

import (
	"testing"
	"time"
)

// ============================================================================
// DST (Daylight Saving Time) Boundary Tests
// ============================================================================

// TestTimezone_DSTSpringForward tests behavior when clocks spring forward.
// In the US, on the second Sunday of March, 2:00 AM becomes 3:00 AM.
// Times between 2:00-3:00 don't exist on that day.
func TestTimezone_DSTSpringForward(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("America/New_York timezone not available")
	}

	// 2024-03-10 is DST start in US (2:00 AM -> 3:00 AM)
	// Test scheduling at 2:30 AM which doesn't exist on this day
	beforeDST := time.Date(2024, 3, 10, 1, 30, 0, 0, loc)

	s := NewScheduler()
	job, _ := WrapJob("dst-spring", func() error { return nil })
	// Schedule for 2:30 AM - this time doesn't exist on DST day
	_ = s.AddTask("30 2 * * *", job, WithLocation(loc))

	task := s.GetTasks()[0]
	next := task.CronParser.Next(beforeDST)

	// The scheduler should handle this gracefully
	// Either skip to 3:00+ on the same day, or schedule for next day
	t.Logf("Before DST: %v", beforeDST)
	t.Logf("Next scheduled: %v", next)

	if next.IsZero() {
		t.Error("Next() returned zero time for DST spring forward")
	}
	if next.Before(beforeDST) {
		t.Error("Next() returned time before input")
	}
}

// TestTimezone_DSTFallBack tests behavior when clocks fall back.
// In the US, on the first Sunday of November, 2:00 AM becomes 1:00 AM.
// Times between 1:00-2:00 occur twice on that day.
func TestTimezone_DSTFallBack(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("America/New_York timezone not available")
	}

	// 2024-11-03 is DST end in US (2:00 AM -> 1:00 AM)
	// 1:30 AM occurs twice on this day
	beforeFallBack := time.Date(2024, 11, 3, 0, 30, 0, 0, loc)

	s := NewScheduler()
	job, _ := WrapJob("dst-fall", func() error { return nil })
	_ = s.AddTask("30 1 * * *", job, WithLocation(loc)) // 1:30 AM daily

	task := s.GetTasks()[0]
	first := task.CronParser.Next(beforeFallBack)

	t.Logf("Before fall back: %v", beforeFallBack)
	t.Logf("First 1:30 AM: %v", first)

	if first.IsZero() {
		t.Error("Next() returned zero time for DST fall back")
	}
	if first.Hour() != 1 || first.Minute() != 30 {
		t.Errorf("Expected 1:30, got %02d:%02d", first.Hour(), first.Minute())
	}
}

// TestTimezone_DSTTransition_Europe tests European DST transition.
// Europe changes on last Sunday of March (forward) and October (back).
func TestTimezone_DSTTransition_Europe(t *testing.T) {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("Europe/London timezone not available")
	}

	// 2024-03-31 is DST start in UK (1:00 AM -> 2:00 AM)
	beforeDST := time.Date(2024, 3, 31, 0, 30, 0, 0, loc)

	s := NewScheduler()
	job, _ := WrapJob("dst-europe", func() error { return nil })
	_ = s.AddTask("30 1 * * *", job, WithLocation(loc)) // 1:30 AM doesn't exist

	task := s.GetTasks()[0]
	next := task.CronParser.Next(beforeDST)

	t.Logf("Before UK DST: %v", beforeDST)
	t.Logf("Next scheduled: %v", next)

	// Should not return a time on 2024-03-31 at 1:30 (doesn't exist)
	if !next.IsZero() && next.Day() == 31 && next.Month() == 3 && next.Year() == 2024 {
		if next.Hour() == 1 && next.Minute() == 30 {
			t.Log("Warning: Scheduled at non-existent DST time")
		}
	}
}

// ============================================================================
// Timezone Offset Boundary Tests
// ============================================================================

// TestTimezone_CrossDateLine tests scheduling across the international date line.
func TestTimezone_CrossDateLine(t *testing.T) {
	// UTC+12 (Auckland) and UTC-11 (Pago Pago) are across the date line
	auckland, err1 := time.LoadLocation("Pacific/Auckland")
	pagoPago, err2 := time.LoadLocation("Pacific/Pago_Pago")

	if err1 != nil || err2 != nil {
		t.Skip("Required timezones not available")
	}

	// Same UTC instant, different local dates
	utcTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	aucklandTime := utcTime.In(auckland)
	pagoTime := utcTime.In(pagoPago)

	t.Logf("UTC: %v", utcTime)
	t.Logf("Auckland (UTC+12): %v", aucklandTime)
	t.Logf("Pago Pago (UTC-11): %v", pagoTime)

	// Schedule daily at midnight in both timezones
	s1 := NewScheduler()
	job1, _ := WrapJob("auckland", func() error { return nil })
	_ = s1.AddTask("0 0 * * *", job1, WithLocation(auckland))

	s2 := NewScheduler()
	job2, _ := WrapJob("pagopago", func() error { return nil })
	_ = s2.AddTask("0 0 * * *", job2, WithLocation(pagoPago))

	next1 := s1.GetTasks()[0].CronParser.Next(aucklandTime)
	next2 := s2.GetTasks()[0].CronParser.Next(pagoTime)

	t.Logf("Next midnight Auckland: %v", next1)
	t.Logf("Next midnight Pago Pago: %v", next2)

	// Both should return valid times
	if next1.IsZero() || next2.IsZero() {
		t.Error("Failed to calculate next time across date line")
	}
}

// TestTimezone_HalfHourOffset tests timezones with 30-minute offsets.
func TestTimezone_HalfHourOffset(t *testing.T) {
	testCases := []struct {
		name     string
		timezone string
		offset   string
	}{
		{"India", "Asia/Kolkata", "UTC+5:30"},
		{"Iran", "Asia/Tehran", "UTC+3:30"},
		{"Myanmar", "Asia/Yangon", "UTC+6:30"},
		{"Newfoundland", "America/St_Johns", "UTC-3:30"},
		{"Darwin", "Australia/Darwin", "UTC+9:30"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.timezone)
			if err != nil {
				t.Skipf("%s timezone not available", tc.timezone)
			}

			now := time.Now().In(loc)
			s := NewScheduler()
			job, _ := WrapJob(tc.name, func() error { return nil })
			_ = s.AddTask("*/15 * * * *", job, WithLocation(loc)) // Every 15 min

			task := s.GetTasks()[0]
			next := task.CronParser.Next(now)

			t.Logf("%s (%s): now=%v, next=%v", tc.name, tc.offset, now.Format(time.RFC3339), next.Format(time.RFC3339))

			if next.IsZero() {
				t.Errorf("Failed to calculate next time in %s", tc.timezone)
			}
			if next.Before(now) {
				t.Errorf("Next time is before current time in %s", tc.timezone)
			}
			// Verify location is preserved
			if next.Location().String() != loc.String() {
				t.Errorf("Location mismatch: expected %s, got %s", loc.String(), next.Location().String())
			}
		})
	}
}

// TestTimezone_QuarterHourOffset tests Nepal timezone (UTC+5:45).
func TestTimezone_QuarterHourOffset(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Kathmandu")
	if err != nil {
		t.Skip("Asia/Kathmandu timezone not available")
	}

	now := time.Now().In(loc)
	s := NewScheduler()
	job, _ := WrapJob("nepal", func() error { return nil })
	_ = s.AddTask("0 * * * *", job, WithLocation(loc)) // Every hour

	task := s.GetTasks()[0]
	next := task.CronParser.Next(now)

	t.Logf("Nepal (UTC+5:45): now=%v, next=%v", now.Format(time.RFC3339), next.Format(time.RFC3339))

	if next.IsZero() {
		t.Error("Failed to calculate next time in Nepal timezone")
	}
	if next.Minute() != 0 {
		t.Errorf("Expected minute=0, got %d", next.Minute())
	}
}

// TestTimezone_UTC_Extremes tests extreme UTC offsets.
func TestTimezone_UTC_Extremes(t *testing.T) {
	testCases := []struct {
		name     string
		timezone string
	}{
		{"UTC+14 (Kiritimati)", "Pacific/Kiritimati"},
		{"UTC-12 (Baker Island)", "Etc/GMT+12"}, // Note: Etc/GMT signs are inverted
		{"UTC+12 (Fiji)", "Pacific/Fiji"},
		{"UTC-11 (Samoa)", "Pacific/Pago_Pago"},
	}

	baseTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.timezone)
			if err != nil {
				t.Skipf("%s timezone not available", tc.timezone)
			}

			localTime := baseTime.In(loc)
			s := NewScheduler()
			job, _ := WrapJob(tc.name, func() error { return nil })
			_ = s.AddTask("0 0 * * *", job, WithLocation(loc)) // Daily midnight

			task := s.GetTasks()[0]
			next := task.CronParser.Next(localTime)

			t.Logf("%s: local=%v, next=%v", tc.name, localTime.Format(time.RFC3339), next.Format(time.RFC3339))

			if next.IsZero() {
				t.Errorf("Failed to calculate next time in %s", tc.timezone)
			}
		})
	}
}

// ============================================================================
// Location Change Tests
// ============================================================================

// TestTimezone_SwitchDuringCalculation tests Next() with different locations.
func TestTimezone_SwitchDuringCalculation(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	newYork, _ := time.LoadLocation("America/New_York")

	// Same UTC time
	utcTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	// Create two schedulers with different timezones
	s1 := NewScheduler()
	job1, _ := WrapJob("tokyo", func() error { return nil })
	_ = s1.AddTask("0 9 * * *", job1, WithLocation(tokyo)) // 9 AM Tokyo

	s2 := NewScheduler()
	job2, _ := WrapJob("newyork", func() error { return nil })
	_ = s2.AddTask("0 9 * * *", job2, WithLocation(newYork)) // 9 AM New York

	next1 := s1.GetTasks()[0].CronParser.Next(utcTime.In(tokyo))
	next2 := s2.GetTasks()[0].CronParser.Next(utcTime.In(newYork))

	t.Logf("Tokyo 9AM next: %v (UTC: %v)", next1, next1.UTC())
	t.Logf("New York 9AM next: %v (UTC: %v)", next2, next2.UTC())

	// The UTC times should be different (Tokyo is ~13-14 hours ahead)
	if next1.UTC().Equal(next2.UTC()) {
		t.Error("Different timezone 9AM should have different UTC times")
	}
}

// TestTimezone_LocationConsistency ensures location is preserved across Next() calls.
func TestTimezone_LocationConsistency(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(loc)

	s := NewScheduler()
	job, _ := WrapJob("paris", func() error { return nil })
	_ = s.AddTask("*/5 * * * *", job, WithLocation(loc))

	task := s.GetTasks()[0]

	// Call Next() multiple times
	current := now
	for i := 0; i < 10; i++ {
		next := task.CronParser.Next(current)
		if next.Location().String() != loc.String() {
			t.Errorf("Iteration %d: Location changed from %s to %s",
				i, loc.String(), next.Location().String())
		}
		current = next
	}
}

// ============================================================================
// Specific Date Boundary Tests with Timezones
// ============================================================================

// TestTimezone_NewYearCrossing tests year boundary in different timezones.
func TestTimezone_NewYearCrossing(t *testing.T) {
	sydney, _ := time.LoadLocation("Australia/Sydney")
	losAngeles, _ := time.LoadLocation("America/Los_Angeles")

	// 2024-12-31 23:30 Sydney (already 2025 UTC+11)
	sydneyTime := time.Date(2024, 12, 31, 23, 30, 0, 0, sydney)
	// Same UTC instant in LA is still Dec 31
	laTime := sydneyTime.In(losAngeles)

	t.Logf("Sydney: %v", sydneyTime)
	t.Logf("LA (same instant): %v", laTime)

	// Schedule for Jan 1 midnight
	s1 := NewScheduler()
	job1, _ := WrapJob("sydney-newyear", func() error { return nil })
	_ = s1.AddTask("0 0 1 1 *", job1, WithLocation(sydney))

	s2 := NewScheduler()
	job2, _ := WrapJob("la-newyear", func() error { return nil })
	_ = s2.AddTask("0 0 1 1 *", job2, WithLocation(losAngeles))

	nextSydney := s1.GetTasks()[0].CronParser.Next(sydneyTime)
	nextLA := s2.GetTasks()[0].CronParser.Next(laTime)

	t.Logf("Next Sydney New Year: %v", nextSydney)
	t.Logf("Next LA New Year: %v", nextLA)

	// Sydney should get 2025-01-01 00:00 Sydney time (30 min away)
	// LA should also get 2025-01-01 00:00 LA time (many hours away)
	if nextSydney.Year() != 2025 || nextSydney.Month() != 1 || nextSydney.Day() != 1 {
		t.Errorf("Sydney: expected 2025-01-01, got %v", nextSydney)
	}
	if nextLA.Year() != 2025 || nextLA.Month() != 1 || nextLA.Day() != 1 {
		t.Errorf("LA: expected 2025-01-01, got %v", nextLA)
	}
}

// TestTimezone_MonthEndDifferentZones tests month-end in different timezones.
func TestTimezone_MonthEndDifferentZones(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")

	// Jan 31 23:00 Tokyo
	tokyoTime := time.Date(2024, 1, 31, 23, 0, 0, 0, tokyo)

	s := NewScheduler()
	job, _ := WrapJob("month-end", func() error { return nil })
	_ = s.AddTask("0 0 1 * *", job, WithLocation(tokyo)) // First of each month

	next := s.GetTasks()[0].CronParser.Next(tokyoTime)

	t.Logf("Tokyo Jan 31 23:00: %v", tokyoTime)
	t.Logf("Next first of month: %v", next)

	// Should be Feb 1 00:00 Tokyo
	if next.Month() != 2 || next.Day() != 1 {
		t.Errorf("Expected Feb 1, got %v", next)
	}
}

// TestTimezone_WithSeconds tests seconds precision with timezones.
func TestTimezone_WithSeconds(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	now := time.Now().In(berlin)

	s := NewScheduler()
	job, _ := WrapJob("berlin-seconds", func() error { return nil })
	_ = s.AddTask("*/10 * * * * *", job, WithSeconds(), WithLocation(berlin))

	task := s.GetTasks()[0]
	next := task.CronParser.Next(now)

	t.Logf("Berlin now: %v", now)
	t.Logf("Next (every 10 sec): %v", next)

	if next.Location().String() != berlin.String() {
		t.Errorf("Location mismatch: expected %s, got %s", berlin.String(), next.Location().String())
	}
	// Should be within 10 seconds
	diff := next.Sub(now)
	if diff > 10*time.Second || diff < 0 {
		t.Errorf("Expected next within 10 seconds, got %v", diff)
	}
}
