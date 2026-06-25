// Package analysis — tier_progression_test.go : tests de la mécanique de progression
// de palier partagée par les citations dérivées (Halo Infinite) et les commendations
// natives (Halo 5). Couvre ComputeTierProgression + ParseTierTargets.
package analysis

import (
	"reflect"
	"testing"
)

func TestParseTierTargets(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []int
	}{
		{"vide", "", nil},
		{"simple croissant", "1,41,120", []int{1, 41, 120}},
		{"désordonné trié", "120,1,41", []int{1, 41, 120}},
		{"espaces et zéro ignoré", " 10 , 0 , 20 ", []int{10, 20}},
		{"non numérique ignoré", "10,abc,30", []int{10, 30}},
		{"négatif ignoré", "-5,10", []int{10}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseTierTargets(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ParseTierTargets(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestComputeTierProgression(t *testing.T) {
	// Paliers façon commendation H5 "Spartan Slayer" : thresholds [1, 41, 120, 300].
	tiers := []int{1, 41, 120, 300}

	cases := []struct {
		name            string
		cumulative      int
		delta           int
		tiers           []int
		wantPct         float64
		wantTierIndex   int
		wantTierCount   int
		wantNextTarget  int
		wantNewMastered bool
		wantAlreadyMast bool
	}{
		{
			// Cumul 20 (delta 20) : palier 1 franchi (tierIndex=1), en route vers 41.
			// pct = (20-1)/(41-1) = 19/40 = 47.5%.
			name: "progression intermédiaire", cumulative: 20, delta: 20, tiers: tiers,
			wantPct: 47.5, wantTierIndex: 1, wantTierCount: 4, wantNextTarget: 41,
		},
		{
			// Cumul pile sur un palier (41) : 2 paliers atteints, prochain = 120.
			// pct = (41-41)/(120-41) = 0%.
			name: "pile sur palier", cumulative: 41, delta: 5, tiers: tiers,
			wantPct: 0.0, wantTierIndex: 2, wantTierCount: 4, wantNextTarget: 120,
		},
		{
			// Franchit le dernier palier CE match (avant=290 < 300, après=305 >= 300).
			name: "newly mastered", cumulative: 305, delta: 15, tiers: tiers,
			wantPct: 100.0, wantTierIndex: 4, wantTierCount: 4, wantNextTarget: 0,
			wantNewMastered: true,
		},
		{
			// Déjà maîtrisé AVANT ce match (avant=320 >= 300) → AlreadyMastered, pas newly.
			name: "déjà maîtrisé", cumulative: 330, delta: 10, tiers: tiers,
			wantPct: 100.0, wantTierIndex: 4, wantTierCount: 4, wantNextTarget: 0,
			wantAlreadyMast: true,
		},
		{
			// Aucun palier connu (Meta/Daily, ou pré-tier_targets) → anneau vide.
			name: "sans paliers", cumulative: 50, delta: 5, tiers: nil,
			wantPct: 0.0, wantTierIndex: 0, wantTierCount: 0, wantNextTarget: 0,
		},
		{
			// Premier palier non encore atteint (cumul 0 < 1) : pct=0, prochain=1.
			name: "avant premier palier", cumulative: 0, delta: 0, tiers: tiers,
			wantPct: 0.0, wantTierIndex: 0, wantTierCount: 4, wantNextTarget: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ComputeTierProgression(c.cumulative, c.delta, c.tiers)
			if got.ProgressPct != c.wantPct {
				t.Errorf("ProgressPct = %v, want %v", got.ProgressPct, c.wantPct)
			}
			if got.TierIndex != c.wantTierIndex {
				t.Errorf("TierIndex = %d, want %d", got.TierIndex, c.wantTierIndex)
			}
			if got.TierCount != c.wantTierCount {
				t.Errorf("TierCount = %d, want %d", got.TierCount, c.wantTierCount)
			}
			if got.NextTierTarget != c.wantNextTarget {
				t.Errorf("NextTierTarget = %d, want %d", got.NextTierTarget, c.wantNextTarget)
			}
			if got.IsNewlyMastered != c.wantNewMastered {
				t.Errorf("IsNewlyMastered = %v, want %v", got.IsNewlyMastered, c.wantNewMastered)
			}
			if got.AlreadyMastered != c.wantAlreadyMast {
				t.Errorf("AlreadyMastered = %v, want %v", got.AlreadyMastered, c.wantAlreadyMast)
			}
		})
	}
}
