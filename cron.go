package golitecron

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidExpr = errors.New("invalid cron expression")
	ErrInvalidStep = errors.New("invalid step value")
)

type CronExpr struct {
	Seconds    map[int]struct{}
	Minutes    map[int]struct{}
	Hours      map[int]struct{}
	DayOfMonth map[int]struct{}
	Month      map[int]struct{}
	DayOfWeek  map[int]struct{}
	Year       map[int]struct{}
}

func NewCron(expr string) *CronExpr {
	cron, err := parse(expr)
	if err != nil {
		return nil
	}
	return cron
}

func parse(cron string) (*CronExpr, error) {
	terms := strings.Fields(cron)

	if len(terms) != 6 {
		return nil, ErrInvalidExpr
	}

	seconds, err := parseExpr(terms[0], 0, 59, 60)
	if err != nil {
		return nil, err
	}
	minutes, err := parseExpr(terms[1], 0, 59, 60)
	if err != nil {
		return nil, err
	}
	hours, err := parseExpr(terms[2], 0, 23, 24)
	if err != nil {
		return nil, err
	}
	dayOfMonth, err := parseExpr(terms[3], 1, 31, 31)
	if err != nil {
		return nil, err
	}
	month, err := parseExpr(terms[4], 1, 12, 12)
	if err != nil {
		return nil, err
	}
	dayOfWeek, err := parseExpr(terms[5], 0, 6, 7)
	if err != nil {
		return nil, err
	}
	year, err := parseExpr(terms[6], 2025, 2100, 1)
	if err != nil {
		return nil, err
	}

	cronExpr := &CronExpr{
		Seconds:    seconds,
		Minutes:    minutes,
		Hours:      hours,
		DayOfMonth: dayOfMonth,
		Month:      month,
		DayOfWeek:  dayOfWeek,
		Year:       year,
	}

	return cronExpr, nil
}

func parseExpr(expr string, min, max, radix int) (map[int]struct{}, error) {
	parsed, err := parseItem(expr, min, max, radix)
	if err != nil {
		return nil, err
	}
	res := make(map[int]struct{})
	for _, v := range parsed {
		res[v] = struct{}{}
	}
	return res, nil
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
