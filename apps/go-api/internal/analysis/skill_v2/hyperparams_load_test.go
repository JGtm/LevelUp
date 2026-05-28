package skill_v2

import (
	"math"
	"testing"
)

func TestLoadPriorsFromHyperparams_EmptyKeepsDefaults(t *testing.T) {
	def := DefaultPriors()
	got := LoadPriorsFromHyperparams(map[string]float64{}, def)
	if got != def {
		t.Errorf("map vide doit retourner les defaults intacts : got %+v, want %+v", got, def)
	}
}

func TestLoadPriorsFromHyperparams_OnlyDrawProbChanges(t *testing.T) {
	def := DefaultPriors()
	got := LoadPriorsFromHyperparams(map[string]float64{"draw_probability_empirical": 0.22}, def)
	if got.DrawProbability != 0.22 {
		t.Errorf("DrawProbability = %v, want 0.22", got.DrawProbability)
	}
	// Le reste est inchangé.
	if got.Mu0 != def.Mu0 || got.Sigma0 != def.Sigma0 || got.Beta != def.Beta || got.Tau != def.Tau {
		t.Errorf("seul DrawProbability doit changer : got %+v vs def %+v", got, def)
	}
}

func TestLoadPriorsFromHyperparams_InvalidDrawProbIgnored(t *testing.T) {
	def := DefaultPriors()
	for _, bad := range []float64{-0.1, 1.0, 1.5} {
		got := LoadPriorsFromHyperparams(map[string]float64{"draw_probability_empirical": bad}, def)
		if got.DrawProbability != def.DrawProbability {
			t.Errorf("draw_prob invalide %v doit être ignoré, got %v want %v", bad, got.DrawProbability, def.DrawProbability)
		}
	}
}

func TestLoadCountHyperparamsFromDB_EmptyKeepsDefaults(t *testing.T) {
	got := LoadCountHyperparamsFromDB(map[string]float64{}, 25.0)
	def := DefaultCountHyperparamsMap()
	for _, ct := range []CountType{CountKill, CountDeath} {
		if got[ct] != def[ct] {
			t.Errorf("map vide : %v doit rester default, got %+v want %+v", ct, got[ct], def[ct])
		}
	}
}

func TestLoadCountHyperparamsFromDB_TypicalMeanReducesToDefaults(t *testing.T) {
	// Pour μ0=25 et une moyenne empirique de 12.5 (typique Slayer), la formule
	// bias = mean - (w_p + w_o)·μ0 doit redonner exactement les défauts
	// (kill bias=0, death bias=25). C'est le garde-fou contre une dérive du
	// modèle quand l'empirique == l'hypothèse de calibration.
	got := LoadCountHyperparamsFromDB(map[string]float64{
		"kill_mean_empirical":  12.5,
		"death_mean_empirical": 12.5,
	}, 25.0)
	def := DefaultCountHyperparamsMap()
	if math.Abs(got[CountKill].Bias-def[CountKill].Bias) > 1e-9 {
		t.Errorf("kill bias = %v, want default %v", got[CountKill].Bias, def[CountKill].Bias)
	}
	if math.Abs(got[CountDeath].Bias-def[CountDeath].Bias) > 1e-9 {
		t.Errorf("death bias = %v, want default %v", got[CountDeath].Bias, def[CountDeath].Bias)
	}
}

func TestLoadCountHyperparamsFromDB_AtypicalMeanShiftsBias(t *testing.T) {
	// Mode à forte cadence (kill_mean=20). bias_kill = 20 - (1 + (-0.5))·25 = 7.5.
	// death_mean=8 → bias_death = 8 - (-1 + 0.5)·25 = 8 + 12.5 = 20.5.
	got := LoadCountHyperparamsFromDB(map[string]float64{
		"kill_mean_empirical":  20.0,
		"death_mean_empirical": 8.0,
	}, 25.0)
	if math.Abs(got[CountKill].Bias-7.5) > 1e-9 {
		t.Errorf("kill bias = %v, want 7.5", got[CountKill].Bias)
	}
	if math.Abs(got[CountDeath].Bias-20.5) > 1e-9 {
		t.Errorf("death bias = %v, want 20.5", got[CountDeath].Bias)
	}
	// Poids et variance inchangés.
	def := DefaultCountHyperparamsMap()
	if got[CountKill].WeightPlayer != def[CountKill].WeightPlayer ||
		got[CountKill].ObservationVar != def[CountKill].ObservationVar {
		t.Errorf("seul Bias doit changer pour kill : %+v", got[CountKill])
	}
}

func TestAppliedHyperparamCount(t *testing.T) {
	cases := []struct {
		params map[string]float64
		want   int
	}{
		{map[string]float64{}, 0},
		{map[string]float64{"draw_probability_empirical": 0.1}, 1},
		{map[string]float64{"kill_mean_empirical": 12, "death_mean_empirical": 11}, 2},
		{map[string]float64{"draw_probability_empirical": 0.1, "kill_mean_empirical": 12, "death_mean_empirical": 11, "match_count_analyzed": 500}, 3},
	}
	for _, c := range cases {
		if got := AppliedHyperparamCount(c.params); got != c.want {
			t.Errorf("AppliedHyperparamCount(%v) = %d, want %d", c.params, got, c.want)
		}
	}
}
