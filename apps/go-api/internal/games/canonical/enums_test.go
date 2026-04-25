package canonical

import (
	"testing"
)

func TestIsKnownOutcome(t *testing.T) {
	t.Parallel()
	cases := map[Outcome]bool{
		OutcomeWin:         true,
		OutcomeLoss:        true,
		OutcomeTie:         true,
		OutcomeDNF:         true,
		Outcome("invalid"): false,
		Outcome(""):        false,
	}
	for in, want := range cases {
		if got := IsKnownOutcome(in); got != want {
			t.Errorf("IsKnownOutcome(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAllOutcomes(t *testing.T) {
	t.Parallel()
	all := AllOutcomes()
	if len(all) != 4 {
		t.Errorf("AllOutcomes len = %d, want 4", len(all))
	}
	seen := make(map[Outcome]struct{})
	for _, o := range all {
		seen[o] = struct{}{}
	}
	for _, want := range []Outcome{OutcomeWin, OutcomeLoss, OutcomeTie, OutcomeDNF} {
		if _, ok := seen[want]; !ok {
			t.Errorf("AllOutcomes manque %q", want)
		}
	}
}
