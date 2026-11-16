package duckdb

import (
	"context"
	"testing"
)

// TestTacticalRepo_Univers_PorteLaCategorieDeMode : l'univers rend, avec la variante, la categorie de mode du
// registre (vide quand elle n'est pas posee) : les deux ensemble donnent la famille de modes d'un match
// (spawn_families.toml), et une variante seule ne distingue pas un BTB d'une Arene.
func TestTacticalRepo_Univers_PorteLaCategorieDeMode(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	tacExec(t, pdb, `UPDATE match_registry SET mode_category = 'BTB', game_variant_name = 'Slayer:BTB Heavies' WHERE match_id = 'm1'`)

	got, err := NewTacticalRepo(pdb).Univers(context.Background(), tacQuery(""))
	if err != nil {
		t.Fatalf("Univers: %v", err)
	}
	vus := map[string][2]string{}
	for _, m := range got.Matchs {
		vus[m.MatchID] = [2]string{m.GameVariantName, m.ModeCategory}
	}
	if vus["m1"] != [2]string{"Slayer:BTB Heavies", "BTB"} {
		t.Errorf("m1 = %v, attendu la variante et la categorie posees", vus["m1"])
	}
	if vus["m2"][1] != "" {
		t.Errorf("m2 sans categorie au registre : %v, attendu vide", vus["m2"])
	}
}
