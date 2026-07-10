package wire

import (
	"testing"

	"levelup/go-api/internal/progression/profile"
)

func TestPerfComponentAxis(t *testing.T) {
	cases := map[string]string{
		"kills_vs_expected":    "combat",
		"accuracy_delta":       "combat",
		"offensive_conversion": "combat",
		"deaths_vs_expected":   "survival",
		"defensive_resistance": "survival",
		"win_factor":           "score",
		"damage_efficiency":    "impact",
		"medal_exploit":        "impact",
		"inconnu":              "",
	}
	for name, want := range cases {
		if got := perfComponentAxis(name); got != want {
			t.Errorf("perfComponentAxis(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestComponentsToAxes_AggregatesByAxis(t *testing.T) {
	comps := []profile.LUSRComponentBreakdown{
		{Name: "kills_vs_expected", CurrentAvg: 0.8},
		{Name: "accuracy_delta", CurrentAvg: 0.6},     // combat avec kills → moyenne 0.7
		{Name: "deaths_vs_expected", CurrentAvg: 0.3}, // survival
		{Name: "win_factor", CurrentAvg: 0.5},         // score
		{Name: "inconnu", CurrentAvg: 0.9},            // ignoré (pas d'axe)
	}
	axes := componentsToAxes(comps)
	if axes["combat"] != 0.7 {
		t.Errorf("combat = %v, want 0.7 (moyenne 0.8/0.6)", axes["combat"])
	}
	if axes["survival"] != 0.3 {
		t.Errorf("survival = %v, want 0.3", axes["survival"])
	}
	if axes["score"] != 0.5 {
		t.Errorf("score = %v, want 0.5", axes["score"])
	}
	if _, ok := axes["impact"]; ok {
		t.Errorf("impact ne devrait pas exister (aucune composante impact)")
	}
}
