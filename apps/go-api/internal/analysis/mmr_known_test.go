package analysis

import (
	"math"
	"testing"
)

func TestMMRKnown(t *testing.T) {
	cases := []struct {
		v    float64
		want bool
	}{
		{1500, true}, {0.5, true}, {0, false}, {-1, false}, {math.NaN(), false}, {math.Inf(1), false},
	}
	for _, tc := range cases {
		if got := MMRKnown(tc.v); got != tc.want {
			t.Errorf("MMRKnown(%v) = %v, attendu %v", tc.v, got, tc.want)
		}
	}
}

func TestKnownMMR(t *testing.T) {
	zero, level := 0.0, 1500.0
	if KnownMMR(nil) != nil {
		t.Error("nil doit rester nil")
	}
	if KnownMMR(&zero) != nil {
		t.Error("un MMR à 0 est absent")
	}
	if got := KnownMMR(&level); got == nil || *got != 1500 {
		t.Errorf("un MMR réel est conservé, got %v", got)
	}
}

func TestKnownMMRValue(t *testing.T) {
	if KnownMMRValue(0) != nil {
		t.Error("0 est absent")
	}
	if got := KnownMMRValue(1200); got == nil || *got != 1200 {
		t.Errorf("un MMR réel est conservé, got %v", got)
	}
}

func TestSQLKnownMMR(t *testing.T) {
	want := "CASE WHEN p.team_mmr > 0 THEN p.team_mmr END"
	if got := SQLKnownMMR("p.team_mmr"); got != want {
		t.Errorf("SQLKnownMMR = %q, attendu %q", got, want)
	}
}
