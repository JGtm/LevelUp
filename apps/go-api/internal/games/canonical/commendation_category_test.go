package canonical

import "testing"

// TestNormalizeCommendationCategory_RawHalo5 : les libellés bruts de l'API
// Metadata Halo 5 (EN majuscules, tels que stockés dans
// commendation_definitions.category) deviennent des clés canoniques.
// Les 5 valeurs listées sont l'inventaire complet relevé en base le 2026-08-02.
func TestNormalizeCommendationCategory_RawHalo5(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want string
	}{
		{"MULTIPLAYER", CommendationCategoryMultiplayer},
		{"GAME MODE", CommendationCategoryGameMode},
		{"WEAPON", CommendationCategoryWeapon},
		{"VEHICLE", CommendationCategoryVehicle},
		{"ENEMY", CommendationCategoryEnemy},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeCommendationCategory(tc.raw); got != tc.want {
				t.Errorf("NormalizeCommendationCategory(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeCommendationCategory_RawHaloInfinite : les libellés FR posés au
// seed (citation_mappings.category, internal/ops/seed.go) deviennent les MÊMES
// clés canoniques que leurs équivalents Halo 5 — accents compris.
// Inventaire complet relevé en base le 2026-08-02.
func TestNormalizeCommendationCategory_RawHaloInfinite(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want string
	}{
		{"Multijoueur", CommendationCategoryMultiplayer},
		{"Mode de jeu", CommendationCategoryGameMode},
		{"Arme", CommendationCategoryWeapon},
		{"Véhicule", CommendationCategoryVehicle},
		{"Ennemi", CommendationCategoryEnemy},
		{"Spartan Companies", CommendationCategorySpartanCompanies},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeCommendationCategory(tc.raw); got != tc.want {
				t.Errorf("NormalizeCommendationCategory(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeCommendationCategory_CrossTitleAgreement : les deux titres
// décrivent la même taxonomie dans deux langues — la clé produite doit être
// identique pour chaque paire EN/FR.
func TestNormalizeCommendationCategory_CrossTitleAgreement(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"MULTIPLAYER", "Multijoueur"},
		{"GAME MODE", "Mode de jeu"},
		{"WEAPON", "Arme"},
		{"VEHICLE", "Véhicule"},
		{"ENEMY", "Ennemi"},
	}
	for _, p := range pairs {
		p := p
		t.Run(p[0], func(t *testing.T) {
			t.Parallel()
			en, fr := NormalizeCommendationCategory(p[0]), NormalizeCommendationCategory(p[1])
			if en != fr {
				t.Errorf("désaccord inter-titres : %q -> %q mais %q -> %q", p[0], en, p[1], fr)
			}
			if en == CommendationCategoryOther {
				t.Errorf("%q/%q retombent sur other — mapping manquant", p[0], p[1])
			}
		})
	}
}

// TestNormalizeCommendationCategory_Idempotent : re-normaliser une clé canonique
// ne la change pas (la normalisation est appliquée à plusieurs étages).
func TestNormalizeCommendationCategory_Idempotent(t *testing.T) {
	t.Parallel()
	for _, key := range AllCommendationCategories() {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeCommendationCategory(key); got != key {
				t.Errorf("NormalizeCommendationCategory(%q) = %q, want %q (idempotent)", key, got, key)
			}
		})
	}
}

// TestNormalizeCommendationCategory_CaseAndSpacing : casse, espaces de bord et
// séparateur (espace / underscore) ne changent pas la clé.
func TestNormalizeCommendationCategory_CaseAndSpacing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want string
	}{
		{"game mode", CommendationCategoryGameMode},
		{"Game Mode", CommendationCategoryGameMode},
		{"  GAME_MODE  ", CommendationCategoryGameMode},
		{"vehicule", CommendationCategoryVehicle},
		{"VÉHICULE", CommendationCategoryVehicle},
		{"spartan_companies", CommendationCategorySpartanCompanies},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeCommendationCategory(tc.raw); got != tc.want {
				t.Errorf("NormalizeCommendationCategory(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeCommendationCategory_UnknownFallsBackToOther : vide, sentinelle
// SQL « misc » et valeur inconnue retombent sur « other » — jamais sur un
// libellé humain.
func TestNormalizeCommendationCategory_UnknownFallsBackToOther(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"", "   ", "misc", "AUTRE", "Nouvelle Categorie 2027"} {
		raw := raw
		t.Run("raw="+raw, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeCommendationCategory(raw); got != CommendationCategoryOther {
				t.Errorf("NormalizeCommendationCategory(%q) = %q, want %q", raw, got, CommendationCategoryOther)
			}
		})
	}
}

// TestCommendationCategoryRank_OrdersOtherLast : l'ordre d'affichage est total,
// déterministe, et place « other » en dernier.
func TestCommendationCategoryRank_OrdersOtherLast(t *testing.T) {
	t.Parallel()
	all := AllCommendationCategories()
	if len(all) == 0 {
		t.Fatal("AllCommendationCategories vide")
	}
	if all[len(all)-1] != CommendationCategoryOther {
		t.Errorf("dernière catégorie = %q, want %q", all[len(all)-1], CommendationCategoryOther)
	}
	for i := 1; i < len(all); i++ {
		if CommendationCategoryRank(all[i-1]) >= CommendationCategoryRank(all[i]) {
			t.Errorf("rangs non strictement croissants entre %q et %q", all[i-1], all[i])
		}
	}
	// Clé hors set : classée après tout le reste (déterminisme du tri).
	if CommendationCategoryRank("inconnue") <= CommendationCategoryRank(CommendationCategoryOther) {
		t.Error("une clé inconnue doit être classée après other")
	}
}

// TestAllCommendationCategories_ReturnsCopy : la liste exposée ne doit pas
// permettre de muter l'ordre interne.
func TestAllCommendationCategories_ReturnsCopy(t *testing.T) {
	t.Parallel()
	first := AllCommendationCategories()
	first[0] = "mutated"
	if AllCommendationCategories()[0] == "mutated" {
		t.Error("AllCommendationCategories expose son tableau interne")
	}
}
