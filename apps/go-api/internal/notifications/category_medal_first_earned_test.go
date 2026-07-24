package notifications

import "testing"

// TestAllCategories_IncludesMedalFirstEarned garde-rail V72-20 : la catégorie
// medal_first_earned est énumérée dans AllCategories (couverture du seed / des
// réglages). Une catégorie non seedée est ACTIVE par défaut côté repo
// (isCategoryEnabledOn : ErrNoRows → true), donc aucune migration de seed n'est
// requise — ce test verrouille seulement l'énumération (parité avec le patron
// TestAllCategories_IncludesRivalEncounter).
func TestAllCategories_IncludesMedalFirstEarned(t *testing.T) {
	for _, c := range AllCategories() {
		if c == CategoryMedalFirstEarned {
			return
		}
	}
	t.Fatalf("CategoryMedalFirstEarned absente de AllCategories()")
}
