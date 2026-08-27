// perf_weights_test.go — lot 3 de .ai/PLAN_PERF_NOTE_OBJECTIFS.md (B3.4a).
//
// Verrouille les profils de poids par chaîne : le profil objectif est FIGÉ au gate 0
// du 2026-08-27 (D-J) et ne doit pas dériver silencieusement ; les chaînes
// non-objectif gardent le profil historique, SANS objective_participation.
package skill

import "testing"

// objectiveWeightsFrozen — les 14 poids du profil objectif, recopiés VERBATIM du
// plan (D-C/D-J) plutôt que lus depuis objectiveChainWeights : un test qui lirait la
// source qu'il vérifie ne verrouille rien. Toute modification du profil produit fait
// échouer ce test — c'est le but, elle exige une re-simulation et un nouvel accord.
var objectiveWeightsFrozen = map[string]float64{
	MetricKeyObjectiveParticipation: 0.12,
	MetricKeyKPM:                    0.10,
	MetricKeyKDA:                    0.09,
	MetricKeyAccuracy:               0.03,
	MetricKeyPSPM:                   0.08,
	MetricKeyDPMDeaths:              0.10,
	MetricKeyAPM:                    0.06,
	MetricKeyDPMDamage:              0.06,
	MetricKeyRankPerf:               0.04,
	MetricKeyKillsVsExpected:        0.09,
	MetricKeyDeathsVsExpected:       0.07,
	MetricKeyMedalExploit:           0.06,
	MetricKeyOffensiveConv:          0.09,
	MetricKeyDefensiveResist:        0.05,
}

// TestWeightsForChain_MapsEveryChain couvre les 7 chaînes de performance : les deux
// chaînes de famille objectif prennent le profil objectif, les cinq autres le profil
// par défaut. Une chaîne inconnue retombe sur le défaut (jamais de profil vide).
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
		got := WeightsForChain(c.chain)
		_, hasOSPM := got[MetricKeyObjectiveParticipation]
		if hasOSPM != c.wantObjectif {
			t.Errorf("WeightsForChain(%q) : ospm présent = %v, attendu %v",
				c.chain, hasOSPM, c.wantObjectif)
		}
		if !c.wantObjectif && len(got) != len(RelativeWeights) {
			t.Errorf("WeightsForChain(%q) : %d métriques, attendu le profil par défaut (%d)",
				c.chain, len(got), len(RelativeWeights))
		}
	}
}

// TestWeightsForChain_ObjectiveProfileIsFrozen vérifie les 14 poids un par un, plus
// la somme (1.04, renormalisée à l'usage). C'est le garde-rail du gate 0.
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
			t.Errorf("profil objectif : poids de %q = %v, FIGÉ à %v (gate 0 du 2026-08-27)", key, w, want)
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
	if sum := sumWeights(got); !nearlyEqual(sum, 1.04, 1e-9) {
		t.Errorf("somme des poids objectif = %v, attendu 1.04", sum)
	}
}

// TestWeightsForChain_DefaultProfileUnchanged : le profil des chaînes non-objectif
// est celui d'avant le lot 3 — mêmes 13 métriques, somme 1.01, aucune trace d'ospm.
// C'est la garantie de non-régression des notes slayer/btb/chaos/firefight.
func TestWeightsForChain_DefaultProfileUnchanged(t *testing.T) {
	got := WeightsForChain(PerfChainRankedSlayer)
	if len(got) != 13 {
		t.Errorf("profil par défaut : %d métriques, attendu 13", len(got))
	}
	if _, ok := got[MetricKeyObjectiveParticipation]; ok {
		t.Error("profil par défaut : ospm ne doit PAS y figurer (les chaînes non-objectif ne la calculent pas)")
	}
	for key, want := range RelativeWeights {
		if got[key] != want {
			t.Errorf("profil par défaut : poids de %q = %v, attendu %v", key, got[key], want)
		}
	}
	if sum := sumWeights(got); !nearlyEqual(sum, 1.01, 1e-9) {
		t.Errorf("somme des poids par défaut = %v, attendu 1.01", sum)
	}
	// Les quatre métriques financées par ospm gardent bien leur valeur HISTORIQUE
	// hors famille objectif (elles ne sont abaissées que dans le profil objectif).
	for key, want := range map[string]float64{
		MetricKeyKPM: 0.14, MetricKeyKDA: 0.11, MetricKeyAccuracy: 0.04, MetricKeyPSPM: 0.10,
	} {
		if got[key] != want {
			t.Errorf("profil par défaut : %q = %v, attendu %v (valeur pré-lot 3)", key, got[key], want)
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
