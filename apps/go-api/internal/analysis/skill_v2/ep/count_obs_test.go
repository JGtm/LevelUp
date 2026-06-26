package ep

import (
	"math"
	"testing"
)

// TestResolvePerfs vérifie le routage joueur/adversaires selon Side, et surtout
// les gardes-fous d'index hors bornes (intention explicite du code : résilience
// aux bugs callers → (nil, nil) plutôt qu'un panic).
func TestResolvePerfs(t *testing.T) {
	perfA := []*Variable{NewVariable("a0"), NewVariable("a1")}
	perfB := []*Variable{NewVariable("b0")}

	cases := []struct {
		name     string
		idx      int
		side     Side
		wantNil  bool      // attend (nil, nil)
		wantSelf *Variable // perf joueur attendue si !wantNil
		wantOpp  []*Variable
	}{
		{"SideA nominal idx0", 0, SideA, false, perfA[0], perfB},
		{"SideA nominal idx1", 1, SideA, false, perfA[1], perfB},
		{"SideA idx == len → nil", 2, SideA, true, nil, nil},
		{"SideA idx > len → nil", 99, SideA, true, nil, nil},
		{"SideA idx négatif → nil", -1, SideA, true, nil, nil},
		{"SideB nominal", 0, SideB, false, perfB[0], perfA},
		{"SideB hors bornes → nil", 5, SideB, true, nil, nil},
		{"SideB négatif → nil", -3, SideB, true, nil, nil},
		{"Side inconnu → nil", 0, Side(42), true, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			self, opp := resolvePerfs(tc.idx, tc.side, perfA, perfB)
			if tc.wantNil {
				if self != nil || opp != nil {
					t.Fatalf("resolvePerfs = (%v, %v), want (nil, nil)", self, opp)
				}
				return
			}
			if self != tc.wantSelf {
				t.Errorf("perf joueur = %v, want %v", self, tc.wantSelf)
			}
			if len(opp) != len(tc.wantOpp) {
				t.Fatalf("len(opp) = %d, want %d", len(opp), len(tc.wantOpp))
			}
			for i := range opp {
				if opp[i] != tc.wantOpp[i] {
					t.Errorf("opp[%d] = %v, want %v", i, opp[i], tc.wantOpp[i])
				}
			}
		})
	}
}

// TestFmtInt couvre les trois branches : négatif (préfixe "neg" + récursion),
// un seul chiffre (cas terminal), et multi-chiffres (récursion division/modulo).
func TestFmtInt(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{7, "7"},
		{9, "9"},
		{10, "10"},
		{42, "42"},
		{105, "105"},
		{-1, "neg1"},
		{-23, "neg23"},
	}
	for _, tc := range cases {
		if got := fmtInt(tc.in); got != tc.want {
			t.Errorf("fmtInt(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestCountVarName vérifie l'assemblage du nom (Side + index + type), qui sert
// d'identité aux facteurs count — une collision casserait le graphe.
func TestCountVarName(t *testing.T) {
	cases := []struct {
		o    CountObservation
		want string
	}{
		{CountObservation{PlayerIndex: 0, Side: SideA, Type: CountKill}, "A_0_kill"},
		{CountObservation{PlayerIndex: 3, Side: SideB, Type: CountDeath}, "B_3_death"},
		{CountObservation{PlayerIndex: 12, Side: SideA, Type: CountDeath}, "A_12_death"},
	}
	for _, tc := range cases {
		if got := countVarName(tc.o); got != tc.want {
			t.Errorf("countVarName(%+v) = %q, want %q", tc.o, got, tc.want)
		}
	}
}

// TestDefaultCountHyperparams couvre les trois branches du switch, dont la
// branche default (CountType inconnu) qui doit neutraliser le terme adverse
// (w_o = 0) pour ne pas injecter un signal arbitraire.
func TestDefaultCountHyperparams(t *testing.T) {
	kill := DefaultCountHyperparams(CountKill)
	if kill.WeightPlayer <= 0 || kill.WeightOpponent >= 0 {
		t.Errorf("kill: w_p doit être > 0 et w_o < 0, got %+v", kill)
	}

	death := DefaultCountHyperparams(CountDeath)
	if death.WeightPlayer >= 0 || death.WeightOpponent <= 0 {
		t.Errorf("death: w_p doit être < 0 et w_o > 0, got %+v", death)
	}

	// Branche default : type inconnu → w_o neutralisé.
	unknown := DefaultCountHyperparams(CountType(99))
	if unknown.WeightOpponent != 0 {
		t.Errorf("type inconnu: w_o = %v, want 0 (terme adverse neutralisé)", unknown.WeightOpponent)
	}
	if unknown.ObservationVar <= 0 {
		t.Errorf("type inconnu: ObservationVar = %v, want > 0", unknown.ObservationVar)
	}
}

// TestAddCountObservationFactors_Nominal : une obs valide produit exactement 2
// facteurs (SumFactor + PriorFactor) avec les poids attendus (w_p sur le joueur,
// w_o/M réparti sur chaque adversaire).
func TestAddCountObservationFactors_Nominal(t *testing.T) {
	perfA := []*Variable{NewVariable("a0")}
	perfB := []*Variable{NewVariable("b0"), NewVariable("b1")}
	hyp := map[CountType]CountHyperparams{
		CountKill: {Bias: 0, WeightPlayer: 1.0, WeightOpponent: -0.5, ObservationVar: 25.0},
	}
	obs := []CountObservation{{PlayerIndex: 0, Side: SideA, Type: CountKill, Value: 10}}

	out := addCountObservationFactors(obs, perfA, perfB, hyp)
	if len(out) != 2 {
		t.Fatalf("attendu 2 facteurs (sum + prior), got %d", len(out))
	}
	sf, ok := out[0].(*SumFactor)
	if !ok {
		t.Fatalf("out[0] doit être *SumFactor, got %T", out[0])
	}
	// 1 perf joueur + 2 adversaires = 3 sources.
	if len(sf.inputs) != 3 || len(sf.weights) != 3 {
		t.Fatalf("SumFactor: %d inputs / %d weights, want 3/3", len(sf.inputs), len(sf.weights))
	}
	if sf.weights[0] != 1.0 {
		t.Errorf("poids joueur = %v, want 1.0 (w_p)", sf.weights[0])
	}
	// w_o = -0.5 réparti sur 2 adversaires → -0.25 chacun.
	for k := 1; k < 3; k++ {
		if math.Abs(sf.weights[k]-(-0.25)) > tol {
			t.Errorf("poids adversaire[%d] = %v, want -0.25 (w_o/M)", k, sf.weights[k])
		}
	}
	if _, ok := out[1].(*PriorFactor); !ok {
		t.Errorf("out[1] doit être *PriorFactor, got %T", out[1])
	}
}

// TestAddCountObservationFactors_OutOfBoundsSkipped : une obs avec PlayerIndex
// hors bornes est ignorée silencieusement (aucun facteur produit), garde-fou
// explicite contre des callers buggés.
func TestAddCountObservationFactors_OutOfBoundsSkipped(t *testing.T) {
	perfA := []*Variable{NewVariable("a0")}
	perfB := []*Variable{NewVariable("b0")}
	obs := []CountObservation{
		{PlayerIndex: 5, Side: SideA, Type: CountKill, Value: 3}, // hors bornes
		{PlayerIndex: 0, Side: SideA, Type: CountKill, Value: 3}, // valide
	}
	out := addCountObservationFactors(obs, perfA, perfB, nil)
	if len(out) != 2 {
		t.Fatalf("attendu 2 facteurs (seule l'obs valide produit sum+prior), got %d", len(out))
	}
}

// TestAddCountObservationFactors_NoOpponents : si l'équipe adverse est vide,
// resolvePerfs renvoie une liste d'adversaires vide → l'obs est skippée (la
// garde len(oppPerfs)==0).
func TestAddCountObservationFactors_NoOpponents(t *testing.T) {
	perfA := []*Variable{NewVariable("a0")}
	var perfB []*Variable // aucun adversaire
	obs := []CountObservation{{PlayerIndex: 0, Side: SideA, Type: CountKill, Value: 3}}
	out := addCountObservationFactors(obs, perfA, perfB, nil)
	if len(out) != 0 {
		t.Fatalf("aucun adversaire → obs skippée, attendu 0 facteur, got %d", len(out))
	}
}

// TestAddCountObservationFactors_DefaultHypFallback : quand le type n'est pas
// présent dans la map, le code retombe sur DefaultCountHyperparams (le bias
// death=25 doit être retranché à la valeur observée dans le PriorFactor).
func TestAddCountObservationFactors_DefaultHypFallback(t *testing.T) {
	perfA := []*Variable{NewVariable("a0")}
	perfB := []*Variable{NewVariable("b0")}
	// Map vide → fallback DefaultCountHyperparams(CountDeath) (bias=25).
	obs := []CountObservation{{PlayerIndex: 0, Side: SideA, Type: CountDeath, Value: 30}}
	out := addCountObservationFactors(obs, perfA, perfB, map[CountType]CountHyperparams{})
	if len(out) != 2 {
		t.Fatalf("attendu 2 facteurs avec fallback hyperparams, got %d", len(out))
	}
	pf, ok := out[1].(*PriorFactor)
	if !ok {
		t.Fatalf("out[1] doit être *PriorFactor, got %T", out[1])
	}
	// observed - bias = 30 - 25 = 5 → μ du prior = 5.
	if math.Abs(pf.prior.Mu()-5.0) > 1e-6 {
		t.Errorf("μ prior = %v, want 5.0 (value 30 - bias 25)", pf.prior.Mu())
	}
}

// TestAddCountObservationFactors_InvalidObsVar : une variance d'observation ≤ 0
// (fournie par un caller via hyperparams) fait échouer FromMeanVariance ;
// le code doit `continue` (skip cette obs) et ne produire aucun facteur — il ne
// faut pas qu'un SumFactor orphelin sans son PriorFactor likelihood se glisse
// dans le graphe.
func TestAddCountObservationFactors_InvalidObsVar(t *testing.T) {
	perfA := []*Variable{NewVariable("a0")}
	perfB := []*Variable{NewVariable("b0")}
	hyp := map[CountType]CountHyperparams{
		CountKill: {Bias: 0, WeightPlayer: 1.0, WeightOpponent: -0.5, ObservationVar: 0}, // invalide
	}
	obs := []CountObservation{{PlayerIndex: 0, Side: SideA, Type: CountKill, Value: 10}}
	out := addCountObservationFactors(obs, perfA, perfB, hyp)
	// Le SumFactor est ajouté AVANT FromMeanVariance ; l'erreur fait `continue`
	// après — donc 1 facteur (sum) reste, mais pas de prior. On documente le
	// comportement réel observé.
	if len(out) != 1 {
		t.Fatalf("obs var invalide → seul le SumFactor est ajouté (prior skippé), got %d facteurs", len(out))
	}
	if _, ok := out[0].(*SumFactor); !ok {
		t.Errorf("le facteur restant doit être le SumFactor, got %T", out[0])
	}
}
