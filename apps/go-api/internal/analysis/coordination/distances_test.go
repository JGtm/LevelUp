package coordination_test

import (
	"testing"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// distances_test.go — LA FORME DE LA DISTANCE A L'EQUIPIER (lot F, 2026-09-13).
//
// Ce que ces tests cadenassent, et pourquoi chacun compte :
//   - une mort d'un match SANS RAYON n'entre pas (meme univers que le taux, correction G2) ;
//   - une mort « equipe a terre » n'entre pas (elle ne dit rien du placement) ;
//   - une mort SANS COEQUIPIER VISIBLE n'entre pas dans la distribution mais est COMPTEE :
//     c'est une absence de mesure, jamais une distance infinie ;
//   - la mediane est celle des valeurs mesurees, moyenne des deux centrales sur un
//     effectif pair ;
//   - les intervalles VIDES sont servis, et le dernier est OUVERT (pas de borne haute).

func metres(v float64) *float64 { return &v }

func mort(matchID string, plusProche *float64, visibles, horsDeVue int) domain.MortAExaminer {
	return domain.MortAExaminer{
		MatchID: matchID, PlusProcheM: plusProche, Visibles: visibles, HorsDeVue: horsDeVue,
	}
}

func TestDistances_MedianeEtIntervalles(t *testing.T) {
	rayons := map[string]float64{"m1": 18}
	morts := []domain.MortAExaminer{
		mort("m1", metres(5), 1, 0),
		mort("m1", metres(15), 1, 0),
		mort("m1", metres(25), 1, 0),
		mort("m1", metres(55), 1, 0),
	}

	got := coordination.Distances(morts, rayons)

	if got.N != 4 {
		t.Fatalf("N = %d, attendu 4", got.N)
	}
	if got.Mediane == nil || *got.Mediane != 20 {
		t.Fatalf("mediane = %v, attendu 20 (moyenne de 15 et 25)", got.Mediane)
	}
	if len(got.Distribution) != len(domain.TacticalBornesDistanceM) {
		t.Fatalf("%d intervalles, attendu %d (les vides sont servis)",
			len(got.Distribution), len(domain.TacticalBornesDistanceM))
	}
	attendus := []int{1, 1, 1, 0, 0, 1}
	for i, n := range attendus {
		if got.Distribution[i].N != n {
			t.Errorf("intervalle %d (min %g) = %d, attendu %d",
				i, got.Distribution[i].MinM, got.Distribution[i].N, n)
		}
	}
	dernier := got.Distribution[len(got.Distribution)-1]
	if dernier.MaxM != nil {
		t.Errorf("le dernier intervalle porte une borne haute (%v) : il doit rester OUVERT", *dernier.MaxM)
	}
	if got.Distribution[0].MaxM == nil || *got.Distribution[0].MaxM != 10 {
		t.Errorf("premier intervalle : borne haute %v, attendu 10", got.Distribution[0].MaxM)
	}
}

func TestDistances_AbsenceDeMesureComptee(t *testing.T) {
	rayons := map[string]float64{"m1": 18}
	morts := []domain.MortAExaminer{
		mort("m1", metres(12), 1, 0),
		// Aucun coequipier VISIBLE : il n'y a AUCUNE distance a mesurer. La verser dans
		// « 50+ » inventerait une mesure.
		mort("m1", nil, 0, 2),
	}

	got := coordination.Distances(morts, rayons)

	if got.N != 1 {
		t.Fatalf("N = %d, attendu 1 : la mort sans coequipier visible n'est pas mesuree", got.N)
	}
	if got.MortsSansDistance != 1 {
		t.Fatalf("MortsSansDistance = %d, attendu 1 : elle doit sortir, pas disparaitre",
			got.MortsSansDistance)
	}
	total := 0
	for _, b := range got.Distribution {
		total += b.N
	}
	if total != 1 {
		t.Fatalf("somme des intervalles = %d, attendu 1", total)
	}
}

func TestDistances_HorsUnivers(t *testing.T) {
	rayons := map[string]float64{"m1": 18}
	morts := []domain.MortAExaminer{
		// Match sans rayon mesure : hors de l'univers de la lecture (correction G2).
		mort("m2", metres(9), 1, 0),
		// Equipe a terre : personne ne pouvait accompagner, la mort ne dit rien du placement.
		mort("m1", metres(9), 0, 0),
	}

	got := coordination.Distances(morts, rayons)

	if got.N != 0 || got.MortsSansDistance != 0 {
		t.Fatalf("N = %d, sans distance = %d, attendu 0 et 0", got.N, got.MortsSansDistance)
	}
	if got.Mediane != nil {
		t.Fatalf("mediane = %v, attendu nil : aucune mesure ne vaut mieux qu'un zero", *got.Mediane)
	}
}

func TestDistances_AucuneMort(t *testing.T) {
	got := coordination.Distances(nil, map[string]float64{"m1": 18})

	if got.Mediane != nil {
		t.Fatalf("mediane = %v, attendu nil", *got.Mediane)
	}
	if len(got.Distribution) != len(domain.TacticalBornesDistanceM) {
		t.Fatalf("%d intervalles, attendu %d : la forme de l'histogramme ne depend pas des donnees",
			len(got.Distribution), len(domain.TacticalBornesDistanceM))
	}
}
