//go:build integration

// reconcile_metadata_seeds_test.go — garde-rail anti-régression : après
// ReconcileMetadataSeeds, une base "ancienne" (migration de seed marquée done mais
// table incomplète) doit contenir les traductions FR des sous-modes du picker et
// la playlist "Quick Play" corrigée. Empêche le retour du bug "mode/playlist en
// anglais". Déplacé depuis internal/migration (Phase 1.5 b7).
//
// CGO requis (driver DuckDB) → tag integration.
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestReconcileMetadataSeeds_ConvergesStaleTranslations(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Base "ancienne" : tables présentes mais seeds de traduction manquants. La
	// playlist Quick Play a sa ligne fr-FR remplie avec l'EN brut (corruption
	// ingestion typique que applyPlaylistFRSeeds doit corriger).
	setup := []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL, PRIMARY KEY (mode_en, lang))`,
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		`INSERT INTO asset_translations VALUES ('1b1691dc-quick','playlist','en-US','Quick Play')`,
		`INSERT INTO asset_translations VALUES ('1b1691dc-quick','playlist','fr-FR','Quick Play')`,
	}
	for _, q := range setup {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}

	if err := ReconcileMetadataSeeds(db); err != nil {
		t.Fatalf("ReconcileMetadataSeeds: %v", err)
	}

	// Sous-modes du picker → FR.
	for _, c := range []struct{ en, wantFR string }{
		{"Slayer", "Assassin"},
		{"Team Slayer", modeTeamSlayerFR},
		{"Neutral Flag CTF", "Drapeau neutre"},
	} {
		var got string
		if err := db.QueryRow(`SELECT name FROM mode_name_tr WHERE mode_en = ? AND lang = 'fr'`, c.en).Scan(&got); err != nil {
			t.Errorf("mode %q absent: %v", c.en, err)
			continue
		}
		if got != c.wantFR {
			t.Errorf("mode %q → %q, want %q", c.en, got, c.wantFR)
		}
	}

	// Playlist Quick Play : fr-FR corrigé en "Partie rapide" (l'EN brut écrasé).
	var pl string
	if err := db.QueryRow(`SELECT name FROM asset_translations WHERE asset_id = '1b1691dc-quick' AND lang = 'fr-FR'`).Scan(&pl); err != nil {
		t.Fatalf("playlist fr-FR: %v", err)
	}
	if pl != "Partie rapide" {
		t.Errorf("playlist Quick Play fr-FR = %q, want Partie rapide", pl)
	}

	// Idempotence : un 2e passage ne casse rien (INSERT OR IGNORE / UPDATE garde-fou).
	if err := ReconcileMetadataSeeds(db); err != nil {
		t.Fatalf("ReconcileMetadataSeeds (2e passage): %v", err)
	}
}
