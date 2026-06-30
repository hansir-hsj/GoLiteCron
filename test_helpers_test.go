package golitecron

import "testing"

func mustCronParser(t *testing.T, expr string, opts ...ParseOption) *CronParser {
	t.Helper()

	parser, err := newCronParser(expr, opts...)
	if err != nil {
		t.Fatalf("newCronParser(%q) failed: %v", expr, err)
	}

	return parser
}
