package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestNormalizeMedalKey(t *testing.T) {
	cases := map[string]string{
		"King of the Hill": "king_of_the_hill",
		"Game End":         "game_end",
		"VIP":              "vip",
		"CTF":              "ctf",
		"Heroic":           "heroic",
		"multikill":        "multikill",
		"  Spree  ":        "spree",
		"10":               "10",
		"":                 "",
	}
	for in, want := range cases {
		if got := NormalizeMedalKey(in); got != want {
			t.Errorf("NormalizeMedalKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBaselineMedalCategory(t *testing.T) {
	cat, super, srt := BaselineMedalCategory("spree", 2)
	if cat != "spree" || super != SuperSectionOther || srt != 2 {
		t.Errorf("baseline spree = (%q,%q,%d), want (spree,other,2)", cat, super, srt)
	}
	// medal_type vide → catégorie "other" (aucune médaille perdue).
	cat, super, srt = BaselineMedalCategory("", 0)
	if cat != SuperSectionOther || super != SuperSectionOther || srt != 0 {
		t.Errorf("baseline vide = (%q,%q,%d), want (other,other,0)", cat, super, srt)
	}
}

// TestBaselineResolver_Totality : la baseline retourne TOUJOURS une catégorie et une
// super-section non vides (garde-rail : aucune médaille orpheline silencieuse).
func TestBaselineResolver_Totality(t *testing.T) {
	r := BaselineMedalCategoryResolver{}
	for _, mt := range []string{"spree", "mode", "multikill", "proficiency", "skill", "style", "other", ""} {
		cat, super, _ := r.Resolve(123, mt, 0)
		if cat == "" || super == "" {
			t.Errorf("baseline resolve(%q) → cat=%q super=%q ; les deux doivent être non vides", mt, cat, super)
		}
	}
}

func sampleCatalog() []domain.MedalCatalogRow {
	return []domain.MedalCatalogRow{
		{MedalID: 10, Label: "Killjoy", Difficulty: "Normal", MedalType: "skill", DifficultyIndex: 0, PersonalScore: 5},
		{MedalID: 20, Label: "Double Kill", Difficulty: "Heroic", MedalType: "multikill", DifficultyIndex: 1},
		{MedalID: 30, Label: "Perfect", Difficulty: "Legendary", MedalType: "skill", DifficultyIndex: 2},
	}
}

// TestMergeMedalCatalog_ZeroSurfacing : le catalogue entier est retourné ; une
// médaille jamais obtenue a Count=0 (surfacée), une obtenue a son total.
func TestMergeMedalCatalog_ZeroSurfacing(t *testing.T) {
	earned := []domain.MedalEarnedRow{{MedalID: 10, TotalCount: 3}}
	items := MergeMedalCatalog(sampleCatalog(), earned, nil) // resolver nil → baseline

	if len(items) != 3 {
		t.Fatalf("attendu 3 items (catalogue complet), got %d", len(items))
	}
	byID := map[int64]domain.MedalSummaryItem{}
	for _, it := range items {
		byID[it.MedalID] = it
	}
	if byID[10].Count != 3 {
		t.Errorf("medal 10 Count = %d, want 3", byID[10].Count)
	}
	if byID[20].Count != 0 || byID[30].Count != 0 {
		t.Errorf("medals non obtenues doivent avoir Count=0 (got 20=%d 30=%d)", byID[20].Count, byID[30].Count)
	}
	// Métadonnées reportées + clé difficulté normalisée + catégorie baseline.
	if byID[10].Name != "Killjoy" || byID[10].DifficultyKey != "normal" || byID[10].Category != "skill" {
		t.Errorf("medal 10 = %+v (Name/DifficultyKey/Category inattendus)", byID[10])
	}
	if byID[30].DifficultyRank != 2 || byID[30].PersonalScore != 0 {
		t.Errorf("medal 30 DifficultyRank/PersonalScore inattendus : %+v", byID[30])
	}
}

// TestMergeMedalCatalog_OrphanFallback : une médaille obtenue absente du catalogue
// produit un item fallback (#<id>, catégorie other) — jamais perdue.
func TestMergeMedalCatalog_OrphanFallback(t *testing.T) {
	earned := []domain.MedalEarnedRow{
		{MedalID: 10, TotalCount: 1},
		{MedalID: 999, TotalCount: 7}, // absent du catalogue
	}
	items := MergeMedalCatalog(sampleCatalog(), earned, nil)
	if len(items) != 4 {
		t.Fatalf("attendu 4 items (3 catalogue + 1 orphelin), got %d", len(items))
	}
	var orphan *domain.MedalSummaryItem
	for i := range items {
		if items[i].MedalID == 999 {
			orphan = &items[i]
		}
	}
	if orphan == nil {
		t.Fatal("médaille orpheline 999 absente du résultat")
	}
	if orphan.Count != 7 || orphan.Name != "#999" || orphan.Category != SuperSectionOther {
		t.Errorf("orphelin = %+v, want Count=7 Name=#999 Category=other", *orphan)
	}
}

// TestGroupMedalsByCategory vérifie Earned/Total/TotalCount et l'ordre (other en
// dernier), avec un resolver enrichi factice mappant vers 2 super-sections.
func TestGroupMedalsByCategory(t *testing.T) {
	catalog := []domain.MedalCatalogRow{
		{MedalID: 1, MedalType: "spree", DifficultyIndex: 0},
		{MedalID: 2, MedalType: "spree", DifficultyIndex: 1},
		{MedalID: 3, MedalType: "skill", DifficultyIndex: 0},
	}
	// Resolver factice : spree → super "classics", skill → super "other".
	resolve := func(_ int64, mt string, di int) (string, string, int) {
		if mt == "spree" {
			return "spree", "classics", di
		}
		return "skill", SuperSectionOther, di
	}
	earned := []domain.MedalEarnedRow{{MedalID: 1, TotalCount: 4}} // seule medal 1 obtenue
	items := MergeMedalCatalog(catalog, earned, resolve)
	groups := GroupMedalsByCategory(items)

	if len(groups) != 2 {
		t.Fatalf("attendu 2 catégories, got %d", len(groups))
	}
	// "classics" (spree) avant "other" (skill) — other toujours en dernier.
	if groups[0].Category != "spree" || groups[0].SuperSection != "classics" {
		t.Errorf("groupe 0 = %q/%q, want spree/classics", groups[0].Category, groups[0].SuperSection)
	}
	if groups[1].Category != "skill" || groups[1].SuperSection != SuperSectionOther {
		t.Errorf("groupe 1 = %q/%q, want skill/other", groups[1].Category, groups[1].SuperSection)
	}
	// spree : 2 médailles, 1 obtenue (Earned=1), TotalCount=4.
	g := groups[0]
	if g.Total != 2 || g.Earned != 1 || g.TotalCount != 4 {
		t.Errorf("groupe spree Total/Earned/TotalCount = %d/%d/%d, want 2/1/4", g.Total, g.Earned, g.TotalCount)
	}
}
