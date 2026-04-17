package chart

import "testing"

func TestOutcomeColor(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{2, HaloColors.Win},
		{3, HaloColors.Loss},
		{1, HaloColors.Tie},
		{4, HaloColors.Tie},
		{99, "#94a3b8"},
	}
	for _, tt := range tests {
		got := OutcomeColor(tt.code)
		if got != tt.want {
			t.Errorf("OutcomeColor(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestPerfColor(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{90, HaloColors.Perf.High},
		{80, HaloColors.Perf.High},
		{70, HaloColors.Perf.Mid},
		{60, HaloColors.Perf.Mid},
		{50, HaloColors.Perf.Low},
		{40, HaloColors.Perf.Low},
		{30, HaloColors.Perf.Bad},
		{0, HaloColors.Perf.Bad},
	}
	for _, tt := range tests {
		got := PerfColor(tt.score)
		if got != tt.want {
			t.Errorf("PerfColor(%f) = %q, want %q", tt.score, got, tt.want)
		}
	}
}
