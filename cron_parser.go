package golitecron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FieldType int

const (
	Seconds FieldType = iota
	Minutes
	Hours
	DayOfMonth
	Months
	DayOfWeek
	Years
)

type parseRule struct {
	field     FieldType
	min       int
	max       int
	parseFunc func(string, int, int, FieldType) (map[int]struct{}, error)
}

type CronParser struct {
	seconds    map[int]struct{}
	minutes    map[int]struct{}
	hours      map[int]struct{}
	dayOfMonth map[int]struct{}
	months     map[int]struct{}
	dayOfWeek  map[int]struct{}
	years      map[int]struct{}

	enableSeconds bool
	enableYears   bool
	location      *time.Location
	timeout       time.Duration
	retry         int
}

type Option func(*CronParser)

func WithSeconds() Option {
	return func(p *CronParser) {
		p.enableSeconds = true
	}
}

func WithYears() Option {
	return func(p *CronParser) {
		p.enableYears = true
	}
}

func WithLocation(loc *time.Location) Option {
	return func(p *CronParser) {
		p.location = loc
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(p *CronParser) {
		if timeout < 0 {
			timeout = 0
		}
		p.timeout = timeout
	}
}

func WithRetry(retry int) Option {
	return func(p *CronParser) {
		if retry < 0 {
			retry = 0
		}
		p.retry = retry
	}
}

var defaultRules = []parseRule{
	{Seconds, 0, 59, parseField},
	{Minutes, 0, 59, parseField},
	{Hours, 0, 23, parseField},
	{DayOfMonth, 1, 31, parseField},
	{Months, 1, 12, parseField},
	{DayOfWeek, 0, 6, parseField},
	{Years, 1970, 2099, parseField},
}

func newCronParser(expr string, opts ...Option) (*CronParser, error) {
	if strings.HasPrefix(expr, "@") {
		switch expr {
		case "@yearly", "@annually":
			expr = Yearly
		case "@monthly":
			expr = Monthly
		case "@weekly":
			expr = Weekly
		case "@daily", "@midnight":
			expr = Daily
		case "@hourly":
			expr = Hourly
		case "@minutely":
			expr = Minutely
		}
	}

	parser := &CronParser{
		location: time.Local,
	}
	for _, opt := range opts {
		opt(parser)
	}

	parts := strings.Fields(expr)
	rules := make([]parseRule, 0, len(defaultRules))
	for _, rule := range defaultRules {
		if rule.field == Seconds && !parser.enableSeconds ||
			rule.field == Years && !parser.enableYears {
			continue
		}
		rules = append(rules, rule)
	}

	if len(parts) != len(rules) {
		return nil, fmt.Errorf("invalid cron expression length: expected %d fields, got %d", len(rules), len(parts))
	}

	parsed := make(map[FieldType]map[int]struct{}, len(parts))
	for i, part := range parts {
		rule := rules[i]
		vals, err := rule.parseFunc(part, rule.min, rule.max, rule.field)
		if err != nil {
			return nil, fmt.Errorf("error parsing field %d (%s): %v", i, part, err)
		}
		if len(vals) == 0 {
			return nil, fmt.Errorf("invalid field %d (%s)", i, part)
		}
		parsed[rule.field] = vals
	}

	fieldMap := map[FieldType]func(map[int]struct{}){
		Seconds:    func(vals map[int]struct{}) { parser.seconds = vals },
		Minutes:    func(vals map[int]struct{}) { parser.minutes = vals },
		Hours:      func(vals map[int]struct{}) { parser.hours = vals },
		DayOfMonth: func(vals map[int]struct{}) { parser.dayOfMonth = vals },
		Months:     func(vals map[int]struct{}) { parser.months = vals },
		DayOfWeek:  func(vals map[int]struct{}) { parser.dayOfWeek = vals },
		Years:      func(vals map[int]struct{}) { parser.years = vals },
	}

	for f, v := range parsed {
		fieldMap[f](v)
	}

	parser.normalization()

	return parser, nil
}

func parseField(field string, min, max int, fieldType FieldType) (map[int]struct{}, error) {
	if field == "*" || field == "?" {
		result := make(map[int]struct{}, max-min+1)
		for i := min; i <= max; i++ {
			result[i] = struct{}{}
		}
		return result, nil
	}

	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		result := make(map[int]struct{})
		for _, part := range parts {
			nums, err := parseField(part, min, max, fieldType)
			if err != nil {
				return nil, err
			}
			for num := range nums {
				result[num] = struct{}{}
			}
		}
		return result, nil
	}

	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range format: %s", field)
		}

		start, err := strconv.Atoi(parts[0])
		if err != nil || start < min || start > max {
			return nil, fmt.Errorf("invalid range start: %s", parts[0])
		}

		end, err := strconv.Atoi(parts[1])
		if err != nil || end < min || end > max {
			return nil, fmt.Errorf("invalid range end: %s", parts[1])
		}

		if start > end {
			return nil, fmt.Errorf("range start cannot be greater than end: %s", field)
		}

		result := make(map[int]struct{}, end-start+1)
		for i := start; i <= end; i++ {
			result[i] = struct{}{}
		}

		return result, nil
	}

	if strings.Contains(field, "/") {
		parts := strings.Split(field, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid step format: %s", field)
		}

		step, err := strconv.Atoi(parts[1])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step value: %s", parts[1])
		}

		radix := max - min + 1
		if radix%step != 0 {
			return nil, fmt.Errorf("invalid step value: %s", parts[1])
		}

		result := make(map[int]struct{})
		for i := min; i <= max; i += step {
			result[i] = struct{}{}
		}

		return result, nil
	}

	if strings.Contains(field, "L") {
		if len(field) > 1 && !strings.HasSuffix(field, "L") {
			return nil, fmt.Errorf("invalid 'L' format: %s", field)
		}
		if len(field) > 1 {
			numStr := field[:len(field)-1]
			num, err := strconv.Atoi(numStr)
			if err != nil || num < min || num > max {
				return nil, fmt.Errorf("invalid 'L' number: %s", numStr)
			}
			// For DayOfWeek, 'L' means the last occurrence of the day in the month
			// We represent it as negative to differentiate from normal days
			// e.g., 5L means the last Friday of the month
			// This requires special handling in the scheduling logic
			// Here we just return the negative value to indicate this
			// The actual calculation will be done in the Next function
			if fieldType == DayOfWeek {
				return map[int]struct{}{-num: {}}, nil
			}
		}
		if fieldType == DayOfMonth {
			// 0 indicates the last day of the month
			return map[int]struct{}{0: {}}, nil
		}
		return nil, fmt.Errorf("expression L not allowed in this field: %s", field)
	}

	if strings.Contains(field, "W") {
		if fieldType != DayOfMonth {
			return nil, fmt.Errorf("expression W only allowed in DayOfMonth field: %s", field)
		}
		if !strings.HasSuffix(field, "W") || len(field) < 2 {
			return nil, fmt.Errorf("invalid 'W' format: %s", field)
		}
		numStr := field[:len(field)-1]
		num, err := strconv.Atoi(numStr)
		if err != nil || num < min || num > max {
			return nil, fmt.Errorf("invalid 'W' number: %s", numStr)
		}
		// 'W' means the nearest weekday (Monday to Friday) to the given day of the month
		// -num indicates the nearest weekday
		return map[int]struct{}{-num: {}}, nil
	}

	num, err := strconv.Atoi(field)
	if err != nil || num < min || num > max {
		return nil, fmt.Errorf("invalid number: %s", field)
	}

	return map[int]struct{}{num: {}}, nil
}

func (p *CronParser) normalization() {
	if len(p.seconds) == 0 {
		p.seconds = map[int]struct{}{0: {}}
	}
}

func (p *CronParser) Next(t time.Time) time.Time {
	t = t.In(p.location)
	next := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())

	for {
		next = next.Add(time.Second)
		year := next.Year()
		month := next.Month()
		day := next.Day()
		hour := next.Hour()
		minute := next.Minute()
		second := next.Second()
		weekday := int(next.Weekday())

		var dayOfMonthValid bool
		for d := range p.dayOfMonth {
			if d == 0 {
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, next.Location()).Day()
				if day == lastDay {
					dayOfMonthValid = true
					break
				}
			} else if d < 0 {
				targetDay := -d
				nearestWeekday := findNearestWeekday(year, month, targetDay, next.Location())
				if day == nearestWeekday {
					dayOfMonthValid = true
					break
				}
			} else if day == d {
				dayOfMonthValid = true
				break
			}
		}

		var dayOfWeekValid bool
		for w := range p.dayOfWeek {
			if w < 0 {
				targetWeekday := -w
				lastWeekday := findLastWeekdayOfMonth(year, month, targetWeekday, next.Location())
				if day == lastWeekday {
					dayOfWeekValid = true
					break
				}
			} else if weekday == w {
				dayOfWeekValid = true
				break
			}
		}

		if dayOfMonthValid && dayOfWeekValid &&
			contains(p.seconds, second) &&
			(!p.enableYears || contains(p.years, year)) &&
			contains(p.minutes, minute) &&
			contains(p.hours, hour) &&
			contains(p.months, int(month)) {
			return next
		}
	}
}

func contains(m map[int]struct{}, value int) bool {
	_, exists := m[value]
	return exists
}

// findNearestWeekday finds the nearest weekday (Monday to Friday) to the target day in the given month and year.
// If the target day is a Saturday, it returns the previous Friday (if possible) or the next Monday.
// If the target day is a Sunday, it returns the next Monday (if possible) or the previous Friday.
// If the target day is a weekday, it returns the target day itself.
// If the target day is out of range for the month, it returns -1.
func findNearestWeekday(year int, month time.Month, targetDay int, loc *time.Location) int {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if targetDay < 1 || targetDay > lastDay {
		return -1
	}

	t := time.Date(year, month, targetDay, 0, 0, 0, 0, loc)
	wd := t.Weekday()

	if wd >= time.Monday && wd <= time.Friday {
		return targetDay
	}

	if wd == time.Saturday {
		if targetDay-1 >= 1 {
			return targetDay - 1
		}
		return targetDay + 2
	}

	if wd == time.Sunday {
		if targetDay+1 <= lastDay {
			return targetDay + 1
		}
		return targetDay - 2
	}

	return targetDay
}

// findLastWeekdayOfMonth finds the last occurrence of the target weekday (0=Sunday, 1=Monday, ..., 6=Saturday)
// in the given month and year. If not found, it returns -1.
func findLastWeekdayOfMonth(year int, month time.Month, targetWeekday int, loc *time.Location) int {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	for day := lastDay; day >= 1; day-- {
		t := time.Date(year, month, day, 0, 0, 0, 0, loc)
		if int(t.Weekday()) == targetWeekday {
			return day
		}
	}
	return -1
}
