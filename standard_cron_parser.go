package golitecron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type StandardCronParser struct {
	minutes    map[int]struct{}
	hours      map[int]struct{}
	dayOfMonth map[int]struct{}
	months     map[int]struct{}
	dayOfWeek  map[int]struct{}
}

func NewStandardCronParser(expr string) (*StandardCronParser, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid cron expression: %s", expr)
	}

	parser := &StandardCronParser{}
	var err error

	parser.minutes, err = parseField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minutes field: %v", err)
	}

	parser.hours, err = parseField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hours field: %v", err)
	}

	parser.dayOfMonth, err = parseField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day of month field: %v", err)
	}

	parser.months, err = parseField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid months field: %v", err)
	}

	parser.dayOfWeek, err = parseField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day of week field: %v", err)
	}

	return parser, nil
}

func parseField(field string, min, max int) (map[int]struct{}, error) {
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
			nums, err := parseField(part, min, max)
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

	num, err := strconv.Atoi(field)
	if err != nil || num < min || num > max {
		return nil, fmt.Errorf("invalid number: %s", field)
	}

	return map[int]struct{}{num: {}}, nil
}

func (p *StandardCronParser) Next(t time.Time) time.Time {
	next := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location()).Add(time.Minute)

	for {
		if contains(p.minutes, next.Minute()) && contains(p.hours, next.Hour()) &&
			contains(p.dayOfWeek, int(next.Weekday())) &&
			contains(p.months, int(next.Month())) &&
			contains(p.dayOfMonth, next.Day()) {
			return next
		}
		next = next.Add(time.Minute)
	}
}

func contains(m map[int]struct{}, value int) bool {
	_, exists := m[value]
	return exists
}
