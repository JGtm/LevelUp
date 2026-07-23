package halo_infinite

import "testing"

// TestMedalCategoryResolver_KnownIDs : les IDs présents dans la table SpartanRecord
// sont résolus vers leur catégorie enrichie (indépendante du medal_type DB).
func TestMedalCategoryResolver_KnownIDs(t *testing.T) {
	r := MedalCategoryResolver{}
	cases := []struct {
		id       int64
		category string
		super    string
	}{
		{3233952928, "skill", "other"},       // Killjoy (medal_type DB = spree, enrichi = skill)
		{622331684, "multikill", "classics"}, // Double Kill
		{865763896, "game_end", "other"},     // Perfection
	}
	for _, c := range cases {
		cat, super, _ := r.Resolve(c.id, "ignored_medal_type", 3)
		if cat != c.category || super != c.super {
			t.Errorf("Resolve(%d) = (%q,%q), want (%q,%q)", c.id, cat, super, c.category, c.super)
		}
	}
}

// TestMedalCategoryResolver_UnknownID_Baseline : un ID absent de la table retombe sur
// la baseline (medal_type + super-section "other" + difficultyIndex comme tri).
func TestMedalCategoryResolver_UnknownID_Baseline(t *testing.T) {
	r := MedalCategoryResolver{}
	cat, super, srt := r.Resolve(9000000001, "skill", 2) // Vengeur custom, hors table
	if cat != "skill" || super != "other" || srt != 2 {
		t.Errorf("Resolve(custom) = (%q,%q,%d), want (skill,other,2)", cat, super, srt)
	}
	// medal_type vide → catégorie "other" (aucune médaille perdue).
	cat, super, _ = r.Resolve(123456789, "", 0)
	if cat != "other" || super != "other" {
		t.Errorf("Resolve(inconnu, medal_type vide) = (%q,%q), want (other,other)", cat, super)
	}
}

// TestMedalCategoryTable_Wellformed est le garde-rail de totalité : toute entrée de la
// table a une catégorie non vide et une super-section parmi les 4 valides. Aucune
// médaille orpheline silencieuse via un mapping mal formé.
func TestMedalCategoryTable_Wellformed(t *testing.T) {
	validSuper := map[string]bool{
		"classics": true, "game_modes": true, "weapons_equipment": true, "other": true,
	}
	if len(medalCategoryTable) < 100 {
		t.Fatalf("table anormalement petite (%d entrées) — génération cassée ?", len(medalCategoryTable))
	}
	for id, e := range medalCategoryTable {
		if e.category == "" {
			t.Errorf("médaille %d : catégorie vide", id)
		}
		if !validSuper[e.superSection] {
			t.Errorf("médaille %d : super-section %q invalide (attendu classics/game_modes/weapons_equipment/other)", id, e.superSection)
		}
	}
}

// TestMedalCategoryResolver_TotalityOverTable : le resolver retourne toujours une
// catégorie et une super-section non vides pour chaque médaille de la table.
func TestMedalCategoryResolver_TotalityOverTable(t *testing.T) {
	r := MedalCategoryResolver{}
	for id := range medalCategoryTable {
		cat, super, _ := r.Resolve(id, "", 0)
		if cat == "" || super == "" {
			t.Errorf("Resolve(%d) → cat=%q super=%q ; les deux doivent être non vides", id, cat, super)
		}
	}
}
