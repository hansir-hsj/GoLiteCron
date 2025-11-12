package golitecron

import (
	"testing"
	"time"
)

func TestScheduleBuilderBasicUnits(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)

	// 测试基本时间单位
	testCases := []struct {
		name     string
		builder  func() *ScheduleBuilder
		expected string
	}{
		{
			name:     "every second",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Second() },
			expected: "* * * * * *",
		},
		{
			name:     "every 10 seconds",
			builder:  func() *ScheduleBuilder { return scheduler.Every(10).Seconds() },
			expected: "*/10 * * * * *",
		},
		{
			name:     "every minute",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Minute() },
			expected: "* * * * *",
		},
		{
			name:     "every 5 minutes",
			builder:  func() *ScheduleBuilder { return scheduler.Every(5).Minutes() },
			expected: "*/5 * * * *",
		},
		{
			name:     "every hour",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Hour() },
			expected: "0 * * * *",
		},
		{
			name:     "every 2 hours",
			builder:  func() *ScheduleBuilder { return scheduler.Every(2).Hours() },
			expected: "0 */2 * * *",
		},
		{
			name:     "every day",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Day() },
			expected: "0 0 * * *",
		},
		{
			name:     "every week",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Week() },
			expected: "0 0 * * 0",
		},
		{
			name:     "every month",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Month() },
			expected: "0 0 1 * *",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.builder()
			cronExpr, err := builder.buildCronExpression()
			if err != nil {
				t.Fatalf("buildCronExpression failed: %v", err)
			}
			if cronExpr != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, cronExpr)
			}
		})
	}
}

func TestScheduleBuilderWeekdays(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)

	weekdayTests := []struct {
		name     string
		builder  func() *ScheduleBuilder
		expected string
	}{
		{
			name:     "every Monday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Monday() },
			expected: "0 0 * * 1",
		},
		{
			name:     "every Tuesday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Tuesday() },
			expected: "0 0 * * 2",
		},
		{
			name:     "every Wednesday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Wednesday() },
			expected: "0 0 * * 3",
		},
		{
			name:     "every Thursday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Thursday() },
			expected: "0 0 * * 4",
		},
		{
			name:     "every Friday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Friday() },
			expected: "0 0 * * 5",
		},
		{
			name:     "every Saturday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Saturday() },
			expected: "0 0 * * 6",
		},
		{
			name:     "every Sunday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Sunday() },
			expected: "0 0 * * 0",
		},
	}

	for _, tc := range weekdayTests {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.builder()
			cronExpr, err := builder.buildCronExpression()
			if err != nil {
				t.Fatalf("buildCronExpression failed: %v", err)
			}
			if cronExpr != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, cronExpr)
			}
		})
	}
}

func TestScheduleBuilderWithTime(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)

	timeTests := []struct {
		name     string
		builder  func() *ScheduleBuilder
		expected string
	}{
		{
			name:     "daily at 10:30",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Day().At("10:30") },
			expected: "30 10 * * *",
		},
		{
			name:     "daily at 09:15:30",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Day().At("09:15:30") },
			expected: "30 15 9 * * *",
		},
		{
			name:     "Monday at 14:15",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Monday().At("14:15") },
			expected: "15 14 * * 1",
		},
		{
			name:     "weekly at 08:00",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Week().At("08:00") },
			expected: "0 8 * * 0",
		},
	}

	for _, tc := range timeTests {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.builder()
			cronExpr, err := builder.buildCronExpression()
			if err != nil {
				t.Fatalf("buildCronExpression failed: %v", err)
			}
			if cronExpr != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, cronExpr)
			}
		})
	}
}

func TestScheduleBuilderTimeSpecParsing(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)
	builder := scheduler.Every().Day()

	timeParseTests := []struct {
		timeSpec     string
		expectedHour int
		expectedMin  int
		expectedSec  int
		shouldError  bool
	}{
		{"10:30", 10, 30, -1, false},
		{"09:15:30", 9, 15, 30, false},
		{"23:59", 23, 59, -1, false},
		{"00:00", 0, 0, -1, false},
		{"24:00", 0, 0, -1, true},    // invalid hour
		{"10:60", 0, 0, -1, true},    // invalid minute
		{"10", 0, 0, -1, true},       // invalid format
		{"10:30:60", 0, 0, -1, true}, // invalid second
	}

	for _, tc := range timeParseTests {
		t.Run(tc.timeSpec, func(t *testing.T) {
			builder.timeSpec = tc.timeSpec
			hour, minute, second, err := builder.parseTimeSpec()

			if tc.shouldError {
				if err == nil {
					t.Errorf("expected error for timeSpec %q, but got none", tc.timeSpec)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for timeSpec %q: %v", tc.timeSpec, err)
			}

			if hour != tc.expectedHour {
				t.Errorf("expected hour %d, got %d", tc.expectedHour, hour)
			}
			if minute != tc.expectedMin {
				t.Errorf("expected minute %d, got %d", tc.expectedMin, minute)
			}
			if second != tc.expectedSec {
				t.Errorf("expected second %d, got %d", tc.expectedSec, second)
			}
		})
	}
}

func TestScheduleBuilderTaskIDGeneration(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)

	idTests := []struct {
		name     string
		builder  func() *ScheduleBuilder
		expected string
	}{
		{
			name:     "every second",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Second() },
			expected: "every-second",
		},
		{
			name:     "every 10 seconds",
			builder:  func() *ScheduleBuilder { return scheduler.Every(10).Seconds() },
			expected: "every-10-seconds",
		},
		{
			name:     "daily",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Day() },
			expected: "daily",
		},
		{
			name:     "daily at 10:30",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Day().At("10:30") },
			expected: "daily-at-10-30",
		},
		{
			name:     "Monday",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Monday() },
			expected: "monday",
		},
		{
			name:     "Monday at 14:15",
			builder:  func() *ScheduleBuilder { return scheduler.Every().Monday().At("14:15") },
			expected: "monday-at-14-15",
		},
	}

	for _, tc := range idTests {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.builder()
			id := builder.generateTaskID()
			if id != tc.expected {
				t.Errorf("expected task ID %q, got %q", tc.expected, id)
			}
		})
	}
}

func TestScheduleBuilderDoFunction(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)

	// 测试不同类型的job函数
	// func() error
	err := scheduler.Every(10).Seconds().Do(func() error {
		return nil
	}, "test-task-1")
	if err != nil {
		t.Fatalf("failed to add task with func() error: %v", err)
	}

	// func()
	err = scheduler.Every().Minute().Do(func() {
		// task execution
	}, "test-task-2")
	if err != nil {
		t.Fatalf("failed to add task with func(): %v", err)
	}

	// Job interface
	job := WrapJob("wrapped-job", func() error {
		return nil
	})
	err = scheduler.Every().Hour().Do(job, "test-task-3")
	if err != nil {
		t.Fatalf("failed to add task with Job interface: %v", err)
	}

	// 验证任务已添加
	tasks := scheduler.GetTasks()
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}

	// 验证任务ID
	expectedIDs := []string{"test-task-1", "test-task-2", "test-task-3"}
	for i, task := range tasks {
		if task.ID != expectedIDs[i] {
			t.Errorf("expected task ID %q, got %q", expectedIDs[i], task.ID)
		}
	}
}

func TestScheduleBuilderWithOptions(t *testing.T) {
	scheduler := NewScheduler(StorageTypeHeap)

	shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

	// 测试链式选项配置
	err := scheduler.Every().Day().At("09:00:30").
		WithTimeout(30*time.Second).
		WithRetry(3).
		WithLocation(shanghaiLoc).
		Do(func() error { return nil }, "options-test")

	if err != nil {
		t.Fatalf("failed to add task with options: %v", err)
	}

	// 验证任务已添加
	tasks := scheduler.GetTasks()
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.ID != "options-test" {
		t.Errorf("expected task ID %q, got %q", "options-test", task.ID)
	}

	// 验证cron解析器配置
	if task.CronParser.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", task.CronParser.timeout)
	}
	if task.CronParser.retry != 3 {
		t.Errorf("expected retry 3, got %d", task.CronParser.retry)
	}
	if task.CronParser.location != shanghaiLoc {
		t.Errorf("expected location Shanghai, got %v", task.CronParser.location)
	}
}
