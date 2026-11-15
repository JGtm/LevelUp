// perf_weights_test.go — profils de poids des notes de performance.
//
// Verrouille les profils de poids par chaîne : le profil de COMBAT des chaînes objectif ne doit
// pas dériver silencieusement (l'objectif et le soutien de l'équipe se greffent dessus, paquet
// objectiveblend) ; les chaînes non-objectif gardent le profil historique.
package skill

import "testing"

// objectiveWeightsFrozen — les 13 poids du profil de combat des chaînes objectif, recopiés
// VERBATIM plutôt que lus depuis objectiveChainWeights : un test qui lirait la source qu'il vérifie
// ne verrouille rien. Toute modification du profil fait échouer ce test — c'est le but, elle exige
// une re-simulation et un nouvel accord.
var objectiveWeightsFrozen = map[string]float64{
	MetricKeyKPM:              0.10,
	MetricKeyKDA:              0.09,
	MetricKeyAccuracy:         0.03,
	MetricKeyPSPM:             0.08,
	MetricKeyDPMDeaths:        0.10,
	MetricKeyAPM:              0.06,
	MetricKeyDPMDamage:        0.06,
	MetricKeyRankPerf:         0.04,
	MetricKeyKillsVsExpected:  0.09,
	MetricKeyDeathsVsExpected: 0.07,
	MetricKeyMedalExploit:     0.06,
	MetricKeyOffensiveConv:    0.09,
	MetricKeyDefensiveResist:  0.05,
}

// TestWeightsForChain_MapsEveryChain couvre les 7 chaînes de performance : les deux chaînes de
// famille objectif prennent le profil objectif (IsObjectiveChain), les cinq autres le profil par
// défaut. Une chaîne inconnue retombe sur le défaut (jamais de profil vide).
func TestWeightsForChain_MapsEveryChain(t *testing.T) {
	cases := []struct {
		chain        string
		wantObjectif bool
	}{
		{LUSRChainArenaObjectif, true},
		{PerfChainRankedObjectif, true},
		{LUSRChainArenaSlayer, false},
		{LUSRChainBTB, false},
		{LUSRChainChaos, false},
		{PerfChainRankedSlayer, false},
		{PerfChainFirefight, false},
		{"", false},
		{PerfChainRanked, false}, // valeur stockée historique, plus jamais émise
	}
	for _, c := range cases {
		if IsObjectiveChain(c.chain) != c.wantObjectif {
			t.Errorf("IsObjectiveChain(%q) = %v, attendu %v", c.chain, !c.wantObjectif, c.wantObjectif)
		}
		got := WeightsForChain(c.chain)
		if !c.wantObjectif && len(got) != len(RelativeWeights) {
			t.Errorf("WeightsForChain(%q) : %d métriques, attendu le profil par défaut (%d)",
				c.chain, len(got), len(RelativeWeights))
		}
	}
}

// TestWeightsForChain_ObjectiveProfileIsFrozen vérifie les 13 poids un par un, plus la somme
// (0.92, renormalisée à l'usage).
func TestWeightsForChain_ObjectiveProfileIsFrozen(t *testing.T) {
	got := WeightsForChain(LUSRChainArenaObjectif)
	if len(got) != len(objectiveWeightsFrozen) {
		t.Fatalf("profil objectif : %d métriques, attendu %d — une métrique a été ajoutée ou retirée",
			len(got), len(objectiveWeightsFrozen))
	}
	for key, want := range objectiveWeightsFrozen {
		if w, ok := got[key]; !ok {
			t.Errorf("profil objectif : métrique %q absente", key)
		} else if w != want {
			t.Errorf("profil objectif : poids de %q = %v, FIGÉ à %v", key, w, want)
		}
	}
	// Les deux chaînes objectif partagent le MÊME profil.
	ranked := WeightsForChain(PerfChainRankedObjectif)
	for key, want := range objectiveWeightsFrozen {
		if ranked[key] != want {
			t.Errorf("ranked_objectif : poids de %q = %v, attendu %v (profil identique à arena_objectif)",
				key, ranked[key], want)
		}
	}
	if sum := sumWeights(got); !nearlyEqual(sum, 0.92, 1e-9) {
		t.Errorf("somme des poids objectif = %v, attendu 0.92", sum)
	}
}

// TestWeightsForChain_DefaultProfileUnchanged : le profil des chaînes non-objectif est celui
// d'avant le lot objectif — mêmes 13 métriques, somme 1.01. Garantie de non-régression des notes
// slayer/btb/chaos/firefight.
func TestWeightsForChain_DefaultProfileUnchanged(t *testing.T) {
	got := WeightsForChain(PerfChainRankedSlayer)
	if len(got) != 13 {
		t.Errorf("profil par défaut : %d métriques, attendu 13", len(got))
	}
	for key, want := range RelativeWeights {
		if got[key] != want {
			t.Errorf("profil par défaut : poids de %q = %v, attendu %v", key, got[key], want)
		}
	}
	if sum := sumWeights(got); !nearlyEqual(sum, 1.01, 1e-9) {
		t.Errorf("somme des poids par défaut = %v, attendu 1.01", sum)
	}
	// Les quatre métriques de combat pur gardent leur valeur HISTORIQUE hors famille objectif
	// (elles ne sont abaissées que dans le profil objectif).
	for key, want := range map[string]float64{
		MetricKeyKPM: 0.14, MetricKeyKDA: 0.11, MetricKeyAccuracy: 0.04, MetricKeyPSPM: 0.10,
	} {
		if got[key] != want {
			t.Errorf("profil par défaut : %q = %v, attendu %v", key, got[key], want)
		}
	}
}

func sumWeights(w map[string]float64) float64 {
	total := 0.0
	for _, v := range w {
		total += v
	}
	return total
}

func nearlyEqual(a, b, eps float64) bool {
	d := a - b
	return d < eps && d > -eps
}
