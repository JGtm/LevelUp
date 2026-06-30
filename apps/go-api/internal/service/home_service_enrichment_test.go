package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestBuildCommendationSnippets_FilterBeforeTruncate verrouille la parité avec
// analysis.BuildCitationSnippets (Halo Infinite) : le filtre AlreadyMastered doit
// s'appliquer AVANT la borne `limit`. Régression (issue H5 #4) : une commendation
// déjà maîtrisée et en tête du tri occupait un slot du top-N puis était filtrée,
// évinçant une commendation valide au-delà de la borne → moins de snippets que prévu.
func TestBuildCommendationSnippets_FilterBeforeTruncate(t *testing.T) {
	rows := []domain.HomeMatchCommendationRaw{
		// Maîtrisée AVANT ce match : before = Progress-Count = 100-50 = 50 >= palier 40.
		// Count le plus élevé → en tête du tri (occuperait un slot du top-N).
		{ID: "mastered", Name: "Mastered", Count: 50, Progress: 100, TierTargets: "40"},
		// Trois commendations en progression (before = 0 < palier 100).
		{ID: "b", Name: "B", Count: 40, Progress: 40, TierTargets: "100"},
		{ID: "c", Name: "C", Count: 30, Progress: 30, TierTargets: "100"},
		{ID: "d", Name: "D", Count: 20, Progress: 20, TierTargets: "100"},
	}

	got := buildCommendationSnippets(rows, 2)

	if len(got) != 2 {
		t.Fatalf("attendu 2 snippets (filtre AVANT troncature), obtenu %d", len(got))
	}
	for _, s := range got {
		if s.Key == "mastered" {
			t.Errorf("la commendation déjà maîtrisée ne doit pas apparaître : %+v", s)
		}
	}
	// Top-2 non maîtrisées par Count décroissant : B puis C (D évincée par la borne).
	if got[0].Key != "b" || got[1].Key != "c" {
		t.Errorf("attendu [b, c], obtenu [%s, %s]", got[0].Key, got[1].Key)
	}
}

// TestBuildCommendationSnippets_Bounds : robustesse aux bornes (rows vide, limit<=0).
func TestBuildCommendationSnippets_Bounds(t *testing.T) {
	rows := []domain.HomeMatchCommendationRaw{{ID: "a", Count: 1, Progress: 1, TierTargets: "10"}}
	if got := buildCommendationSnippets(nil, 3); got != nil {
		t.Errorf("rows nil → nil attendu, obtenu %v", got)
	}
	if got := buildCommendationSnippets(rows, 0); got != nil {
		t.Errorf("limit=0 → nil attendu, obtenu %v", got)
	}
}
