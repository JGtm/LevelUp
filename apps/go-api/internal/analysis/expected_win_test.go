package analysis

import "testing"

func TestExpectedVsActual(t *testing.T) {
	t.Run("nominal", func(t *testing.T) {
		exp, act := ExpectedVsActual([]float64{0.4, 0.6}, 3, 5)
		if exp == nil {
			t.Fatalf("expected non-nil")
		}
		if *exp != 0.5 {
			t.Errorf("expected avg = %v, want 0.5", *exp)
		}
		if act != 0.6 {
			t.Errorf("actual = %v, want 0.6", act)
		}
	})
	t.Run("no predictions -> expected nil", func(t *testing.T) {
		exp, act := ExpectedVsActual(nil, 2, 4)
		if exp != nil {
			t.Errorf("expected nil, got %v", *exp)
		}
		if act != 0.5 {
			t.Errorf("actual = %v, want 0.5", act)
		}
	})
	t.Run("empty scope -> actual 0", func(t *testing.T) {
		exp, act := ExpectedVsActual(nil, 0, 0)
		if exp != nil {
			t.Errorf("expected nil")
		}
		if act != 0 {
			t.Errorf("actual = %v, want 0", act)
		}
	})
}

func TestAggregateKDA(t *testing.T) {
	cases := []struct {
		name                          string
		kills, assists, deaths, games int
		want                          float64
	}{
		{"zero games", 10, 3, 5, 0, 0},
		{"positive net", 30, 9, 12, 6, (30 + 9.0/3 - 12) / 6}, // (30+3-12)/6 = 3.5
		{"negative net", 5, 0, 20, 5, (5 + 0 - 20) / 5.0},     // -3
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AggregateKDA(c.kills, c.assists, c.deaths, c.games)
			if got != c.want {
				t.Errorf("AggregateKDA(%d,%d,%d,%d) = %v, want %v", c.kills, c.assists, c.deaths, c.games, got, c.want)
			}
		})
	}
}
