// performance_ospm_test.go — lot 3 de .ai/PLAN_PERF_NOTE_OBJECTIFS.md (B3.4 b/c).
//
// Couvre la métrique `objective_participation` (ospm) : calcul par minute, règle de
// PRÉSENCE (couverture personal_score_awards, D-J), redistribution du poids quand la
// couverture manque, et le cas produit qui motive tout le lot — le match « écrasé
// mais actif à l'objectif ».
package sync

import (
	"math"
	"testing"
)

func ptrF(v float64) *float64 { return &v }

// ── Extraction : ospm = points d'objectif / minutes ─────────────────────────

func TestExtractMatchMetrics_OSPM(t *testing.T) {
	cases := []struct {
		name     string
		row      historyRow
		wantOSPM *float64
	}{
		{
			name:     "couvert : 300 pts sur 10 min → 30/min",
			row:      historyRow{TimePlayedSeconds: 600, ObjectiveScore: ptrF(300)},
			wantOSPM: ptrF(30),
		},
		{
			name:     "couvert à 0 : valeur légitime, PAS une absence",
			row:      historyRow{TimePlayedSeconds: 600, ObjectiveScore: ptrF(0)},
			wantOSPM: ptrF(0),
		},
		{
			name:     "non couvert : métrique absente",
			row:      historyRow{TimePlayedSeconds: 600, ObjectiveScore: nil},
			wantOSPM: nil,
		},
		{
			name:     "durée nulle : même repli 600 s que les autres métriques",
			row:      historyRow{TimePlayedSeconds: 0, ObjectiveScore: ptrF(150)},
			wantOSPM: ptrF(15),
		},
		{
			name:     "durée courte : 120 pts sur 5 min → 24/min",
			row:      historyRow{TimePlayedSeconds: 300, ObjectiveScore: ptrF(120)},
			wantOSPM: ptrF(24),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			row := c.row
			m := extractMatchMetrics(&row)
			switch {
			case c.wantOSPM == nil && m.OSPM != nil:
				t.Fatalf("OSPM = %v, attendu nil (match sans couverture PSA)", *m.OSPM)
			case c.wantOSPM != nil && m.OSPM == nil:
				t.Fatalf("OSPM = nil, attendu %v", *c.wantOSPM)
			case c.wantOSPM != nil && math.Abs(*m.OSPM-*c.wantOSPM) > 1e-9:
				t.Fatalf("OSPM = %v, attendu %v", *m.OSPM, *c.wantOSPM)
			}
		})
	}
}

func TestGetMetricValue_OSPM(t *testing.T) {
	m := &matchMetrics{OSPM: ptrF(12.5)}
	if v, ok := getMetricValue(m, MetricKeyObjectiveParticipation); !ok || v != 12.5 {
		t.Errorf("getMetricValue(ospm) = (%v, %v), attendu (12.5, true)", v, ok)
	}
	if _, ok := getMetricValue(&matchMetrics{}, MetricKeyObjectiveParticipation); ok {
		t.Error("getMetricValue(ospm) sur un match non couvert : attendu ok=false")
	}
}

// TestPrepareHistoryMetrics_OSPMSeriesOnlyCoveredMatches : la série de référence ne
// contient QUE les matchs couverts. Un match non couvert n'y entre pas comme 0 —
// sinon la population de référence serait diluée de faux zéros.
func TestPrepareHistoryMetrics_OSPMSeriesOnlyCoveredMatches(t *testing.T) {
	history := []historyRow{
		{TimePlayedSeconds: 600, ObjectiveScore: ptrF(100)},
		{TimePlayedSeconds: 600, ObjectiveScore: nil},
		{TimePlayedSeconds: 600, ObjectiveScore: ptrF(0)},
		{TimePlayedSeconds: 600, ObjectiveScore: nil},
		{TimePlayedSeconds: 600, ObjectiveScore: ptrF(50)},
	}
	series := prepareHistoryMetrics(history)[MetricKeyObjectiveParticipation]
	if len(series) != 3 {
		t.Fatalf("série ospm : %d valeurs (%v), attendu 3 (les matchs couverts, dont celui à 0)", len(series), series)
	}
	want := []float64{0, 5, 10} // triée croissante : 0, 50/10, 100/10
	for i, w := range want {
		if math.Abs(series[i]-w) > 1e-9 {
			t.Errorf("série ospm[%d] = %v, attendu %v (série triée)", i, series[i], w)
		}
	}
}

// ── Fixtures de scoring ─────────────────────────────────────────────────────

// objectiveHistory construit n matchs de référence à combat ÉTALÉ (du match raté au
// gros match) et à points d'objectif croissants (50 → 50+10*(n-1), soit 5 à 24 pts/min).
//
// L'étalement n'est pas cosmétique : sur une population homogène, un match faible
// tombe au percentile 0 de TOUTES les métriques et sa note vaut 0 quoi qu'il arrive —
// aucune variation ne serait alors observable.
func objectiveHistory(n int, covered bool) []historyRow {
	history := make([]historyRow, n)
	for i := range history {
		history[i] = historyRow{
			TimePlayedSeconds: 600,
			Kills:             float64(3 + i),
			Deaths:            float64(20 - i%15),
			Assists:           float64(1 + i%6),
			DamageDealt:       float64(1000 + 300*i),
			DamageTaken:       float64(6000 - 150*i),
			PersonalScore:     float64(800 + 150*i),
			Accuracy:          float64(25 + i),
		}
		if covered {
			history[i].ObjectiveScore = ptrF(float64(50 + i*10))
		}
	}
	return history
}

// crushedButActive — le match du cas produit : combat écrasé (4 kills, 15 morts,
// dégâts et précision en berne) mais forte activité à l'objectif.
func crushedButActive(objectiveScore *float64) *historyRow {
	return &historyRow{
		TimePlayedSeconds: 600,
		Kills:             4,
		Deaths:            15,
		Assists:           2,
		DamageDealt:       1500,
		DamageTaken:       5000,
		PersonalScore:     1200,
		Accuracy:          30,
		ObjectiveScore:    objectiveScore,
	}
}

func mustScore(t *testing.T, current *historyRow, history []historyRow, weights map[string]float64) float64 {
	t.Helper()
	got := computeRelativePerformanceScore(current, history, weights)
	if got == nil {
		t.Fatal("note nil, attendue non-nil (historique suffisant)")
	}
	return *got
}

// ── Redistribution du poids quand la couverture manque ──────────────────────

// TestComputeRelativePerformanceScore_OSPMAbsentRedistributesWeight : sur un match
// SANS couverture PSA, la note calculée avec le profil objectif est EXACTEMENT celle
// qu'on obtient avec le même profil privé d'ospm. Autrement dit le poids 0.12 sort de
// la renormalisation au lieu de compter comme un zéro — c'est la définition de
// « métrique absente » (D-J).
//
// L'historique, lui, EST couvert : la série ospm existe donc, et seule l'absence sur
// le match courant explique l'exclusion (le test serait trivialement vrai avec un
// historique vide de la métrique).
func TestComputeRelativePerformanceScore_OSPMAbsentRedistributesWeight(t *testing.T) {
	history := objectiveHistory(20, true)
	current := crushedButActive(nil)

	profileWithOSPM := WeightsForChain(LUSRChainArenaObjectif)
	profileWithoutOSPM := make(map[string]float64, len(profileWithOSPM))
	for k, v := range profileWithOSPM {
		if k == MetricKeyObjectiveParticipation {
			continue
		}
		profileWithoutOSPM[k] = v
	}

	withOSPM := mustScore(t, current, history, profileWithOSPM)
	withoutOSPM := mustScore(t, current, history, profileWithoutOSPM)
	if math.Abs(withOSPM-withoutOSPM) > 1e-9 {
		t.Errorf("match non couvert : note %v avec ospm au profil, %v sans — attendues identiques "+
			"(le poids d'ospm doit être redistribué, pas compté comme 0)", withOSPM, withoutOSPM)
	}
}

// TestComputeRelativePerformanceScore_CoveredAtZeroIsScored : un match COUVERT dont
// le joueur n'a marqué aucun point d'objectif est classé (percentile bas), là où un
// match NON couvert ignore la métrique. Les deux notes doivent donc différer, et
// celle du couvert-à-0 être la plus basse — c'est le piège que D-J interdit de
// confondre.
func TestComputeRelativePerformanceScore_CoveredAtZeroIsScored(t *testing.T) {
	history := objectiveHistory(20, true)
	weights := WeightsForChain(LUSRChainArenaObjectif)

	coveredAtZero := mustScore(t, crushedButActive(ptrF(0)), history, weights)
	notCovered := mustScore(t, crushedButActive(nil), history, weights)

	if coveredAtZero == notCovered {
		t.Fatalf("couvert-à-0 et non-couvert rendent la même note (%v) : la couverture n'est pas distinguée", coveredAtZero)
	}
	if coveredAtZero >= notCovered {
		t.Errorf("couvert-à-0 = %v, non-couvert = %v : le match couvert sans action d'objectif "+
			"doit être pénalisé (percentile ospm bas), pas avantagé", coveredAtZero, notCovered)
	}
}

// ── Cas produit : « écrasé mais actif à l'objectif » ────────────────────────

// TestComputeRelativePerformanceScore_ObjectiveActivityLiftsCrushedMatch est la
// raison d'être du lot : un match d'objectif au combat faible mais à forte
// participation remonte. Deux comparaisons, du plus précis au plus global :
//
//	(a) effet d'ospm SEUL — même profil objectif, avec puis sans couverture ;
//	(b) effet du RÉGIME — profil objectif contre profil historique (celui qui
//	    s'appliquait à ce match avant le lot 3).
func TestComputeRelativePerformanceScore_ObjectiveActivityLiftsCrushedMatch(t *testing.T) {
	history := objectiveHistory(20, true)
	objectiveWeights := WeightsForChain(LUSRChainArenaObjectif)

	// 400 pts sur 10 min = 40/min, au-dessus de tout l'historique (5 à 24/min).
	active := mustScore(t, crushedButActive(ptrF(400)), history, objectiveWeights)
	inactive := mustScore(t, crushedButActive(nil), history, objectiveWeights)
	if active <= inactive {
		t.Errorf("(a) match écrasé mais actif à l'objectif : note %v, contre %v sans donnée d'objectif — "+
			"la participation doit faire remonter la note", active, inactive)
	}

	legacy := mustScore(t, crushedButActive(ptrF(400)), history, RelativeWeights)
	if active <= legacy {
		t.Errorf("(b) même match : %v sous le régime objectif contre %v sous le profil historique — "+
			"le nouveau régime doit valoriser ce match", active, legacy)
	}

	// Contre-épreuve : un porteur de combat SANS activité d'objectif ne doit pas
	// être avantagé par le nouveau régime (la note baisse ou reste stable).
	carry := &historyRow{
		TimePlayedSeconds: 600, Kills: 25, Deaths: 4, Assists: 6,
		DamageDealt: 6000, DamageTaken: 2000, PersonalScore: 3500, Accuracy: 60,
		ObjectiveScore: ptrF(0),
	}
	carryObjective := mustScore(t, carry, history, objectiveWeights)
	carryLegacy := mustScore(t, carry, history, RelativeWeights)
	if carryObjective > carryLegacy {
		t.Errorf("contre-épreuve : le porteur de combat sans objectif passe de %v à %v — "+
			"le régime objectif ne doit pas le récompenser davantage", carryLegacy, carryObjective)
	}
}
