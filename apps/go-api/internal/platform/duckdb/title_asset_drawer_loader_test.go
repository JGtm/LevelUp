package duckdb

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestLoadTeamColorNames_ReadsColor prouve que LoadTeamColorNames lit bien la colonne
// `color` de team_colors (couleur d'identité hex exposée jusqu'au scoreboard front) en
// plus des noms EN/FR. Utilise openAssetMemDB (helper in-memory partagé, non taggué
// integration).
func TestLoadTeamColorNames_ReadsColor(t *testing.T) {
	db := openAssetMemDB(t)
	ctx := context.Background()

	if _, err := db.Exec(ctx, `CREATE TABLE team_colors (
		team_id INTEGER, name_en VARCHAR, name_fr VARCHAR, color VARCHAR
	)`); err != nil {
		t.Fatalf("create team_colors: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO team_colors VALUES
		(0, 'Red', 'Rouge', '#b00000'),
		(1, 'Blue', 'Bleu', '#178dd8'),
		(2, 'Yellow', 'Jaune', NULL)`); err != nil {
		t.Fatalf("insert team_colors: %v", err)
	}

	m := LoadTeamColorNames(ctx, db)
	if len(m) != 3 {
		t.Fatalf("LoadTeamColorNames: attendu 3 entrées, obtenu %d", len(m))
	}
	if got := m[0].Color; got != "#b00000" {
		t.Errorf("team 0 color = %q, want #b00000", got)
	}
	if got := m[1].Color; got != "#178dd8" {
		t.Errorf("team 1 color = %q, want #178dd8", got)
	}
	// COALESCE(color, '') → couleur absente dégrade en "" (jamais fatal).
	if got := m[2].Color; got != "" {
		t.Errorf("team 2 color (NULL) = %q, want \"\"", got)
	}
	// Les noms restent lus en parallèle (non-régression).
	if m[0].NameFR != "Rouge" || m[0].NameEN != "Red" {
		t.Errorf("team 0 names = %q/%q, want Red/Rouge", m[0].NameEN, m[0].NameFR)
	}
}
