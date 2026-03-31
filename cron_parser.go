package golitecron

import (
	"fmt"
	"sort"
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

	// Pre-sorted slices for field-jumping algorithm in Next().
	sortedSeconds []int
	sortedMinutes []int
	sortedHours   []int
	sortedMonths  []int
	sortedYears   []int

	// Track if dayOfMonth/dayOfWeek are wildcards for OR/AND logic.
	// In standard cron, if both are specified (non-wildcard), they use OR logic.
	dayOfMonthWildcard bool
	dayOfWeekWildcard  bool
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

		// Track wildcards for dayOfMonth/dayOfWeek OR logic
		isWildcard := (part == "*" || part == "?")
		switch rule.field {
		case DayOfMonth:
			parser.dayOfMonthWildcard = isWildcard
		case DayOfWeek:
			parser.dayOfWeekWildcard = isWildcard
		}
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

	// Check "/" before "-" because "10-30/5" contains both but should be handled as step
	if strings.Contains(field, "/") {
		parts := strings.Split(field, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid step format: %s", field)
		}

		step, err := strconv.Atoi(parts[1])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step value: %s", parts[1])
		}

		// Determine the start and end values for the step
		// Supports: */step, start-end/step, or just number/step
		base := parts[0]
		start := min
		end := max

		if base == "*" || base == "?" {
			// */step: start from min, go to max
			start = min
			end = max
		} else if strings.Contains(base, "-") {
			// start-end/step: parse the range
			rangeParts := strings.Split(base, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format in step expression: %s", field)
			}
			start, err = strconv.Atoi(rangeParts[0])
			if err != nil || start < min || start > max {
				return nil, fmt.Errorf("invalid range start in step expression: %s", rangeParts[0])
			}
			end, err = strconv.Atoi(rangeParts[1])
			if err != nil || end < min || end > max {
				return nil, fmt.Errorf("invalid range end in step expression: %s", rangeParts[1])
			}
			if start > end {
				return nil, fmt.Errorf("range start cannot be greater than end: %s", field)
			}
		} else {
			// number/step: start from the number
			start, err = strconv.Atoi(base)
			if err != nil || start < min || start > max {
				return nil, fmt.Errorf("invalid start value in step expression: %s", base)
			}
			end = max
		}

		// Generate values from start to end with given step
		// No need to check if (end-start+1) % step == 0, standard cron allows any step
		result := make(map[int]struct{})
		for i := start; i <= end; i += step {
			result[i] = struct{}{}
		}

		return result, nil
	}

	// Pure range without step (e.g., "10-30")
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
	if p.enableSeconds && len(p.seconds) == 0 {
		p.seconds = map[int]struct{}{0: {}}
	}
	// Pre-sort field values for field-jumping algorithm in Next().
	p.sortedSeconds = sortedKeys(p.seconds)
	p.sortedMinutes = sortedKeys(p.minutes)
	p.sortedHours = sortedKeys(p.hours)
	p.sortedMonths = sortedKeys(p.months)
	if p.enableYears {
		p.sortedYears = sortedKeys(p.years)
	}
}

func sortedKeys(m map[int]struct{}) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// Next returns the next time after t that matches the cron expression.
// Uses a field-jumping algorithm: processes fields from most significant (year)
// to least significant (second), jumping directly to the next valid value.
// Complexity: O(F × V) where F=number of fields, V=max values per field.
// Typical iterations: 10~50 instead of millions with brute force.
func (p *CronParser) Next(t time.Time) time.Time {
	t = t.In(p.location)
	if p.enableSeconds {
		t = t.Add(time.Second).Truncate(time.Second)
	} else {
		t = t.Add(time.Minute).Truncate(time.Minute)
	}

	year := t.Year()
	month := int(t.Month())
	day := t.Day()
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()

	// Safety bound: search at most 5 years (covers 4-year leap cycle + margin).
	maxYear := year + 5

	for iteration := 0; iteration < 25; iteration++ {
		// Step 1: Year
		if p.enableYears {
			y, found := nextInSorted(p.sortedYears, year)
			if !found || y > maxYear {
				return time.Time{}
			}
			if y != year {
				year = y
				month = p.sortedMonths[0]
				day = 1
				hour = p.sortedHours[0]
				minute = p.sortedMinutes[0]
				second = p.firstSecond()
			}
		} else if year > maxYear {
			return time.Time{}
		}

		// Step 2: Month
		m, found := nextInSorted(p.sortedMonths, month)
		if !found {
			year++
			month = p.sortedMonths[0]
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if m != month {
			month = m
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
		}

		// Step 3: Day (special handling for L/W and OR logic)
		lastDay := daysInMonth(year, time.Month(month))
		if day > lastDay {
			month++
			if month > 12 {
				month = 1
				year++
			}
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		d, dayFound := p.nextValidDay(year, time.Month(month), day)
		if !dayFound {
			month++
			if month > 12 {
				month = 1
				year++
			}
			day = 1
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if d != day {
			day = d
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
		}

		// Step 4: Hour
		h, found := nextInSorted(p.sortedHours, hour)
		if !found {
			day++
			hour = p.sortedHours[0]
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if h != hour {
			hour = h
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
		}

		// Step 5: Minute
		mi, found := nextInSorted(p.sortedMinutes, minute)
		if !found {
			hour++
			minute = p.sortedMinutes[0]
			second = p.firstSecond()
			continue
		}
		if mi != minute {
			minute = mi
			second = p.firstSecond()
		}

		// Step 6: Second (only when seconds precision is enabled)
		if p.enableSeconds {
			s, found := nextInSorted(p.sortedSeconds, second)
			if !found {
				minute++
				second = p.sortedSeconds[0]
				continue
			}
			second = s
		}

		// Construct result and validate the date is real
		// (e.g., Feb 30 would overflow to March in time.Date).
		result := time.Date(year, time.Month(month), day, hour, minute, second, 0, p.location)
		if result.Year() == year && result.Month() == time.Month(month) && result.Day() == day {
			return result
		}
		// Date overflow: advance to next month.
		month++
		if month > 12 {
			month = 1
			year++
		}
		day = 1
		hour = p.sortedHours[0]
		minute = p.sortedMinutes[0]
		second = p.firstSecond()
	}

	return time.Time{}
}

// nextInSorted finds the smallest value >= val in a sorted slice.
// Returns (value, true) if found, or (0, false) if val exceeds all elements.
func nextInSorted(sorted []int, val int) (int, bool) {
	idx := sort.SearchInts(sorted, val)
	if idx < len(sorted) {
		return sorted[idx], true
	}
	return 0, false
}

// firstSecond returns the first valid second value, or 0 if seconds are not enabled.
func (p *CronParser) firstSecond() int {
	if p.enableSeconds && len(p.sortedSeconds) > 0 {
		return p.sortedSeconds[0]
	}
	return 0
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// nextValidDay finds the next day >= startDay that satisfies the dayOfMonth/dayOfWeek
// constraints in the given year/month. Returns (day, true) or (0, false) if none found.
func (p *CronParser) nextValidDay(year int, month time.Month, startDay int) (int, bool) {
	lastDay := daysInMonth(year, month)
	for day := startDay; day <= lastDay; day++ {
		if p.isDayValid(year, month, day) {
			return day, true
		}
	}
	return 0, false
}

// isDayValid checks if a day satisfies the combined dayOfMonth/dayOfWeek constraints,
// applying standard cron OR/AND logic.
func (p *CronParser) isDayValid(year int, month time.Month, day int) bool {
	if p.dayOfMonthWildcard && p.dayOfWeekWildcard {
		return true
	}

	domValid := p.isDayOfMonthMatch(year, month, day)
	dowValid := p.isDayOfWeekMatch(year, month, day)

	if p.dayOfMonthWildcard {
		return dowValid
	}
	if p.dayOfWeekWildcard {
		return domValid
	}
	// Both specified: OR logic (standard cron behavior).
	return domValid || dowValid
}

// isDayOfMonthMatch checks if day matches the dayOfMonth field,
// handling L (last day) and W (nearest weekday) special values.
func (p *CronParser) isDayOfMonthMatch(year int, month time.Month, day int) bool {
	for d := range p.dayOfMonth {
		switch {
		case d == 0: // L: last day of month
			if day == daysInMonth(year, month) {
				return true
			}
		case d < 0: // W: nearest weekday to day -d
			if day == findNearestWeekday(year, month, -d, p.location) {
				return true
			}
		default:
			if day == d {
				return true
			}
		}
	}
	return false
}

// isDayOfWeekMatch checks if day matches the dayOfWeek field,
// handling nL (last nth weekday of month) special values.
func (p *CronParser) isDayOfWeekMatch(year int, month time.Month, day int) bool {
	weekday := int(time.Date(year, month, day, 0, 0, 0, 0, p.location).Weekday())
	for w := range p.dayOfWeek {
		if w < 0 {
			targetWeekday := -w
			lastDay := findLastWeekdayOfMonth(year, month, targetWeekday, p.location)
			if day == lastDay {
				return true
			}
		} else if weekday == w {
			return true
		}
	}
	return false
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
