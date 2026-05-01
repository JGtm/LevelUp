//go:build integration

// Tests de la migration seed Prestige (challenge_template + preset_arc).
//
// Couvre :
//   - seedTemplates  : first run, idempotence, update de lignes existantes,
//     fichier absent, title_slug manquant, TOML synthétique
//   - seedPresets    : first run, idempotence, fichier absent
//   - seedPrestigeFromTOML : run complet (templates + arcs + steps)
//   - RegisterPrestigeSeedMigration : idempotence du registre
//
// Lancer avec : go test -tags=integration ./internal/migration/ -run TestSeed -v

package migration

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot remonte 4 niveaux depuis apps/go-api/internal/migration/.
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("repoRoot abs: %v", err)
	}
	return abs
}

func haloTomlDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "config", "titles", "halo_infinite")
}

// ─────────── seedTemplates ───────────

func TestSeedTemplates_HaloInfinite_PopulatesRows(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	if err := seedTemplates(db, filepath.Join(haloTomlDir(t), "challenges", "templates.toml")); err != nil {
		t.Fatalf("seedTemplates: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template WHERE title_slug = 'halo_infinite'").Scan(&count)
	if count < 27 {
		t.Errorf("expected >= 27 templates, got %d", count)
	}
}

func TestSeedTemplates_Idempotent(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	path := filepath.Join(haloTomlDir(t), "challenges", "templates.toml")
	if err := seedTemplates(db, path); err != nil {
		t.Fatalf("run1: %v", err)
	}
	var c1 int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template").Scan(&c1)

	if err := seedTemplates(db, path); err != nil {
		t.Fatalf("run2: %v", err)
	}
	var c2 int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template").Scan(&c2)

	if err := seedTemplates(db, path); err != nil {
		t.Fatalf("run3: %v", err)
	}
	var c3 int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template").Scan(&c3)

	if c1 != c2 || c2 != c3 {
		t.Errorf("idempotence: run1=%d run2=%d run3=%d", c1, c2, c3)
	}
}

func TestSeedTemplates_UpdatesChangedRows(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	path := filepath.Join(haloTomlDir(t), "challenges", "templates.toml")
	if err := seedTemplates(db, path); err != nil {
		t.Fatal(err)
	}

	var id string
	db.QueryRow("SELECT id FROM challenge_template ORDER BY id LIMIT 1").Scan(&id)
	if id == "" {
		t.Fatal("aucun template trouvé après seed")
	}

	if _, err := db.Exec("UPDATE challenge_template SET normal_target = 99999 WHERE id = ?", id); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Re-seed → UPSERT doit restaurer la valeur d'origine
	if err := seedTemplates(db, path); err != nil {
		t.Fatal(err)
	}

	var target float64
	db.QueryRow("SELECT normal_target FROM challenge_template WHERE id = ?", id).Scan(&target)
	if target == 99999 {
		t.Errorf("upsert n'a pas mis à jour la ligne modifiée (id=%s)", id)
	}
}

func TestSeedTemplates_MissingFile_ReturnsError(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	if err := seedTemplates(db, "/no/such/path/templates.toml"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSeedTemplates_MissingTitleSlug_ReturnsError(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	dir := t.TempDir()
	content := "[meta]\nschema_version = 1\n\n[[templates]]\nid = \"x\"\nmetric = \"FieldKDA\"\nwindow_type = \"session\"\ncadence = \"daily\"\neval_type = \"cumulative\"\nlabel_en = \"X\"\nlabel_fr = \"X\"\nnormal_target = 1.0\nheroic_target = 2.0\nlegendary_target = 3.0\nmythic_target = 4.0\n"
	path := filepath.Join(dir, "templates.toml")
	os.WriteFile(path, []byte(content), 0644)

	if err := seedTemplates(db, path); err == nil {
		t.Error("expected error for missing title_slug")
	}
}

func TestSeedTemplates_WithSyntheticTOML(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	dir := t.TempDir()
	content := `
[meta]
title_slug = "test_game"

[[templates]]
id = "test_game.daily.kda"
metric = "FieldKDA"
window_type = "session"
cadence = "daily"
eval_type = "cumulative"
label_en = "KDA"
label_fr = "KDA"
normal_target = 1.0
heroic_target = 1.5
legendary_target = 2.0
mythic_target = 3.0

[[templates]]
id = "test_game.weekly.wins"
metric = "FieldWinRate"
window_type = "session"
window_value = "3"
cadence = "weekly"
eval_type = "threshold"
label_en = "Win rate"
label_fr = "Taux de victoire"
normal_target = 40.0
heroic_target = 50.0
legendary_target = 60.0
mythic_target = 70.0
`
	path := filepath.Join(dir, "templates.toml")
	os.WriteFile(path, []byte(content), 0644)

	if err := seedTemplates(db, path); err != nil {
		t.Fatalf("seedTemplates: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template WHERE title_slug = 'test_game'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 templates, got %d", count)
	}

	// mode_filter absent dans TOML → doit valoir "universal"
	var modeFilter string
	db.QueryRow("SELECT mode_filter FROM challenge_template WHERE id = 'test_game.weekly.wins'").Scan(&modeFilter)
	if modeFilter != "universal" {
		t.Errorf("mode_filter default: got %q, want %q", modeFilter, "universal")
	}
}

// ─────────── seedPresets ───────────

func TestSeedPresets_HaloInfinite_PopulatesRows(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	if err := seedPresets(db, filepath.Join(haloTomlDir(t), "arcs", "presets.toml")); err != nil {
		t.Fatalf("seedPresets: %v", err)
	}

	var arcCount, stepCount int
	db.QueryRow("SELECT COUNT(*) FROM preset_arc WHERE title_slug = 'halo_infinite'").Scan(&arcCount)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc_step").Scan(&stepCount)

	if arcCount < 1 {
		t.Errorf("preset_arc: expected >= 1, got %d", arcCount)
	}
	if stepCount < 1 {
		t.Errorf("preset_arc_step: expected >= 1, got %d", stepCount)
	}
}

func TestSeedPresets_Idempotent(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	path := filepath.Join(haloTomlDir(t), "arcs", "presets.toml")
	if err := seedPresets(db, path); err != nil {
		t.Fatal(err)
	}
	var arcs1, steps1 int
	db.QueryRow("SELECT COUNT(*) FROM preset_arc").Scan(&arcs1)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc_step").Scan(&steps1)

	if err := seedPresets(db, path); err != nil {
		t.Fatal(err)
	}
	var arcs2, steps2 int
	db.QueryRow("SELECT COUNT(*) FROM preset_arc").Scan(&arcs2)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc_step").Scan(&steps2)

	if arcs1 != arcs2 || steps1 != steps2 {
		t.Errorf("idempotence: arcs %d→%d steps %d→%d", arcs1, arcs2, steps1, steps2)
	}
}

func TestSeedPresets_MissingFile_ReturnsError(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	if err := seedPresets(db, "/no/such/path/presets.toml"); err == nil {
		t.Error("expected error for missing file")
	}
}

// ─────────── seedPrestigeFromTOML (full) ───────────

func TestSeedPrestigeFromTOML_HaloInfinite_FullRun(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	if err := seedPrestigeFromTOML(db, haloTomlDir(t)); err != nil {
		t.Fatalf("seedPrestigeFromTOML: %v", err)
	}

	var templates, arcs, steps int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template").Scan(&templates)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc").Scan(&arcs)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc_step").Scan(&steps)

	if templates < 27 {
		t.Errorf("challenge_template: expected >= 27, got %d", templates)
	}
	if arcs < 1 {
		t.Errorf("preset_arc: expected >= 1, got %d", arcs)
	}
	if steps < 1 {
		t.Errorf("preset_arc_step: expected >= 1, got %d", steps)
	}
}

func TestSeedPrestigeFromTOML_Idempotent(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	dir := haloTomlDir(t)
	if err := seedPrestigeFromTOML(db, dir); err != nil {
		t.Fatalf("run1: %v", err)
	}
	var tmpl1, arc1, step1 int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template").Scan(&tmpl1)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc").Scan(&arc1)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc_step").Scan(&step1)

	if err := seedPrestigeFromTOML(db, dir); err != nil {
		t.Fatalf("run2: %v", err)
	}
	var tmpl2, arc2, step2 int
	db.QueryRow("SELECT COUNT(*) FROM challenge_template").Scan(&tmpl2)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc").Scan(&arc2)
	db.QueryRow("SELECT COUNT(*) FROM preset_arc_step").Scan(&step2)

	if tmpl1 != tmpl2 || arc1 != arc2 || step1 != step2 {
		t.Errorf("idempotence: templates %d→%d arcs %d→%d steps %d→%d",
			tmpl1, tmpl2, arc1, arc2, step1, step2)
	}
}

func TestSeedPrestigeFromTOML_MissingPresetsFile_ReturnsError(t *testing.T) {
	db := openMemDB(t)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	// Répertoire avec templates.toml valide mais sans arcs/presets.toml
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "challenges"), 0755)
	os.WriteFile(filepath.Join(dir, "challenges", "templates.toml"), []byte("[meta]\ntitle_slug = \"test_game\"\n"), 0644)

	if err := seedPrestigeFromTOML(db, dir); err == nil {
		t.Error("expected error when arcs/presets.toml est absent")
	}
}

// ─────────── RegisterPrestigeSeedMigration ───────────

func TestRegisterPrestigeSeedMigration_Idempotent(t *testing.T) {
	before := len(registry)

	dir := haloTomlDir(t)
	RegisterPrestigeSeedMigration(dir)
	RegisterPrestigeSeedMigration(dir)
	RegisterPrestigeSeedMigration(dir)

	after := len(registry)
	if after-before > 1 {
		t.Errorf("double registration: registry grew by %d (expected 0 or 1)", after-before)
	}
}
