package benchmark

import (
	"testing"
	"time"

	golitecron "github.com/hansir-hsj/GoLiteCron"
)

// Test expressions with varying complexity
var benchExpressions = map[string]string{
	"Simple":   "0 0 * * *",          // Every day at midnight
	"Medium":   "*/15 9-17 * * 1-5",  // Every 15min during work hours on weekdays
	"Complex":  "0 0,30 9-17 1,15 *", // Specific times on 1st and 15th
	"Minutely": "* * * * *",          // Every minute
	"Hourly":   "0 * * * *",          // Every hour
}

var benchExpressionsWithSeconds = map[string]string{
	"EverySecond":  "* * * * * *",        // Every second
	"Every10Sec":   "*/10 * * * * *",     // Every 10 seconds
	"ComplexSec":   "0,30 */5 * * * *",   // 0 and 30 sec of every 5 min
	"RangeSec":     "10-20 0 * * * *",    // Seconds 10-20 at minute 0
	"WorkHoursSec": "0 */15 9-17 * * 1-5", // Every 15min during work hours
}

// Helper to create a scheduler and add a task, returning the task's CronParser
func createParserViaScheduler(b *testing.B, expr string, opts ...golitecron.Option) *golitecron.CronParser {
	b.Helper()
	s := golitecron.NewScheduler()
	job, err := golitecron.WrapJob("bench-job", func() error { return nil })
	if err != nil {
		b.Fatalf("failed to wrap job: %v", err)
	}
	if err := s.AddTask(expr, job, opts...); err != nil {
		b.Fatalf("failed to add task: %v", err)
	}
	tasks := s.GetTasks()
	if len(tasks) == 0 {
		b.Fatal("no tasks found")
	}
	return tasks[0].CronParser
}

// ============================================================================
// Cron Expression Parsing Benchmarks (via AddTask)
// ============================================================================

func BenchmarkParse_Simple(b *testing.B) {
	expr := benchExpressions["Simple"]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job)
	}
}

func BenchmarkParse_Medium(b *testing.B) {
	expr := benchExpressions["Medium"]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job)
	}
}

func BenchmarkParse_Complex(b *testing.B) {
	expr := benchExpressions["Complex"]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job)
	}
}

func BenchmarkParse_WithSeconds(b *testing.B) {
	expr := benchExpressionsWithSeconds["Every10Sec"]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job, golitecron.WithSeconds())
	}
}

func BenchmarkParse_WithLocation(b *testing.B) {
	expr := benchExpressions["Simple"]
	loc, _ := time.LoadLocation("America/New_York")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job, golitecron.WithLocation(loc))
	}
}

// ============================================================================
// CronParser.Next() Benchmarks - Core Scheduling Algorithm
// ============================================================================

func BenchmarkNext_Simple(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Simple"])
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_Medium(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Medium"])
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_Complex(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Complex"])
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_Minutely(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Minutely"])
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_WithSeconds(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressionsWithSeconds["Every10Sec"], golitecron.WithSeconds())
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_WithSeconds_Complex(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressionsWithSeconds["WorkHoursSec"], golitecron.WithSeconds())
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

// ============================================================================
// CronParser.Next() Sequential Calls - Simulates Real Scheduling
// ============================================================================

func BenchmarkNext_Sequential_100(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Simple"])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		for j := 0; j < 100; j++ {
			t = parser.Next(t)
		}
	}
}

func BenchmarkNext_Sequential_1000(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Simple"])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		for j := 0; j < 1000; j++ {
			t = parser.Next(t)
		}
	}
}

// ============================================================================
// Parallel Benchmarks - Thread Safety
// ============================================================================

func BenchmarkNext_Parallel(b *testing.B) {
	parser := createParserViaScheduler(b, benchExpressions["Medium"])
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = parser.Next(now)
		}
	})
}

func BenchmarkParse_Parallel(b *testing.B) {
	expr := benchExpressions["Medium"]
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := golitecron.NewScheduler()
			job, _ := golitecron.WrapJob("bench", func() error { return nil })
			_ = s.AddTask(expr, job)
		}
	})
}

// ============================================================================
// Edge Cases
// ============================================================================

func BenchmarkNext_EndOfMonth(b *testing.B) {
	parser := createParserViaScheduler(b, "0 0 28-31 * *") // Last days of month
	// Start from a time near end of month
	now := time.Date(2024, 1, 28, 0, 0, 0, 0, time.Local)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_LeapYear(b *testing.B) {
	parser := createParserViaScheduler(b, "0 0 29 2 *") // Feb 29
	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkNext_YearBoundary(b *testing.B) {
	parser := createParserViaScheduler(b, "0 0 1 1 *") // Jan 1
	now := time.Date(2024, 12, 31, 23, 59, 0, 0, time.Local)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}
