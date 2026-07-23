package analysis

import (
	"math"
	"testing"
)

func fPtr(f float64) *float64 { return &f }

func TestExpectedFDA(t *testing.T) {
	cases := []struct {
		name                       string
		killsExp, deathsExp, assum *float64
		want                       *float64
	}{
		{"nominal_kda", fPtr(10), fPtr(5), fPtr(6), fPtr(7)},     // 10 + 6/3 - 5 = 7
		{"assists_nil_kd_only", fPtr(10), fPtr(5), nil, fPtr(5)}, // 10 + 0 - 5 = 5
		{"assists_inf_treated_zero", fPtr(10), fPtr(5), fPtr(math.Inf(1)), fPtr(5)},
		{"assists_nan_treated_zero", fPtr(8), fPtr(3), fPtr(math.NaN()), fPtr(5)},
		{"kills_nil", nil, fPtr(5), fPtr(6), nil},
		{"deaths_nil", fPtr(10), nil, fPtr(6), nil},
		{"kills_pos_inf", fPtr(math.Inf(1)), fPtr(5), fPtr(6), nil},
		{"kills_neg_inf", fPtr(math.Inf(-1)), fPtr(5), fPtr(6), nil},
		{"deaths_nan", fPtr(10), fPtr(math.NaN()), fPtr(6), nil},
		{"negative_diff", fPtr(3), fPtr(9), fPtr(3), fPtr(-5)}, // 3 + 1 - 9 = -5
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ExpectedFDA(c.killsExp, c.deathsExp, c.assum)
			if c.want == nil {
				if got != nil {
					t.Fatalf("ExpectedFDA = %v, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ExpectedFDA = nil, want %v", *c.want)
			}
			if math.Abs(*got-*c.want) > 1e-9 {
				t.Fatalf("ExpectedFDA = %v, want %v", *got, *c.want)
			}
		})
	}
}

func TestFDADiff(t *testing.T) {
	cases := []struct {
		name             string
		actual, expected *float64
		want             *float64
	}{
		{"nominal", fPtr(7.5), fPtr(5.0), fPtr(2.5)},
		{"negative", fPtr(2.0), fPtr(5.0), fPtr(-3.0)},
		{"actual_nil", nil, fPtr(5.0), nil},
		{"expected_nil", fPtr(7.0), nil, nil},
		{"actual_nan", fPtr(math.NaN()), fPtr(5.0), nil},
		{"expected_inf", fPtr(7.0), fPtr(math.Inf(1)), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FDADiff(c.actual, c.expected)
			if c.want == nil {
				if got != nil {
					t.Fatalf("FDADiff = %v, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("FDADiff = nil, want %v", *c.want)
			}
			if math.Abs(*got-*c.want) > 1e-9 {
				t.Fatalf("FDADiff = %v, want %v", *got, *c.want)
			}
		})
	}
}
