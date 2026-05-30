package halo

import "testing"

func TestParseISO8601DurationSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"garbage", 0},
		{"P1DT7H50M24.6360455S", 1*86400 + 7*3600 + 50*60 + 24}, // 114624
		{"PT30M", 1800},
		{"PT1H", 3600},
		{"PT45.9S", 45},
		{"P2D", 172800},
		{"PT0S", 0},
	}
	for _, c := range cases {
		if got := parseISO8601DurationSeconds(c.in); got != c.want {
			t.Errorf("parseISO8601DurationSeconds(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
