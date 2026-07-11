package temporal_test

import (
	"testing"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
)

func TestDefaultEventWeights_MatchHistoricalValues(t *testing.T) {
	t.Parallel()
	w := temporal.DefaultEventWeights()
	cases := map[string]float64{
		"mode":                        1.5,
		string(canonical.EventAssist): 0.5,
		string(canonical.EventDeath):  0.0,
		string(canonical.EventKill):   1.0,
		string(canonical.EventMedal):  1.0,
		"":                            1.0,
		"unknown_type":                1.0,
	}
	for et, want := range cases {
		if got := w.For(et); got != want {
			t.Errorf("For(%q) = %v, want %v", et, got, want)
		}
	}
}

func TestEventWeights_IsZero(t *testing.T) {
	t.Parallel()
	if !(temporal.EventWeights{}).IsZero() {
		t.Error("zero-value EventWeights devrait etre IsZero")
	}
	if temporal.DefaultEventWeights().IsZero() {
		t.Error("DefaultEventWeights ne doit pas etre IsZero")
	}
}

// Poids explicites = defaut → score/résidu byte-identiques à ceux du chemin défaut.
func TestComputeEngagement_ExplicitDefaultWeightsByteIdentical(t *testing.T) {
	t.Parallel()
	base := engagementInputFixture()

	base.Weights = temporal.EventWeights{}
	got1, err := temporal.ComputeEngagementScore(base)
	if err != nil {
		t.Fatalf("compute (zero weights) err = %v", err)
	}

	base.Weights = temporal.DefaultEventWeights()
	got2, err := temporal.ComputeEngagementScore(base)
	if err != nil {
		t.Fatalf("compute (default weights) err = %v", err)
	}

	if got1.ResidualBrut != got2.ResidualBrut {
		t.Errorf("ResidualBrut diffère: %v vs %v (défaut explicite doit être byte-identique)", got1.ResidualBrut, got2.ResidualBrut)
	}
	if got1.MeanPaceJoueur != got2.MeanPaceJoueur || got1.MeanPaceLobby != got2.MeanPaceLobby {
		t.Errorf("paces diffèrent avec poids défaut explicite")
	}
}

// Poids objectif plus élevés → paces différentes (le levier agit).
func TestComputeEngagement_DifferentWeightsChangePace(t *testing.T) {
	t.Parallel()
	base := engagementInputFixture()
	base.Weights = temporal.DefaultEventWeights()
	def, _ := temporal.ComputeEngagementScore(base)

	heavy := base
	heavy.Weights = temporal.EventWeights{Objective: 3.0, Assist: 1.0, Death: 0.0, Default: 2.0}
	got, _ := temporal.ComputeEngagementScore(heavy)

	if def.MeanPaceJoueur == got.MeanPaceJoueur {
		t.Errorf("des poids différents devraient changer MeanPaceJoueur (%v)", got.MeanPaceJoueur)
	}
}
