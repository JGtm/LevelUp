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

func TestIsKnownDominanceFlag(t *testing.T) {
	t.Parallel()
	cases := map[DominanceFlag]bool{
		DominanceNone:        true,
		DominanceDomination:  true,
		DominanceHumiliation: true,
		DominanceRemontada:   true,
		DominanceDebandade:   true,
		DominanceContreRem:   true,
		DominanceFlag(99):    false,
		DominanceFlag(-1):    false,
	}
	for in, want := range cases {
		if got := IsKnownDominanceFlag(in); got != want {
			t.Errorf("IsKnownDominanceFlag(%d) = %v, want %v", in, got, want)
		}
	}
}

func TestDominanceFlagValues_Stable(t *testing.T) {
	t.Parallel()
	// Les valeurs int sont stables (stockees en DB). Tout changement = breaking.
	want := map[DominanceFlag]int{
		DominanceNone:        0,
		DominanceDomination:  1,
		DominanceHumiliation: 2,
		DominanceRemontada:   3,
		DominanceDebandade:   4,
		DominanceContreRem:   5,
	}
	for flag, expected := range want {
		if int(flag) != expected {
			t.Errorf("DominanceFlag value drift: %v should equal %d", flag, expected)
		}
	}
}

func TestAllDominanceFlags(t *testing.T) {
	t.Parallel()
	all := AllDominanceFlags()
	if len(all) != 6 {
		t.Errorf("AllDominanceFlags len = %d, want 6", len(all))
	}
	if all[0] != DominanceNone {
		t.Error("DominanceNone should be first in AllDominanceFlags")
	}
}

func TestIsKnownHighlightEventType(t *testing.T) {
	t.Parallel()
	cases := map[HighlightEventType]bool{
		EventKill:                     true,
		EventDeath:                    true,
		EventAssist:                   true,
		EventMedal:                    true,
		EventFinisher:                 true,
		EventClutch:                   true,
		EventFirstKill:                true,
		EventFirstDeath:               true,
		HighlightEventType(""):        false,
		HighlightEventType("unknown"): false,
	}
	for in, want := range cases {
		if got := IsKnownHighlightEventType(in); got != want {
			t.Errorf("IsKnownHighlightEventType(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAllHighlightEventTypes(t *testing.T) {
	t.Parallel()
	all := AllHighlightEventTypes()
	if len(all) != 8 {
		t.Errorf("AllHighlightEventTypes len = %d, want 8", len(all))
	}
	seen := make(map[HighlightEventType]struct{})
	for _, e := range all {
		seen[e] = struct{}{}
	}
	for _, want := range []HighlightEventType{
		EventKill, EventDeath, EventAssist, EventMedal,
		EventFinisher, EventClutch, EventFirstKill, EventFirstDeath,
	} {
		if _, ok := seen[want]; !ok {
			t.Errorf("AllHighlightEventTypes manque %q", want)
		}
	}
}

func TestHighlightEventTypeValues_Stable(t *testing.T) {
	t.Parallel()
	// Les chaines miroitent shared.highlight_events.event_type. Tout changement
	// est breaking pour la lecture DB.
	want := map[HighlightEventType]string{
		EventKill:       "kill",
		EventDeath:      "death",
		EventAssist:     "assist",
		EventMedal:      "medal",
		EventFinisher:   "finisher",
		EventClutch:     "clutch",
		EventFirstKill:  "first_kill",
		EventFirstDeath: "first_death",
	}
	for ev, expected := range want {
		if string(ev) != expected {
			t.Errorf("HighlightEventType value drift: %s should equal %q", ev, expected)
		}
	}
}
