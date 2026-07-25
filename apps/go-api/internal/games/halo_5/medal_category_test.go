package halo_5

import "testing"

// TestMedalCategoryResolver_KnownIDs : les IDs présents dans la table wiki sont résolus
// vers leur catégorie enrichie (indépendante du medal_type coarse de la DB, qui collapse
// tous les modes sous « mode » et armes+véhicules sous « proficiency »).
func TestMedalCategoryResolver_KnownIDs(t *testing.T) {
	r := MedalCategoryResolver{}
	cases := []struct {
		id       int64
		category string
		super    string
	}{
		{2078758684, "multikill", "classics"},       // Double Kill
		{2430242797, "spree", "classics"},           // Killing Spree
		{775545297, "weapons", "weapons_equipment"}, // Sniper Kill
		{92444561, "vehicles", "weapons_equipment"}, // Warthog Destroyed
		{3886151616, "ctf", "game_modes"},           // Flag Capture
		{126799571, "infection", "game_modes"},      // Zombie Slayer
		{3565443938, "strongholds", "game_modes"},   // Stronghold Captured
		{2955425834, "warzone", "game_modes"},       // Elite Kill
		{1212900805, "objective", "game_modes"},     // Immortal
		{2966496172, "style", "other"},              // Assassination
	}
	for _, c := range cases {
		cat, super, _ := r.Resolve(c.id, "ignored_medal_type", 0)
		if cat != c.category || super != c.super {
			t.Errorf("Resolve(%d) = (%q,%q), want (%q,%q)", c.id, cat, super, c.category, c.super)
		}
	}
}

// TestMedalCategoryResolver_UnknownID_Baseline : un ID absent de la table retombe sur la
// baseline (medal_type normalisé + super-section "other" + difficultyIndex comme tri).
func TestMedalCategoryResolver_UnknownID_Baseline(t *testing.T) {
	r := MedalCategoryResolver{}
	cat, super, srt := r.Resolve(9000000001, "style", 2) // médaille custom hors table
	if cat != "style" || super != "other" || srt != 2 {
		t.Errorf("Resolve(custom) = (%q,%q,%d), want (style,other,2)", cat, super, srt)
	}
	// medal_type vide → catégorie "other" (aucune médaille perdue).
	cat, super, _ = r.Resolve(123456789, "", 0)
	if cat != "other" || super != "other" {
		t.Errorf("Resolve(inconnu, medal_type vide) = (%q,%q), want (other,other)", cat, super)
	}
}

// validSuperSections : les 4 super-sections partagées avec Halo Infinite.
var validSuperSections = map[string]bool{
	"classics": true, "game_modes": true, "weapons_equipment": true, "other": true,
}

// validCategories : les 11 clés de catégorie émises par la table Halo 5. Chacune DOIT
// avoir une entrée [medals.category.<clé>] dans le manifeste front (medals.toml) — sinon
// l'UI affiche la clé brute (dégradation visible). Cette allowlist interdit qu'un typo de
// génération introduise une catégorie silencieusement non traduite.
var validCategories = map[string]string{
	"multikill":   "classics",
	"spree":       "classics",
	"weapons":     "weapons_equipment",
	"vehicles":    "weapons_equipment",
	"warzone":     "game_modes",
	"infection":   "game_modes",
	"ctf":         "game_modes",
	"oddball":     "game_modes",
	"strongholds": "game_modes",
	"objective":   "game_modes",
	"style":       "other",
}

// TestMedalCategoryTable_Wellformed est le garde-rail de totalité : toute entrée a une
// catégorie non vide parmi les 11 clés connues et une super-section cohérente parmi les 4
// valides. Aucune médaille orpheline silencieuse via un mapping mal formé.
func TestMedalCategoryTable_Wellformed(t *testing.T) {
	if len(medalCategoryTable) < 200 {
		t.Fatalf("table anormalement petite (%d entrées) — génération cassée ?", len(medalCategoryTable))
	}
	for id, e := range medalCategoryTable {
		if e.category == "" {
			t.Errorf("médaille %d : catégorie vide", id)
			continue
		}
		wantSuper, known := validCategories[e.category]
		if !known {
			t.Errorf("médaille %d : catégorie %q hors allowlist (ajouter la traduction medals.toml + cette map)", id, e.category)
			continue
		}
		if e.superSection != wantSuper {
			t.Errorf("médaille %d : catégorie %q attend super-section %q, a %q", id, e.category, wantSuper, e.superSection)
		}
		if !validSuperSections[e.superSection] {
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
