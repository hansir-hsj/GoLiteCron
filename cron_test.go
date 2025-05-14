package golitecron

import (
	"testing"
)

func TestParseItem(t *testing.T) {
	cases := []struct {
		expr   string
		min    int
		max    int
		radix  int
		expect []int
	}{
		{
			expr:  "*",
			min:   0,
			max:   59,
			radix: 60,
			expect: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
				11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
				22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
				33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43,
				44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
				55, 56, 57, 58, 59},
		},
		{
			expr:   "*",
			min:    9,
			max:    11,
			radix:  60,
			expect: []int{9, 10, 11},
		},
		{
			expr:   "1,5,10,35",
			min:    0,
			max:    59,
			radix:  60,
			expect: []int{1, 5, 10, 35},
		},
		{
			expr:   "1-10",
			min:    0,
			max:    59,
			radix:  60,
			expect: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			expr:   "*/10",
			min:    0,
			max:    59,
			radix:  60,
			expect: []int{0, 10, 20, 30, 40, 50},
		},
		{
			expr:   "1,5,10,15,20/10",
			min:    0,
			max:    59,
			radix:  60,
			expect: []int{10, 20},
		},
		{
			expr:   "1-20/10",
			min:    0,
			max:    59,
			radix:  60,
			expect: []int{10, 20},
		},
		{
			expr:   "1/10",
			min:    0,
			max:    59,
			radix:  60,
			expect: []int{},
		},
	}

	for _, c := range cases {
		result, err := parseItem(c.expr, c.min, c.max, c.radix)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if len(result) != len(c.expect) {
			t.Errorf("expected %d items, got %d", len(c.expect), len(result))
			continue
		}
		for i, v := range result {
			if v != c.expect[i] {
				t.Errorf("expected %d, got %d", c.expect[i], v)
			}
		}
	}

}
