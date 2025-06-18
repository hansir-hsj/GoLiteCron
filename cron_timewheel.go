package golitecron

// import (
// 	"container/list"
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"sync"
// 	"time"
// )

// // CronField 表示Cron表达式中的一个字段
// type CronField struct {
// 	Values map[int]struct{} // 允许的值
// 	Min    int              // 最小值
// 	Max    int              // 最大值
// }

// // CronExpr 表示解析后的Cron表达式
// type CronExpr struct {
// 	Second CronField
// 	Minute CronField
// 	Hour   CronField
// 	Day    CronField
// 	Month  CronField
// 	Week   CronField
// }

// // Task 表示一个定时任务
// type Task struct {
// 	Expr     *CronExpr
// 	Callback func()
// 	Name     string
// }

// // TimeWheel 表示一个时间轮
// type TimeWheel struct {
// 	tickDuration time.Duration // 每个槽的时间间隔
// 	wheelSize    int           // 时间轮大小（槽数量）
// 	slots        []*list.List  // 槽数组
// 	currentTick  int           // 当前槽索引
// 	tasks        map[string]*Task
// 	mu           sync.RWMutex
// 	stopChan     chan struct{}
// 	wg           sync.WaitGroup
// }

// // NewTimeWheel 创建一个新的时间轮
// func NewTimeWheel(tickDuration time.Duration, wheelSize int) *TimeWheel {
// 	slots := make([]*list.List, wheelSize)
// 	for i := 0; i < wheelSize; i++ {
// 		slots[i] = list.New()
// 	}

// 	return &TimeWheel{
// 		tickDuration: tickDuration,
// 		wheelSize:    wheelSize,
// 		slots:        slots,
// 		currentTick:  0,
// 		tasks:        make(map[string]*Task),
// 		stopChan:     make(chan struct{}),
// 	}
// }

// // Start 启动时间轮
// func (tw *TimeWheel) Start() {
// 	tw.wg.Add(1)
// 	go tw.run()
// }

// // Stop 停止时间轮
// func (tw *TimeWheel) Stop() {
// 	close(tw.stopChan)
// 	tw.wg.Wait()
// }

// // AddTask 添加一个Cron任务
// func (tw *TimeWheel) AddTask(name string, exprStr string, callback func()) error {
// 	expr, err := ParseCronExpr(exprStr)
// 	if err != nil {
// 		return err
// 	}

// 	task := &Task{
// 		Expr:     expr,
// 		Callback: callback,
// 		Name:     name,
// 	}

// 	tw.mu.Lock()
// 	defer tw.mu.Unlock()

// 	// 计算任务首次执行的时间
// 	nextTime := calculateNextExecutionTime(time.Now(), expr)
// 	delay := nextTime.Sub(time.Now())

// 	// 将任务添加到时间轮中
// 	tw.scheduleTask(task, delay)
// 	tw.tasks[name] = task

// 	return nil
// }

// // RemoveTask 移除一个任务
// func (tw *TimeWheel) RemoveTask(name string) {
// 	tw.mu.Lock()
// 	defer tw.mu.Unlock()

// 	delete(tw.tasks, name)
// 	// 注意：这里没有从槽中移除任务，实际实现中需要改进
// }

// // scheduleTask 将任务安排到指定的延迟后执行
// func (tw *TimeWheel) scheduleTask(task *Task, delay time.Duration) {
// 	// 计算需要等待的tick数
// 	ticks := int(delay / tw.tickDuration)
// 	if ticks < 1 {
// 		ticks = 1
// 	}

// 	// 计算任务应该放入的槽索引
// 	slotIndex := (tw.currentTick + ticks) % tw.wheelSize

// 	// 将任务添加到对应槽
// 	tw.slots[slotIndex].PushBack(task)
// }

// // run 时间轮主循环
// func (tw *TimeWheel) run() {
// 	defer tw.wg.Done()

// 	ticker := time.NewTicker(tw.tickDuration)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			tw.advanceClock()
// 		case <-tw.stopChan:
// 			return
// 		}
// 	}
// }

// // advanceClock 推进时钟，处理当前槽中的任务
// func (tw *TimeWheel) advanceClock() {
// 	tw.mu.Lock()
// 	defer tw.mu.Unlock()

// 	// 获取当前槽
// 	slot := tw.slots[tw.currentTick]

// 	// 复制任务列表，避免在执行过程中修改
// 	tasksToExecute := make([]*Task, 0, slot.Len())
// 	for e := slot.Front(); e != nil; e = e.Next() {
// 		task := e.Value.(*Task)
// 		tasksToExecute = append(tasksToExecute, task)
// 	}
// 	slot.Init() // 清空当前槽

// 	// 执行所有到期的任务
// 	now := time.Now()
// 	for _, task := range tasksToExecute {
// 		// 执行任务
// 		go task.Callback()

// 		// 计算下一次执行时间并重新调度
// 		nextTime := calculateNextExecutionTime(now, task.Expr)
// 		delay := nextTime.Sub(now)
// 		tw.scheduleTask(task, delay)
// 	}

// 	// 移动到下一个槽
// 	tw.currentTick = (tw.currentTick + 1) % tw.wheelSize
// }

// // calculateNextExecutionTime 计算Cron表达式的下一次执行时间
// func calculateNextExecutionTime(now time.Time, expr *CronExpr) time.Time {
// 	// 简化实现，实际应该递归处理每个字段
// 	second := now.Second()
// 	minute := now.Minute()
// 	hour := now.Hour()
// 	day := now.Day()
// 	month := int(now.Month())
// 	year := now.Year()

// 	// 简单起见，这里只做了基本实现
// 	// 实际的Cron解析器需要处理更多复杂情况
// 	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local).Add(time.Second)
// }

// // ParseCronExpr 解析Cron表达式
// func ParseCronExpr(exprStr string) (*CronExpr, error) {
// 	fields := strings.Fields(exprStr)
// 	if len(fields) != 6 {
// 		return nil, fmt.Errorf("无效的Cron表达式: %s，需要6个字段", exprStr)
// 	}

// 	return &CronExpr{
// 		Second: parseField(fields[0], 0, 59),
// 		Minute: parseField(fields[1], 0, 59),
// 		Hour:   parseField(fields[2], 0, 23),
// 		Day:    parseField(fields[3], 1, 31),
// 		Month:  parseField(fields[4], 1, 12),
// 		Week:   parseField(fields[5], 0, 6),
// 	}, nil
// }

// // parseField 解析单个Cron字段
// func parseField(field string, min, max int) CronField {
// 	values := make(map[int]struct{})

// 	if field == "*" {
// 		// 匹配所有值
// 		for i := min; i <= max; i++ {
// 			values[i] = struct{}{}
// 		}
// 		return CronField{Values: values, Min: min, Max: max}
// 	}

// 	// 解析逗号分隔的值
// 	parts := strings.Split(field, ",")
// 	for _, part := range parts {
// 		if strings.Contains(part, "-") {
// 			// 解析范围
// 			rangeParts := strings.Split(part, "-")
// 			if len(rangeParts) == 2 {
// 				start, _ := strconv.Atoi(rangeParts[0])
// 				end, _ := strconv.Atoi(rangeParts[1])
// 				for i := start; i <= end; i++ {
// 					values[i] = struct{}{}
// 				}
// 			}
// 		} else if strings.Contains(part, "/") {
// 			// 解析间隔
// 			stepParts := strings.Split(part, "/")
// 			if len(stepParts) == 2 {
// 				startStr := stepParts[0]
// 				step, _ := strconv.Atoi(stepParts[1])

// 				start := min
// 				if startStr != "*" {
// 					start, _ = strconv.Atoi(startStr)
// 				}

// 				for i := start; i <= max; i += step {
// 					values[i] = struct{}{}
// 				}
// 			}
// 		} else {
// 			// 解析单个值
// 			if num, err := strconv.Atoi(part); err == nil {
// 				values[num] = struct{}{}
// 			}
// 		}
// 	}

// 	return CronField{Values: values, Min: min, Max: max}
// }
