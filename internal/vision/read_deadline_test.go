package vision

import "testing"

// #300: page reads carry a bounded, caller-controlled deadline instead of a
// fixed 5s that discards a whole research flow when one page is slow.
func TestReadDeadlineMSClampsCallerBudget(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"unset uses the 5s default", 0, 5000},
		{"negative falls back to the default", -1, 5000},
		{"caller budget is honoured", 15000, 15000},
		{"below the floor is clamped up", 250, 1000},
		{"above the ceiling is clamped down", 60000, 30000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := ReadOptions{TimeoutMs: tc.in}
			if got := ReadDeadlineMS(opts); got != tc.want {
				t.Fatalf("ReadDeadlineMS(%d)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}
