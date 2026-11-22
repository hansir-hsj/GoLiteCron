package golitecron

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type fieldType int

const (
	secondsField fieldType = iota
	minutesField
	hoursField
	dayOfMonthField
	monthsField
	dayOfWeekField
	yearsField
)

type parseRule struct {
	field     fieldType
	min       int
	max       int
	parseFunc func(string, int, int, fieldType) (map[int]struct{}, error)
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
	// Pre-sorted slices for Next() field-jumping algorithm.
	sortedSeconds []int
	sortedMinutes []int
	sortedHours   []int
	sortedMonths  []int
	sortedYears   []int

	// Track wildcards for dayOfMonth/dayOfWeek OR logic.
	dayOfMonthWildcard bool
	dayOfWeekWildcard  bool
}

// ParseOption configures cron expression parsing.
type ParseOption interface {
	applyParseOption(*taskSettings)
}

// TaskOption configures scheduled task parsing and execution policy.
type TaskOption interface {
	applyTaskOption(*taskSettings)
}

// ScheduleOption can be used for both parsing and task scheduling.
type ScheduleOption interface {
	ParseOption
	TaskOption
}

type taskSettings struct {
	enableSeconds bool
	enableYears   bool
	location      *time.Location
	timeout       time.Duration
	retry         int
}

type secondsOption struct{}

func (secondsOption) applyParseOption(settings *taskSettings) {
	settings.enableSeconds = true
}

func (secondsOption) applyTaskOption(settings *taskSettings) {
	settings.enableSeconds = true
}

type yearsOption struct{}

func (yearsOption) applyParseOption(settings *taskSettings) {
	settings.enableYears = true
}

func (yearsOption) applyTaskOption(settings *taskSettings) {
	settings.enableYears = true
}

type locationOption struct {
	location *time.Location
}

func (opt locationOption) applyParseOption(settings *taskSettings) {
	if opt.location == nil {
		return
	}
	settings.location = opt.location
}

func (opt locationOption) applyTaskOption(settings *taskSettings) {
	opt.applyParseOption(settings)
}

type timeoutOption struct {
	timeout time.Duration
}

func (opt timeoutOption) applyTaskOption(settings *taskSettings) {
	timeout := opt.timeout
	if timeout < 0 {
		timeout = 0
	}
	settings.timeout = timeout
}

type retryOption struct {
	retry int
}

func (opt retryOption) applyTaskOption(settings *taskSettings) {
	retry := opt.retry
	if retry < 0 {
		retry = 0
	}
	settings.retry = retry
}

func defaultTaskSettings() taskSettings {
	return taskSettings{
		location: time.Local,
	}
}

func newParseSettings(opts ...ParseOption) taskSettings {
	settings := defaultTaskSettings()
	for _, opt := range opts {
		opt.applyParseOption(&settings)
	}
	return settings
}

func newTaskSettings(opts ...TaskOption) taskSettings {
	settings := defaultTaskSettings()
	for _, opt := range opts {
		opt.applyTaskOption(&settings)
	}
	return settings
}

func WithSeconds() ScheduleOption {
	return secondsOption{}
}

func WithYears() ScheduleOption {
	return yearsOption{}
}

func WithLocation(loc *time.Location) ScheduleOption {
	return locationOption{location: loc}
}

func WithTimeout(timeout time.Duration) TaskOption {
	return timeoutOption{timeout: timeout}
}

func WithRetry(retry int) TaskOption {
	return retryOption{retry: retry}
}

func newCronParserFromSettings(expr string, settings taskSettings) (*CronParser, error) {
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
		default:
			return nil, fmt.Errorf("unknown cron macro: %s", expr)
		}
	}

	parser := &CronParser{
		enableSeconds: settings.enableSeconds,
		enableYears:   settings.enableYears,
		location:      settings.location,
	}

	parts := strings.Fields(expr)
	rules := make([]parseRule, 0, len(defaultRules))
	for _, rule := range defaultRules {
		if rule.field == secondsField && !parser.enableSeconds ||
			rule.field == yearsField && !parser.enableYears {
			continue
		}
		rules = append(rules, rule)
	}

	if len(parts) != len(rules) {
		return nil, fmt.Errorf("invalid cron expression length: expected %d fields, got %d", len(rules), len(parts))
	}

	parsed := make(map[fieldType]map[int]struct{}, len(parts))
	for i, rawPart := range parts {
		rule := rules[i]
		part := normalizeFieldNames(rawPart, rule.field)
		vals, err := rule.parseFunc(part, rule.min, rule.max, rule.field)
		if err != nil {
			return nil, fmt.Errorf("error parsing field %d (%s): %w", i, rawPart, err)
		}
		if len(vals) == 0 {
			return nil, fmt.Errorf("invalid field %d (%s)", i, rawPart)
		}
		if rule.field == dayOfWeekField {
			vals = normalizeDayOfWeek(vals)
		}
		parsed[rule.field] = vals

		isWildcard := part == "*" || part == "?"
		switch rule.field {
		case dayOfMonthField:
			parser.dayOfMonthWildcard = isWildcard
		case dayOfWeekField:
			parser.dayOfWeekWildcard = isWildcard
		}
	}

	fieldMap := map[fieldType]func(map[int]struct{}){
		secondsField:    func(vals map[int]struct{}) { parser.seconds = vals },
		minutesField:    func(vals map[int]struct{}) { parser.minutes = vals },
		hoursField:      func(vals map[int]struct{}) { parser.hours = vals },
		dayOfMonthField: func(vals map[int]struct{}) { parser.dayOfMonth = vals },
		monthsField:     func(vals map[int]struct{}) { parser.months = vals },
		dayOfWeekField:  func(vals map[int]struct{}) { parser.dayOfWeek = vals },
		yearsField:      func(vals map[int]struct{}) { parser.years = vals },
	}

	for f, v := range parsed {
		fieldMap[f](v)
	}

	parser.normalization()

	return parser, nil
}

func newCronParser(expr string, opts ...ParseOption) (*CronParser, error) {
	return newCronParserFromSettings(expr, newParseSettings(opts...))
}

// Parse parses a cron expression into a reusable schedule parser.
func Parse(expr string, opts ...ParseOption) (*CronParser, error) {
	return newCronParser(expr, opts...)
}

var defaultRules = []parseRule{
	{secondsField, 0, 59, parseField},
	{minutesField, 0, 59, parseField},
	{hoursField, 0, 23, parseField},
	{dayOfMonthField, 1, 31, parseField},
	{monthsField, 1, 12, parseField},
	{dayOfWeekField, 0, 7, parseField},
	{yearsField, 1970, 2099, parseField},
}

func normalizeFieldNames(field string, fieldType fieldType) string {
	field = strings.ToUpper(field)
	var names []string
	switch fieldType {
	case monthsField:
		names = []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}
	case dayOfWeekField:
		names = []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}
	default:
		return field
	}
	// Replace whole tokens only (an L suffix is kept), so "JANJAN" stays invalid.
	var b strings.Builder
	start := 0
	rangeEnd := false
	flush := func(end int) {
		token := field[start:end]
		name, suffix := token, ""
		if len(token) > 1 && strings.HasSuffix(token, "L") {
			name, suffix = token[:len(token)-1], "L"
		}
		for value, candidate := range names {
			if name == candidate {
				if fieldType == monthsField {
					value++
				}
				// SUN closing a range means 7, so FRI-SUN works like 5-7.
				if fieldType == dayOfWeekField && value == 0 && rangeEnd && suffix == "" {
					value = 7
				}
				token = strconv.Itoa(value) + suffix
				break
			}
		}
		_, _ = b.WriteString(token)
	}
	for i, r := range field {
		if r == ',' || r == '-' || r == '/' || r == '#' {
			flush(i)
			_, _ = b.WriteRune(r)
			start = i + 1
			rangeEnd = r == '-'
		}
	}
	flush(len(field))
	return b.String()
}

func normalizeDayOfWeek(values map[int]struct{}) map[int]struct{} {
	if _, ok := values[7]; ok {
		delete(values, 7)
		values[0] = struct{}{}
	}
	return values
}

func parseField(field string, min, max int, fieldType fieldType) (map[int]struct{}, error) {
	if field == "*" || field == "?" {
		return parseWildcardField(min, max), nil
	}

	if strings.Contains(field, ",") {
		return parseListField(field, min, max, fieldType)
	}

	// Check "/" before "-" because "10-30/5" contains both but should be handled as step
	if strings.Contains(field, "/") {
		return parseStepField(field, min, max, fieldType)
	}

	// Pure range without step (e.g., "10-30")
	if strings.Contains(field, "-") {
		return parseRangeField(field, min, max)
	}

	if strings.Contains(field, "L") {
		return parseLastField(field, min, max, fieldType)
	}

	if strings.Contains(field, "W") {
		return parseNearestWeekdayField(field, min, max, fieldType)
	}

	num, err := strconv.Atoi(field)
	if err != nil || num < min || num > max {
		return nil, fmt.Errorf("invalid number: %s", field)
	}

	return map[int]struct{}{num: {}}, nil
}

func parseWildcardField(min, max int) map[int]struct{} {
	result := make(map[int]struct{}, max-min+1)
	for i := min; i <= max; i++ {
		result[i] = struct{}{}
	}
	return result
}

func parseListField(field string, min, max int, fieldType fieldType) (map[int]struct{}, error) {
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

func parseStepField(field string, min, max int, fieldType fieldType) (map[int]struct{}, error) {
	parts := strings.Split(field, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid step format: %s", field)
	}

	step, err := strconv.Atoi(parts[1])
	if err != nil || step <= 0 {
		return nil, fmt.Errorf("invalid step value: %s", parts[1])
	}

	start, end, err := stepRange(parts[0], field, min, max)
	if err != nil {
		return nil, err
	}
	// 7 is only a Sunday alias, so open-ended weekday steps stop at Saturday.
	if fieldType == dayOfWeekField && !strings.Contains(parts[0], "-") && start < end && end == 7 {
		end = 6
	}

	result := make(map[int]struct{})
	for i := start; i <= end; i += step {
		result[i] = struct{}{}
	}
	return result, nil
}

func stepRange(base string, field string, min, max int) (int, int, error) {
	if base == "*" || base == "?" {
		return min, max, nil
	}
	if strings.Contains(base, "-") {
		start, end, err := parseRangeBounds(base, min, max, " in step expression")
		if err != nil {
			return 0, 0, err
		}
		if start > end {
			return 0, 0, fmt.Errorf("range start cannot be greater than end: %s", field)
		}
		return start, end, nil
	}

	start, err := strconv.Atoi(base)
	if err != nil || start < min || start > max {
		return 0, 0, fmt.Errorf("invalid start value in step expression: %s", base)
	}
	return start, max, nil
}

func parseRangeField(field string, min, max int) (map[int]struct{}, error) {
	start, end, err := parseRangeBounds(field, min, max, "")
	if err != nil {
		return nil, err
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

func parseRangeBounds(field string, min, max int, context string) (int, int, error) {
	parts := strings.Split(field, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format%s: %s", context, field)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil || start < min || start > max {
		return 0, 0, fmt.Errorf("invalid range start%s: %s", context, parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil || end < min || end > max {
		return 0, 0, fmt.Errorf("invalid range end%s: %s", context, parts[1])
	}
	return start, end, nil
}

func parseLastField(field string, min, max int, fieldType fieldType) (map[int]struct{}, error) {
	if len(field) > 1 && !strings.HasSuffix(field, "L") {
		return nil, fmt.Errorf("invalid 'L' format: %s", field)
	}
	if len(field) > 1 {
		numStr := field[:len(field)-1]
		num, err := strconv.Atoi(numStr)
		if err != nil || num < min || num > max {
			return nil, fmt.Errorf("invalid 'L' number: %s", numStr)
		}
		if fieldType == dayOfWeekField {
			if num == 7 {
				num = 0
			}
			if num == 0 {
				return map[int]struct{}{-7: {}}, nil
			}
			return map[int]struct{}{-num: {}}, nil
		}
		return nil, fmt.Errorf("numeric 'L' is only supported in day-of-week field: %s", field)
	}
	if fieldType == dayOfMonthField {
		return map[int]struct{}{0: {}}, nil
	}
	return nil, fmt.Errorf("expression L not allowed in this field: %s", field)
}

func parseNearestWeekdayField(field string, min, max int, fieldType fieldType) (map[int]struct{}, error) {
	if fieldType != dayOfMonthField {
		return nil, fmt.Errorf("expression W only allowed in day-of-month field: %s", field)
	}
	if !strings.HasSuffix(field, "W") || len(field) < 2 {
		return nil, fmt.Errorf("invalid 'W' format: %s", field)
	}
	numStr := field[:len(field)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil || num < min || num > max {
		return nil, fmt.Errorf("invalid 'W' number: %s", numStr)
	}
	return map[int]struct{}{-num: {}}, nil
}

func (p *CronParser) normalization() {
	if p.enableSeconds && len(p.seconds) == 0 {
		p.seconds = map[int]struct{}{0: {}}
	}
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
