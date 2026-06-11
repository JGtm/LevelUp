package domain

import "testing"

// TestHaloInfiniteAchievementCategories_Counts : les listes sources couvrent les
// 144 succès Halo Infinite (34 MP + 94 campagne + 16 autres). La table normalisée
// compte 143 clés : "Together. Again?" et "Together. Again." partagent la même
// clé normalisée (même catégorie, sans impact fonctionnel).
func TestHaloInfiniteAchievementCategories_Counts(t *testing.T) {
	if got := len(haloInfiniteMultiplayerAchievements); got != 34 {
		t.Errorf("multiplayer: attendu 34 noms, obtenu %d", got)
	}
	if got := len(haloInfiniteCampaignAchievements); got != 94 {
		t.Errorf("campaign: attendu 94 noms, obtenu %d", got)
	}
	if got := len(haloInfiniteOtherAchievements); got != 16 {
		t.Errorf("other: attendu 16 noms, obtenu %d", got)
	}

	counts := map[AchievementCategory]int{}
	for _, cat := range haloInfiniteAchievementCategories {
		counts[cat]++
	}
	if counts[AchievementCategoryMultiplayer] != 34 {
		t.Errorf("table multiplayer: attendu 34 clés, obtenu %d", counts[AchievementCategoryMultiplayer])
	}
	if counts[AchievementCategoryCampaign] != 93 {
		t.Errorf("table campaign: attendu 93 clés (94 noms - 1 collision Together. Again), obtenu %d",
			counts[AchievementCategoryCampaign])
	}
	if counts[AchievementCategoryOther] != 16 {
		t.Errorf("table other: attendu 16 clés, obtenu %d", counts[AchievementCategoryOther])
	}
}

// TestHaloInfiniteAchievementCategories_NoCrossCategoryCollision : aucune clé
// normalisée ne doit apparaître dans deux listes sources de catégories
// différentes (sinon l'ordre d'insertion déciderait silencieusement).
func TestHaloInfiniteAchievementCategories_NoCrossCategoryCollision(t *testing.T) {
	seen := map[string]AchievementCategory{}
	check := func(names []string, cat AchievementCategory) {
		for _, n := range names {
			key := normalizeAchievementName(n)
			if prev, ok := seen[key]; ok && prev != cat {
				t.Errorf("collision inter-catégories sur %q (clé %q) : %s vs %s", n, key, prev, cat)
			}
			seen[key] = cat
		}
	}
	check(haloInfiniteMultiplayerAchievements, AchievementCategoryMultiplayer)
	check(haloInfiniteCampaignAchievements, AchievementCategoryCampaign)
	check(haloInfiniteOtherAchievements, AchievementCategoryOther)
}

// TestAchievementCategoryFor_Normalization : la normalisation absorbe les
// variations typographiques constatées entre le guide Steam et la DB.
func TestAchievementCategoryFor_Normalization(t *testing.T) {
	tests := []struct {
		name string
		want AchievementCategory
	}{
		// apostrophes typographiques (U+2019) vs droites
		{"You’re Up, Rook’", AchievementCategoryMultiplayer},
		// ellipse Unicode vs trois points ASCII
		{"One Down…", AchievementCategoryCampaign},
		{"One Down...", AchievementCategoryCampaign},
		// espace parasite en fin de nom (constaté en DB sur "All-Seeing I ")
		{"All-Seeing I ", AchievementCategoryCampaign},
		// virgule présente en DB ("Run Rabbit, Run") mais absente du guide
		{"Run Rabbit Run", AchievementCategoryCampaign},
		// guillemets littéraux du nom Xbox
		{"\"Need a Weapon?\"", AchievementCategoryOther},
		// casse différente
		{"medic!", AchievementCategoryMultiplayer},
		// Winter Update (succès co-op post-guide, mappés campagne)
		{"Wolves at the Doors", AchievementCategoryCampaign},
	}
	for _, tc := range tests {
		got, unmapped := AchievementCategoryFor("halo_infinite", tc.name)
		if got != tc.want || unmapped {
			t.Errorf("AchievementCategoryFor(halo_infinite, %q) = (%q, %v), attendu (%q, false)",
				tc.name, got, unmapped, tc.want)
		}
	}
}

// TestAchievementCategoryFor_Limits : nom inconnu → other + unmapped ;
// titre sans mapping → catégorie vide.
func TestAchievementCategoryFor_Limits(t *testing.T) {
	got, unmapped := AchievementCategoryFor("halo_infinite", "Some Future DLC Achievement")
	if got != AchievementCategoryOther || !unmapped {
		t.Errorf("nom inconnu: attendu (other, true), obtenu (%q, %v)", got, unmapped)
	}

	got, unmapped = AchievementCategoryFor("halo_5", "Clocking In")
	if got != "" || unmapped {
		t.Errorf("titre sans mapping: attendu (\"\", false), obtenu (%q, %v)", got, unmapped)
	}
}
