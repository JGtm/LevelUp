package duckdb

import (
	"slices"
	"testing"
)

// TestMatchsDesLignes_MatchsOuLaLectureARencontreLesRestants : quand les lignes portent leur
// match (Q32, Q32b), la jambe kill-feed ne lit que les matchs ou la lecture a rencontre les
// xuids encore sans nom. La lecture agregee (Q29) passe par match_participants : couverte par
// TestSquadRepo_Annuaire_Q29MemeNomsQueLaVue (integration), ou x_kfl et x_kvp ne sont nommes
// que par le kill-feed.
func TestMatchsDesLignes_MatchsOuLaLectureARencontreLesRestants(t *testing.T) {
	lecture := lectureANommer{
		xuids:    []string{"x_nomme", "x_restant", "x_restant", "x_autre"},
		matchs:   []string{"m1", "m2", "m3", "m4"},
		matchIDs: []string{"m1", "m2", "m3", "m4", "m5"},
	}
	got := lecture.matchsDesLignes([]string{"x_restant"})
	slices.Sort(got)
	if !slices.Equal(got, []string{"m2", "m3"}) {
		t.Errorf("matchs = %v, attendu [m2 m3] (les seuls ou x_restant a ete rencontre)", got)
	}
}
