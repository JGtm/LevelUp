//go:build integration

// prestige_test.go — tests autonomes des seeds Prestige (déplacés depuis
// internal/migration/steps_metadata_prestige_seed_test.go, Phase 1.5 b10).
//
// Volontairement self-contained : crée le schéma challenge_template/preset_arc/
// preset_arc_step à la main + des TOML synthétiques dans t.TempDir(), sans passer
// par RunForDB ni le registry global. Couvre seedTemplates/seedPresets/
// seedPrestigeFromTOML (populate + idempotence + erreurs fichier/slug manquant).
package migrations

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupPrestigeSchema crée les tables Prestige (base, sans les ALTER additifs qui
// portent des defaults et ne sont pas requis par les INSERT de seed).
func setupPrestigeSchema(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE challenge_template (
			id                  VARCHAR PRIMARY KEY,
			title_slug          VARCHAR NOT NULL,
			metric              VARCHAR NOT NULL,
			window_type         VARCHAR NOT NULL,
			window_value        VARCHAR,
			cadence             VARCHAR NOT NULL,
			eval_type           VARCHAR NOT NULL,
			mode_filter         VARCHAR NOT NULL DEFAULT 'universal',
			label_en            VARCHAR NOT NULL,
			label_fr            VARCHAR NOT NULL,
			description_en      VARCHAR,
			description_fr      VARCHAR,
			normal_target       DOUBLE NOT NULL,
			heroic_target       DOUBLE NOT NULL,
			legendary_target    DOUBLE NOT NULL,
			mythic_target       DOUBLE NOT NULL,
			schema_version      INTEGER NOT NULL DEFAULT 1,
			updated_at          TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE TABLE preset_arc (
			id              VARCHAR PRIMARY KEY,
			title_slug      VARCHAR NOT NULL,
			title_en        VARCHAR NOT NULL,
			title_fr        VARCHAR NOT NULL,
			description_en  VARCHAR,
			description_fr  VARCHAR,
			schema_version  INTEGER NOT NULL DEFAULT 1,
			updated_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE TABLE preset_arc_step (
			preset_arc_id   VARCHAR NOT NULL,
			position        INTEGER NOT NULL,
			template_id     VARCHAR NOT NULL,
			target_tier     VARCHAR NOT NULL,
			PRIMARY KEY (preset_arc_id, position)
		);
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

const templatesTOML = `
[meta]
title_slug = "halo_infinite"

[[templates]]
id = "kills_weekly"
metric = "kills"
window_type = "rolling"
window_value = "7d"
cadence = "weekly"
eval_type = "sum"
mode_filter = "universal"
label_en = "Weekly Kills"
label_fr = "Eliminations hebdo"
description_en = "Kills over a week"
description_fr = "Eliminations sur une semaine"
normal_target = 50.0
heroic_target = 100.0
legendary_target = 200.0
mythic_target = 400.0

[[templates]]
id = "wins_ranked"
metric = "wins"
window_type = "season"
window_value = ""
cadence = "season"
eval_type = "sum"
label_en = "Ranked Wins"
label_fr = "Victoires classees"
description_en = "Wins in ranked"
description_fr = "Victoires en classe"
normal_target = 10.0
heroic_target = 25.0
legendary_target = 50.0
mythic_target = 100.0
`

const presetsTOML = `
[meta]
title_slug = "halo_infinite"

[[arcs]]
id = "arc_slayer"
title_en = "Slayer Arc"
title_fr = "Arc Slayer"
description_en = "Climb via slayer"
description_fr = "Grimper via slayer"

  [[arcs.steps]]
  position = 1
  template_id = "kills_weekly"
  target_tier = "normal"

  [[arcs.steps]]
  position = 2
  template_id = "wins_ranked"
  target_tier = "heroic"
`

// writePrestigeTOML écrit templates.toml + presets.toml sous tomlDir/{challenges,arcs}.
func writePrestigeTOML(t *testing.T, tomlDir, templates, presets string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(tomlDir, "challenges"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tomlDir, "arcs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tomlDir, "challenges", "templates.toml"), []byte(templates), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tomlDir, "arcs", "presets.toml"), []byte(presets), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSeedPrestigeFromTOML_Populates vérifie le seed complet (templates + arcs + steps).
func TestSeedPrestigeFromTOML_Populates(t *testing.T) {
	db := setupPrestigeSchema(t)
	tomlDir := t.TempDir()
	writePrestigeTOML(t, tomlDir, templatesTOML, presetsTOML)

	if err := seedPrestigeFromTOML(db, tomlDir); err != nil {
		t.Fatalf("seed: %v", err)
	}

	assertCount(t, db, "challenge_template", 2)
	assertCount(t, db, "preset_arc", 1)
	assertCount(t, db, "preset_arc_step", 2)

	var labelFR, modeFilter string
	if err := db.QueryRow(`SELECT label_fr, mode_filter FROM challenge_template WHERE id = 'kills_weekly'`).
		Scan(&labelFR, &modeFilter); err != nil {
		t.Fatalf("query template: %v", err)
	}
	if labelFR != "Eliminations hebdo" {
		t.Errorf("label_fr = %q", labelFR)
	}
	// mode_filter vide dans le TOML (wins_ranked) → défaut 'universal'.
	var winsMode string
	if err := db.QueryRow(`SELECT mode_filter FROM challenge_template WHERE id = 'wins_ranked'`).Scan(&winsMode); err != nil {
		t.Fatalf("query wins_ranked: %v", err)
	}
	if winsMode != "universal" {
		t.Errorf("mode_filter défaut = %q, want universal", winsMode)
	}
}

// TestSeedPrestigeFromTOML_Idempotent : deux exécutions → mêmes cardinalités (ON CONFLICT).
func TestSeedPrestigeFromTOML_Idempotent(t *testing.T) {
	db := setupPrestigeSchema(t)
	tomlDir := t.TempDir()
	writePrestigeTOML(t, tomlDir, templatesTOML, presetsTOML)

	if err := seedPrestigeFromTOML(db, tomlDir); err != nil {
		t.Fatalf("seed 1: %v", err)
	}
	if err := seedPrestigeFromTOML(db, tomlDir); err != nil {
		t.Fatalf("seed 2: %v", err)
	}

	assertCount(t, db, "challenge_template", 2)
	assertCount(t, db, "preset_arc", 1)
	assertCount(t, db, "preset_arc_step", 2)
}

// TestSeedTemplates_MissingFile : erreur propre si templates.toml absent.
func TestSeedTemplates_MissingFile(t *testing.T) {
	db := setupPrestigeSchema(t)
	if err := seedTemplates(db, filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("attendu erreur sur fichier manquant")
	}
}

// TestSeedTemplates_MissingTitleSlug : erreur si meta.title_slug vide.
func TestSeedTemplates_MissingTitleSlug(t *testing.T) {
	db := setupPrestigeSchema(t)
	tomlDir := t.TempDir()
	path := filepath.Join(tomlDir, "templates.toml")
	if err := os.WriteFile(path, []byte("[[templates]]\nid = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seedTemplates(db, path); err == nil {
		t.Fatal("attendu erreur sur meta.title_slug manquant")
	}
}

// TestSeedPresets_MissingTitleSlug : erreur si meta.title_slug vide côté arcs.
func TestSeedPresets_MissingTitleSlug(t *testing.T) {
	db := setupPrestigeSchema(t)
	tomlDir := t.TempDir()
	path := filepath.Join(tomlDir, "presets.toml")
	if err := os.WriteFile(path, []byte("[[arcs]]\nid = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seedPresets(db, path); err == nil {
		t.Fatal("attendu erreur sur meta.title_slug manquant")
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Errorf("%s: %d rows, want %d", table, got, want)
	}
}
