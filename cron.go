package golitecron

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidExpr = errors.New("invalid cron expression")
	ErrInvalidStep = errors.New("invalid step value")
)

type FieldType int

const (
	Seconds FieldType = iota
	Minutes
	Hours
	DayOfMonth
	Month
	DayOfWeek
	Year
)

type parseRule struct {
	field     FieldType
	min       int
	max       int
	radix     int
	require   bool
	parseFunc func(string, int, int, int) ([]int, error)
}

var defaultRules = []parseRule{
	{Seconds, 0, 59, 60, false, parseItem},
	{Minutes, 0, 59, 60, true, parseItem},
	{Hours, 0, 23, 24, true, parseItem},
	{DayOfMonth, 1, 31, 31, true, parseItem},
	{Month, 1, 12, 12, true, parseItem},
	{DayOfWeek, 0, 6, 7, true, parseItem},
	{Year, 2025, 2100, 1, false, parseItem},
}

type Cron struct {
	Seconds    map[int]struct{}
	Minutes    map[int]struct{}
	Hours      map[int]struct{}
	DayOfMonth map[int]struct{}
	Month      map[int]struct{}
	DayOfWeek  map[int]struct{}
	Year       map[int]struct{}

	enableSeconds bool
	enableYear    bool
}

type Option func(*Cron)

func EnableSeconds() Option {
	return func(cron *Cron) {
		cron.enableSeconds = true
	}
}

func EnableYear() Option {
	return func(cron *Cron) {
		cron.enableYear = true
	}
}

func NewCron(expr string, opts ...Option) (*Cron, error) {
	c := &Cron{}
	for _, opt := range opts {
		opt(c)
	}

	parsed, err := c.parse(expr)
	if err != nil {
		return nil, err
	}

	fieldMap := map[FieldType]func([]int){
		Seconds:    func(vals []int) { c.Seconds = toMap(vals) },
		Minutes:    func(vals []int) { c.Minutes = toMap(vals) },
		Hours:      func(vals []int) { c.Hours = toMap(vals) },
		DayOfMonth: func(vals []int) { c.DayOfMonth = toMap(vals) },
		Month:      func(vals []int) { c.Month = toMap(vals) },
		DayOfWeek:  func(vals []int) { c.DayOfWeek = toMap(vals) },
		Year:       func(vals []int) { c.Year = toMap(vals) },
	}

	for f, v := range parsed {
		if fun, ok := fieldMap[FieldType(f)]; ok {
			fun(v)
		}
	}

	return c, nil
}

func (c *Cron) parse(cron string) (map[FieldType][]int, error) {
	terms := strings.Fields(cron)

	rules := make([]parseRule, 0, len(defaultRules))
	for _, rule := range defaultRules {
		if rule.field == Seconds && !c.enableSeconds || rule.field == Year && !c.enableYear {
			continue
		}
		rules = append(rules, rule)
	}

	if len(terms) != len(rules) {
		return nil, fmt.Errorf("invalid cron expression format: expected %d terms, got %d", len(rules), len(terms))
	}

	result := make(map[FieldType][]int, len(rules))

	for i, term := range terms {
		rule := rules[i]
		vals, err := rule.parseFunc(term, rule.min, rule.max, rule.radix)
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			return nil, ErrInvalidExpr
		}
		result[rule.field] = vals
	}

	return result, nil
}

func toMap(vals []int) map[int]struct{} {
	res := make(map[int]struct{})
	for _, v := range vals {
		res[v] = struct{}{}
	}
	return res
}

func parseItem(expr string, min, max, radix int) ([]int, error) {
	var res []int
	if expr == "*" || expr == "?" {
		for i := min; i <= max; i++ {
			res = append(res, i)
		}
		return res, nil
	}

	if strings.Contains(expr, "/") {
		items := strings.Split(expr, "/")
		if len(items) != 2 {
			return nil, ErrInvalidExpr
		}

		step, err := strconv.Atoi(items[1])
		if err != nil {
			return nil, err
		}
		if step <= 0 {
			return nil, ErrInvalidStep
		}

		if radix%step != 0 {
			return nil, ErrInvalidStep
		}

		parsed, err := parseItem(items[0], min, max, radix)
		if err != nil {
			return nil, err
		}

		for i := min; i <= max; i = i + step {
			if slices.Contains(parsed, i) {
				res = append(res, i)
			}
		}

		return res, nil
	}

	if strings.Contains(expr, ",") {
		items := strings.Split(expr, ",")
		for _, item := range items {
			v, err := strconv.Atoi(item)
			if err != nil {
				return nil, err
			}
			if v < min || v > max {
				return nil, ErrInvalidExpr
			}

			res = append(res, v)
		}

		return res, nil
	}

	if strings.Contains(expr, "-") {
		items := strings.Split(expr, "-")
		if len(items) != 2 {
			return nil, ErrInvalidExpr
		}
		start, err := strconv.Atoi(items[0])
		if err != nil {
			return nil, err
		}
		end, err := strconv.Atoi(items[1])
		if err != nil {
			return nil, err
		}
		if start < min {
			return nil, ErrInvalidExpr
		}
		if end > max {
			return nil, ErrInvalidExpr
		}

		for i := start; i <= end; i++ {
			res = append(res, i)
		}

		return res, nil
	}

	return nil, nil
}
