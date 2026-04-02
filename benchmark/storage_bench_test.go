package benchmark

import (
	"fmt"
	"testing"
	"time"

	golitecron "github.com/hansir-hsj/GoLiteCron"
)

// Task counts for scaling benchmarks
var taskCounts = []int{10, 100, 1000, 10000}

// Helper to create a mock task
func createMockTask(id string, nextRun time.Time) *golitecron.Task {
	return &golitecron.Task{
		ID:          id,
		NextRunTime: nextRun,
	}
}

// ============================================================================
// TaskQueue (Heap) Benchmarks
// ============================================================================

func BenchmarkHeap_AddTask(b *testing.B) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tq := golitecron.NewTaskQueue()
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Hour))
		tq.AddTask(task)
	}
}

func BenchmarkHeap_AddTask_10(b *testing.B) {
	benchmarkHeapAddN(b, 10)
}

func BenchmarkHeap_AddTask_100(b *testing.B) {
	benchmarkHeapAddN(b, 100)
}

func BenchmarkHeap_AddTask_1000(b *testing.B) {
	benchmarkHeapAddN(b, 1000)
}

func BenchmarkHeap_AddTask_10000(b *testing.B) {
	benchmarkHeapAddN(b, 10000)
}

func benchmarkHeapAddN(b *testing.B, n int) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tq := golitecron.NewTaskQueue()
		for j := 0; j < n; j++ {
			task := createMockTask(fmt.Sprintf("task-%d", j), now.Add(time.Duration(j)*time.Minute))
			tq.AddTask(task)
		}
	}
}

func BenchmarkHeap_RemoveTask(b *testing.B) {
	now := time.Now()
	// Pre-allocate all tasks and queues to avoid StopTimer/StartTimer overhead
	tasks := make([]*golitecron.Task, b.N)
	queues := make([]*golitecron.TaskQueue, b.N)
	for i := 0; i < b.N; i++ {
		tasks[i] = createMockTask("remove-task", now.Add(time.Hour))
		queues[i] = golitecron.NewTaskQueue()
		queues[i].AddTask(tasks[i])
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queues[i].RemoveTask(tasks[i])
	}
}

func BenchmarkHeap_RemoveTask_From1000(b *testing.B) {
	now := time.Now()
	baseTasks := make([]*golitecron.Task, 1000)
	for i := 0; i < 1000; i++ {
		baseTasks[i] = createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Duration(i)*time.Minute))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tq := golitecron.NewTaskQueue()
		for _, t := range baseTasks {
			tq.AddTask(t)
		}
		b.StartTimer()
		// Remove middle task
		tq.RemoveTask(baseTasks[500])
	}
}

func BenchmarkHeap_Tick_Empty(b *testing.B) {
	tq := golitecron.NewTaskQueue()
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tq.Tick(now)
	}
}

func BenchmarkHeap_Tick_1000Tasks_0Ready(b *testing.B) {
	now := time.Now()
	tq := golitecron.NewTaskQueue()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Hour+time.Duration(i)*time.Minute))
		tq.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tq.Tick(now)
	}
}

func BenchmarkHeap_Tick_1000Tasks_10Ready(b *testing.B) {
	benchmarkHeapTickReady(b, 1000, 10)
}

func BenchmarkHeap_Tick_1000Tasks_100Ready(b *testing.B) {
	benchmarkHeapTickReady(b, 1000, 100)
}

func benchmarkHeapTickReady(b *testing.B, total, ready int) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tq := golitecron.NewTaskQueue()
		for j := 0; j < total; j++ {
			var nextRun time.Time
			if j < ready {
				nextRun = now.Add(-time.Minute) // Past -> ready
			} else {
				nextRun = now.Add(time.Hour + time.Duration(j)*time.Minute)
			}
			task := createMockTask(fmt.Sprintf("task-%d", j), nextRun)
			tq.AddTask(task)
		}
		b.StartTimer()
		_ = tq.Tick(now)
	}
}

func BenchmarkHeap_GetTasks_1000(b *testing.B) {
	now := time.Now()
	tq := golitecron.NewTaskQueue()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Duration(i)*time.Minute))
		tq.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tq.GetTasks()
	}
}

func BenchmarkHeap_TaskExist_1000(b *testing.B) {
	now := time.Now()
	tq := golitecron.NewTaskQueue()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Duration(i)*time.Minute))
		tq.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tq.TaskExist("task-500")
	}
}

// ============================================================================
// TimeWheel Benchmarks
// ============================================================================

func BenchmarkTimeWheel_AddTask(b *testing.B) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw := golitecron.NewDynamicTimeWheel()
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Hour))
		tw.AddTask(task)
	}
}

func BenchmarkTimeWheel_AddTask_10(b *testing.B) {
	benchmarkTimeWheelAddN(b, 10)
}

func BenchmarkTimeWheel_AddTask_100(b *testing.B) {
	benchmarkTimeWheelAddN(b, 100)
}

func BenchmarkTimeWheel_AddTask_1000(b *testing.B) {
	benchmarkTimeWheelAddN(b, 1000)
}

func BenchmarkTimeWheel_AddTask_10000(b *testing.B) {
	benchmarkTimeWheelAddN(b, 10000)
}

func benchmarkTimeWheelAddN(b *testing.B, n int) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw := golitecron.NewDynamicTimeWheel()
		for j := 0; j < n; j++ {
			task := createMockTask(fmt.Sprintf("task-%d", j), now.Add(time.Duration(j)*time.Second))
			tw.AddTask(task)
		}
	}
}

func BenchmarkTimeWheel_AddTask_FarFuture(b *testing.B) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw := golitecron.NewDynamicTimeWheel()
		// Add task far in future to trigger level expansion
		task := createMockTask("far-task", now.Add(24*time.Hour))
		tw.AddTask(task)
	}
}

func BenchmarkTimeWheel_RemoveTask(b *testing.B) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tw := golitecron.NewDynamicTimeWheel()
		task := createMockTask("remove-task", now.Add(time.Hour))
		tw.AddTask(task)
		b.StartTimer()
		tw.RemoveTask(task)
	}
}

func BenchmarkTimeWheel_RemoveTask_From1000(b *testing.B) {
	now := time.Now()
	tasks := make([]*golitecron.Task, 1000)
	for i := 0; i < 1000; i++ {
		tasks[i] = createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Duration(i)*time.Second))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tw := golitecron.NewDynamicTimeWheel()
		for _, t := range tasks {
			tw.AddTask(t)
		}
		b.StartTimer()
		tw.RemoveTask(tasks[500])
	}
}

func BenchmarkTimeWheel_Tick_Empty(b *testing.B) {
	tw := golitecron.NewDynamicTimeWheel()
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tw.Tick(now)
	}
}

func BenchmarkTimeWheel_Tick_1000Tasks_0Ready(b *testing.B) {
	now := time.Now()
	tw := golitecron.NewDynamicTimeWheel()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Hour+time.Duration(i)*time.Second))
		tw.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tw.Tick(now)
	}
}

func BenchmarkTimeWheel_Tick_1000Tasks_10Ready(b *testing.B) {
	benchmarkTimeWheelTickReady(b, 1000, 10)
}

func BenchmarkTimeWheel_Tick_1000Tasks_100Ready(b *testing.B) {
	benchmarkTimeWheelTickReady(b, 1000, 100)
}

func benchmarkTimeWheelTickReady(b *testing.B, total, ready int) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tw := golitecron.NewDynamicTimeWheel()
		for j := 0; j < total; j++ {
			var nextRun time.Time
			if j < ready {
				nextRun = now.Add(-time.Second) // Past -> ready
			} else {
				nextRun = now.Add(time.Hour + time.Duration(j)*time.Second)
			}
			task := createMockTask(fmt.Sprintf("task-%d", j), nextRun)
			tw.AddTask(task)
		}
		b.StartTimer()
		_ = tw.Tick(now)
	}
}

func BenchmarkTimeWheel_GetTasks_1000(b *testing.B) {
	now := time.Now()
	tw := golitecron.NewDynamicTimeWheel()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Duration(i)*time.Second))
		tw.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tw.GetTasks()
	}
}

func BenchmarkTimeWheel_TaskExist_1000(b *testing.B) {
	now := time.Now()
	tw := golitecron.NewDynamicTimeWheel()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Duration(i)*time.Second))
		tw.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tw.TaskExist("task-500")
	}
}

// ============================================================================
// Heap vs TimeWheel Comparison (same scenarios)
// ============================================================================

func BenchmarkComparison_Add1000_Heap(b *testing.B) {
	benchmarkHeapAddN(b, 1000)
}

func BenchmarkComparison_Add1000_TimeWheel(b *testing.B) {
	benchmarkTimeWheelAddN(b, 1000)
}

func BenchmarkComparison_Tick1000_10Ready_Heap(b *testing.B) {
	benchmarkHeapTickReady(b, 1000, 10)
}

func BenchmarkComparison_Tick1000_10Ready_TimeWheel(b *testing.B) {
	benchmarkTimeWheelTickReady(b, 1000, 10)
}

// ============================================================================
// Parallel Benchmarks
// ============================================================================

func BenchmarkHeap_AddTask_Parallel(b *testing.B) {
	tq := golitecron.NewTaskQueue()
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := createMockTask(fmt.Sprintf("task-%d-%d", i, time.Now().UnixNano()), now.Add(time.Hour))
			tq.AddTask(task)
			i++
		}
	})
}

func BenchmarkTimeWheel_AddTask_Parallel(b *testing.B) {
	tw := golitecron.NewDynamicTimeWheel()
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := createMockTask(fmt.Sprintf("task-%d-%d", i, time.Now().UnixNano()), now.Add(time.Hour))
			tw.AddTask(task)
			i++
		}
	})
}

func BenchmarkHeap_Tick_Parallel(b *testing.B) {
	now := time.Now()
	tq := golitecron.NewTaskQueue()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Hour+time.Duration(i)*time.Minute))
		tq.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = tq.Tick(now)
		}
	})
}

func BenchmarkTimeWheel_Tick_Parallel(b *testing.B) {
	now := time.Now()
	tw := golitecron.NewDynamicTimeWheel()
	for i := 0; i < 1000; i++ {
		task := createMockTask(fmt.Sprintf("task-%d", i), now.Add(time.Hour+time.Duration(i)*time.Second))
		tw.AddTask(task)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = tw.Tick(now)
		}
	})
}
