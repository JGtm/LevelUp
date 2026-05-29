package skill_v2

import (
	"math"
	"testing"
)

func TestEstimateCouplingMatrix_PerfectlyCorrelated(t *testing.T) {
	// 4 joueurs, μ identiques sur les 3 modes (mais variant entre joueurs) →
	// corrélation parfaite (r=1) → poids capé à Phase4ModeCouplingMaxWeight (0.4).
	ps := map[string][]GroupState{
		"p1": {{"slayer", 10}, {"obj", 10}, {"chaos", 10}},
		"p2": {{"slayer", 20}, {"obj", 20}, {"chaos", 20}},
		"p3": {{"slayer", 30}, {"obj", 30}, {"chaos", 30}},
		"p4": {{"slayer", 40}, {"obj", 40}, {"chaos", 40}},
	}
	m := EstimateCouplingMatrix(ps)
	for _, pair := range [][2]string{{"slayer", "obj"}, {"slayer", "chaos"}, {"obj", "chaos"}} {
		got := m[pair[0]][pair[1]]
		if math.Abs(got-Phase4ModeCouplingMaxWeight) > 1e-9 {
			t.Errorf("coupling[%s][%s] = %v, want %v (corrélation parfaite capée)",
				pair[0], pair[1], got, Phase4ModeCouplingMaxWeight)
		}
		// Symétrie.
		if m[pair[1]][pair[0]] != got {
			t.Errorf("matrice non symétrique pour %s/%s", pair[0], pair[1])
		}
	}
}

func TestEstimateCouplingMatrix_Decorrelated(t *testing.T) {
	// 3 vecteurs centrés mutuellement orthogonaux → toutes corrélations = 0.
	ps := map[string][]GroupState{
		"p1": {{"slayer", 23}, {"obj", 23}, {"chaos", 23}},
		"p2": {{"slayer", 25}, {"obj", 23}, {"chaos", 25}},
		"p3": {{"slayer", 23}, {"obj", 25}, {"chaos", 25}},
		"p4": {{"slayer", 25}, {"obj", 25}, {"chaos", 23}},
	}
	m := EstimateCouplingMatrix(ps)
	for _, pair := range [][2]string{{"slayer", "obj"}, {"slayer", "chaos"}, {"obj", "chaos"}} {
		if got := m[pair[0]][pair[1]]; math.Abs(got) > 1e-9 {
			t.Errorf("coupling[%s][%s] = %v, want 0 (décorrélés)", pair[0], pair[1], got)
		}
	}
}

func TestEstimateCouplingMatrix_NegativeClampedToZero(t *testing.T) {
	// Anti-corrélation parfaite (slayer ↑ ⇒ obj ↓) → clampée à 0 (le leak
	// additif ne modélise pas l'anti-corrélation).
	ps := map[string][]GroupState{
		"p1": {{"slayer", 10}, {"obj", 40}},
		"p2": {{"slayer", 20}, {"obj", 30}},
		"p3": {{"slayer", 30}, {"obj", 20}},
		"p4": {{"slayer", 40}, {"obj", 10}},
	}
	m := EstimateCouplingMatrix(ps)
	if got := m["slayer"]["obj"]; got != 0 {
		t.Errorf("anti-corrélation → coupling = %v, want 0", got)
	}
}

func TestEstimateCouplingMatrix_InsufficientSamples(t *testing.T) {
	// Seulement 2 joueurs ayant les deux modes (< minPlayersForCoupling) → pas
	// d'entrée pour la paire.
	ps := map[string][]GroupState{
		"p1": {{"slayer", 10}, {"obj", 10}},
		"p2": {{"slayer", 20}, {"obj", 20}},
		"p3": {{"slayer", 30}}, // un seul mode
	}
	m := EstimateCouplingMatrix(ps)
	if _, ok := m["slayer"]["obj"]; ok {
		t.Errorf("paire sous le seuil ne doit pas produire d'entrée, got %v", m["slayer"]["obj"])
	}
}

func TestCouplingWeightFor(t *testing.T) {
	hp := map[string]float64{
		ModeCouplingHyperparamName("slayer", "obj"): 0.35,
	}
	if got := CouplingWeightFor(hp, "slayer", "obj", 0.3); got != 0.35 {
		t.Errorf("entrée présente : got %v, want 0.35", got)
	}
	if got := CouplingWeightFor(hp, "slayer", "chaos", 0.3); got != 0.3 {
		t.Errorf("entrée absente → fallback : got %v, want 0.3", got)
	}
}

func TestModeCouplingHyperparamName(t *testing.T) {
	if got := ModeCouplingHyperparamName("arena_slayer", "btb"); got != "mode_coupling_arena_slayer_btb" {
		t.Errorf("got %q", got)
	}
}
