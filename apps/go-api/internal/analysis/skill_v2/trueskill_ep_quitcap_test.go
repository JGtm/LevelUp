package skill_v2

import (
	"math"
	"testing"
)

// TestApplyQuitPenaltyPost_Cap vérifie que la pénalité de quit appliquée à μ est
// bornée par QuitPenaltyMuCap (étude volatilité [2026-05-31]) : un quit ne peut
// plus retrancher 2.5 μ d'un coup (= plusieurs sous-paliers).
func TestApplyQuitPenaltyPost_Cap(t *testing.T) {
	cases := []struct {
		name     string
		delta    float64
		wantDrop float64
	}{
		{"unrelated brut 2.5 → capé", DefaultQuitDeltaUnrelated, QuitPenaltyMuCap}, // 2.5 → 1.5
		{"related 1.0 sous le cap", DefaultQuitDeltaRelated, DefaultQuitDeltaRelated},
		{"delta 0 → default related", 0, DefaultQuitDeltaRelated},
		{"delta énorme 10 → capé", 10, QuitPenaltyMuCap},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			team := []Gaussian{{Mu: 25, Sigma: 1}}
			counts := []PlayerCounts{{Quit: true, QuitPenaltyDelta: c.delta}}
			applyQuitPenaltyPost(team, counts)
			gotDrop := 25 - team[0].Mu
			if math.Abs(gotDrop-c.wantDrop) > 1e-9 {
				t.Errorf("drop μ = %v, want %v", gotDrop, c.wantDrop)
			}
		})
	}
}

// TestApplyQuitPenaltyPost_NoQuit : sans quit, μ inchangé.
func TestApplyQuitPenaltyPost_NoQuit(t *testing.T) {
	team := []Gaussian{{Mu: 25, Sigma: 1}}
	applyQuitPenaltyPost(team, []PlayerCounts{{Quit: false, QuitPenaltyDelta: 2.5}})
	if team[0].Mu != 25 {
		t.Errorf("non-quit ne doit pas toucher μ : got %v", team[0].Mu)
	}
}
