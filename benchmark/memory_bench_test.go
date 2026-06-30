package benchmark

import (
	"fmt"
	"runtime"
	"testing"

	golitecron "github.com/hansir-hsj/GoLiteCron"
	robfigcron "github.com/robfig/cron/v3"
)

// ============================================================================
// Memory Profiling Benchmarks
// These benchmarks focus on memory allocation patterns and help identify
// memory leaks or excessive allocations
// ============================================================================

// getMemStats returns current memory statistics
func getMemStats() runtime.MemStats {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// ============================================================================
// CronParser Memory Analysis
// ============================================================================

func BenchmarkMemory_Parse_GoLiteCron(b *testing.B) {
	expr := "*/15 9-17 * * 1-5"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		job, _ := golitecron.WrapJob("bench", func() error { return nil })
		_ = s.AddTask(expr, job)
	}
}

func BenchmarkMemory_Parse_RobfigCron(b *testing.B) {
	expr := "*/15 9-17 * * 1-5"
	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

// ============================================================================
// Storage Backend Memory Analysis
// ============================================================================

func BenchmarkMemory_Heap_1000Tasks(b *testing.B) {
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

func BenchmarkMemory_Heap_10000Tasks(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := golitecron.NewScheduler()
		for j := 0; j < 10000; j++ {
			job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d", j), func() error { return nil })
			_ = s.AddTask("*/5 * * * *", job)
		}
	}
}

// ============================================================================
// Scheduler Memory Analysis
// ============================================================================

func BenchmarkMemory_Scheduler_Heap_100Tasks(b *testing.B) {
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

func BenchmarkMemory_Scheduler_Heap_1000Tasks(b *testing.B) {
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

// ============================================================================
// Comparison with robfig/cron Memory
// ============================================================================

func BenchmarkMemory_Comparison_100Tasks_GoLiteCron(b *testing.B) {
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

func BenchmarkMemory_Comparison_100Tasks_RobfigCron(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := robfigcron.New()
		for j := 0; j < 100; j++ {
			_, _ = c.AddFunc("*/5 * * * *", func() {})
		}
	}
}

func BenchmarkMemory_Comparison_1000Tasks_GoLiteCron(b *testing.B) {
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

func BenchmarkMemory_Comparison_1000Tasks_RobfigCron(b *testing.B) {
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
// Memory Leak Detection Tests (not benchmarks, but important for memory analysis)
// ============================================================================

func TestMemory_LeakDetection_AddRemove(t *testing.T) {
	// Test that memory is properly released after removing tasks
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	for round := 0; round < 10; round++ {
		s := golitecron.NewScheduler()

		// Add 1000 tasks
		for i := 0; i < 1000; i++ {
			job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d", i), func() error { return nil })
			_ = s.AddTask("*/5 * * * *", job)
		}
		tasks := s.GetTasks()

		// Remove all tasks
		for _, task := range tasks {
			s.RemoveTaskByID(task.ID)
		}
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// Check memory growth is reasonable (less than 10MB growth after GC)
	growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	if growth > 10*1024*1024 {
		t.Errorf("Potential memory leak detected: %d bytes growth after add/remove cycles", growth)
	}
}

func TestMemory_LeakDetection_Tick(t *testing.T) {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	for round := 0; round < 100; round++ {
		s := golitecron.NewScheduler()
		for i := 0; i < 100; i++ {
			job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d", i), func() error { return nil })
			_ = s.AddTask("*/5 * * * *", job)
		}
		for _, task := range s.GetTasks() {
			s.RemoveTaskByID(task.ID)
		}
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	if growth > 5*1024*1024 {
		t.Errorf("Potential memory leak in Tick: %d bytes growth", growth)
	}
}

// ============================================================================
// Memory Profile Helper Tests
// Run with: go test -bench=. -memprofile=mem.out ./benchmark/
// Then analyze with: go tool pprof mem.out
// ============================================================================

func BenchmarkMemoryProfile_FullWorkflow(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create scheduler
		s := golitecron.NewScheduler()

		// Add various tasks
		expressions := []string{
			"* * * * *",
			"*/5 * * * *",
			"0 * * * *",
			"0 0 * * *",
			"0 0 1 * *",
		}

		for j := 0; j < 20; j++ {
			for _, expr := range expressions {
				job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d-%s", j, expr), func() error { return nil })
				_ = s.AddTask(expr, job)
			}
		}

		tasks := s.GetTasks()

		// Remove half the tasks
		for i := 0; i < len(tasks)/2; i++ {
			s.RemoveTaskByID(tasks[i].ID)
		}
	}
}

// ============================================================================
// Per-Task Memory Overhead Analysis
// ============================================================================

func BenchmarkMemoryOverhead_PerTask_Heap(b *testing.B) {
	taskCounts := []int{100, 500, 1000, 5000}

	for _, count := range taskCounts {
		b.Run(fmt.Sprintf("Tasks_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s := golitecron.NewScheduler()
				for j := 0; j < count; j++ {
					job, _ := golitecron.WrapJob(fmt.Sprintf("task-%d", j), func() error { return nil })
					_ = s.AddTask("*/5 * * * *", job)
				}
			}
		})
	}
}
