package benchmark

import (
	"fmt"
	"testing"
	"time"

	golitecron "github.com/hansir-hsj/GoLiteCron"
	robfigcron "github.com/robfig/cron/v3"
)

// ============================================================================
// Cron Expression Parsing Comparison
// ============================================================================

func BenchmarkComparison_Parse_Simple_GoLiteCron(b *testing.B) {
	expr := "0 0 * * *"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job)
	}
}

func BenchmarkComparison_Parse_Simple_RobfigCron(b *testing.B) {
	expr := "0 0 * * *"
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

func BenchmarkComparison_Parse_Complex_GoLiteCron(b *testing.B) {
	expr := "*/15 9-17 * * 1-5"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job)
	}
}

func BenchmarkComparison_Parse_Complex_RobfigCron(b *testing.B) {
	expr := "*/15 9-17 * * 1-5"
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

// ============================================================================
// Next() Calculation Comparison - Core Scheduling Algorithm
// ============================================================================

func BenchmarkComparison_Next_Simple_GoLiteCron(b *testing.B) {
	s := golitecron.NewScheduler()
	job, _ := golitecron.WrapJob("bench", func() error { return nil })
	_ = s.AddTask("0 0 * * *", job)
	parser := s.GetTasks()[0].CronParser
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkComparison_Next_Simple_RobfigCron(b *testing.B) {
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	schedule, _ := parser.Parse("0 0 * * *")
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schedule.Next(now)
	}
}

func BenchmarkComparison_Next_Complex_GoLiteCron(b *testing.B) {
	s := golitecron.NewScheduler()
	job, _ := golitecron.WrapJob("bench", func() error { return nil })
	_ = s.AddTask("*/15 9-17 * * 1-5", job)
	parser := s.GetTasks()[0].CronParser
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkComparison_Next_Complex_RobfigCron(b *testing.B) {
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	schedule, _ := parser.Parse("*/15 9-17 * * 1-5")
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schedule.Next(now)
	}
}

func BenchmarkComparison_Next_Minutely_GoLiteCron(b *testing.B) {
	s := golitecron.NewScheduler()
	job, _ := golitecron.WrapJob("bench", func() error { return nil })
	_ = s.AddTask("* * * * *", job)
	parser := s.GetTasks()[0].CronParser
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkComparison_Next_Minutely_RobfigCron(b *testing.B) {
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	schedule, _ := parser.Parse("* * * * *")
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schedule.Next(now)
	}
}

// ============================================================================
// Sequential Next() - Simulating Real Scheduling
// ============================================================================

func BenchmarkComparison_NextSequential100_GoLiteCron(b *testing.B) {
	s := golitecron.NewScheduler()
	job, _ := golitecron.WrapJob("bench", func() error { return nil })
	_ = s.AddTask("0 0 * * *", job)
	parser := s.GetTasks()[0].CronParser

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		for j := 0; j < 100; j++ {
			t = parser.Next(t)
		}
	}
}

func BenchmarkComparison_NextSequential100_RobfigCron(b *testing.B) {
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	schedule, _ := parser.Parse("0 0 * * *")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		for j := 0; j < 100; j++ {
			t = schedule.Next(t)
		}
	}
}

// ============================================================================
// Scheduler Add Task Comparison
// ============================================================================

func BenchmarkComparison_AddTasks100_GoLiteCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		for j := 0; j < 100; j++ {
			job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d", j), func() error { return nil })
			_ = s.AddTask("*/5 * * * *", job)
		}
	}
}

func BenchmarkComparison_AddTasks100_RobfigCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := robfigcron.New()
		for j := 0; j < 100; j++ {
			_, _ = c.AddFunc("*/5 * * * *", func() {})
		}
	}
}

func BenchmarkComparison_AddTasks1000_GoLiteCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		for j := 0; j < 1000; j++ {
			job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d", j), func() error { return nil })
			_ = s.AddTask("*/5 * * * *", job)
		}
	}
}

func BenchmarkComparison_AddTasks1000_RobfigCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := robfigcron.New()
		for j := 0; j < 1000; j++ {
			_, _ = c.AddFunc("*/5 * * * *", func() {})
		}
	}
}

// ============================================================================
// Scheduler Start/Stop Comparison
// ============================================================================

func BenchmarkComparison_StartStop_GoLiteCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("task", func() error { return nil })
		_ = s.AddTask("* * * * *", job)
		s.Start()
		s.Stop()
	}
}

func BenchmarkComparison_StartStop_RobfigCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := robfigcron.New()
		_, _ = c.AddFunc("* * * * *", func() {})
		c.Start()
		c.Stop()
	}
}

// ============================================================================
// Seconds Precision Comparison (robfig/cron also supports seconds)
// ============================================================================

func BenchmarkComparison_Parse_WithSeconds_GoLiteCron(b *testing.B) {
	expr := "*/10 * * * * *"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job, golitecron.WithSeconds())
	}
}

func BenchmarkComparison_Parse_WithSeconds_RobfigCron(b *testing.B) {
	expr := "*/10 * * * * *"
	parser := robfigcron.NewParser(robfigcron.Second | robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

func BenchmarkComparison_Next_WithSeconds_GoLiteCron(b *testing.B) {
	s := golitecron.NewScheduler()
	job, _ := golitecron.WrapJob("bench", func() error { return nil })
	_ = s.AddTask("*/10 * * * * *", job, golitecron.WithSeconds())
	parser := s.GetTasks()[0].CronParser
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Next(now)
	}
}

func BenchmarkComparison_Next_WithSeconds_RobfigCron(b *testing.B) {
	parser := robfigcron.NewParser(robfigcron.Second | robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	schedule, _ := parser.Parse("*/10 * * * * *")
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schedule.Next(now)
	}
}

// ============================================================================
// Parallel Comparison
// ============================================================================

func BenchmarkComparison_Next_Parallel_GoLiteCron(b *testing.B) {
	s := golitecron.NewScheduler()
	job, _ := golitecron.WrapJob("bench", func() error { return nil })
	_ = s.AddTask("*/15 9-17 * * 1-5", job)
	parser := s.GetTasks()[0].CronParser
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = parser.Next(now)
		}
	})
}

func BenchmarkComparison_Next_Parallel_RobfigCron(b *testing.B) {
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	schedule, _ := parser.Parse("*/15 9-17 * * 1-5")
	now := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = schedule.Next(now)
		}
	})
}

func BenchmarkComparison_AddTask_Parallel_GoLiteCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := golitecron.NewScheduler()
			job, _ := golitecron.WrapJob("bench", func() error { return nil })
			_ = s.AddTask("*/5 * * * *", job)
		}
	})
}

func BenchmarkComparison_AddTask_Parallel_RobfigCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := robfigcron.New()
			_, _ = c.AddFunc("*/5 * * * *", func() {})
		}
	})
}
