package mappings

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// writeTOML écrit un fichier TOML temporaire pour les tests Registry.
func writeTOML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// fieldsTOMLContent retourne un fields.toml minimal valide.
func fieldsTOMLContent(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 10
group = "combat"
`
}

func assetsTOMLContent(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 10
`
}

func outcomesTOMLContent(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"
`
}

func TestRegistry_LoadFromConfigDir_AllFiles(t *testing.T) {
	tmp := t.TempDir()
	mappingsDir := filepath.Join(tmp, "config", "titles", "test_title", "mappings")
	writeTOML(t, mappingsDir, "fields.toml", fieldsTOMLContent("test_title"))
	writeTOML(t, mappingsDir, "assets.toml", assetsTOMLContent("test_title"))
	writeTOML(t, mappingsDir, "outcomes.toml", outcomesTOMLContent("test_title"))

	r := NewRegistry()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	errs := r.LoadFromConfigDir(tmp, []string{"test_title"}, logger)
	if len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}

	// Fields chargés
	if fset, ok := r.Get("test_title"); !ok {
		t.Error("Get fields ok=false")
	} else if fset.SchemaVersion() != 1 {
		t.Errorf("fields schema = %d", fset.SchemaVersion())
	}

	// Assets chargés
	aset, ok := r.GetAssets("test_title")
	if !ok || aset == nil {
		t.Fatal("GetAssets ok=false")
	}
	if a, found := aset.Get("mode", "ranked"); !found {
		t.Error("asset mode.ranked introuvable")
	} else if lbl, _ := a.Label("fr"); lbl != "Classé" {
		t.Errorf("asset label fr = %q", lbl)
	}

	// Outcomes chargés
	oset, ok := r.GetOutcomes("test_title")
	if !ok || oset == nil {
		t.Fatal("GetOutcomes ok=false")
	}
	if o, found := oset.Get("win"); !found {
		t.Error("outcome win introuvable")
	} else if o.ColorToken != "outcome.positive" {
		t.Errorf("outcome color = %q", o.ColorToken)
	}

	// Logs : 3 events mappings_loaded (fields, assets, outcomes).
	logs := buf.String()
	for _, kind := range []string{"fields", "assets", "outcomes"} {
		if !bytes.Contains([]byte(logs), []byte(`"kind":"`+kind+`"`)) {
			t.Errorf("log mappings_loaded kind=%s absent", kind)
		}
	}
}

func TestRegistry_LoadFromConfigDir_FieldsOnly(t *testing.T) {
	// Si seul fields.toml existe, assets/outcomes restent absents sans erreur.
	tmp := t.TempDir()
	mappingsDir := filepath.Join(tmp, "config", "titles", "minimal_title", "mappings")
	writeTOML(t, mappingsDir, "fields.toml", fieldsTOMLContent("minimal_title"))

	r := NewRegistry()
	errs := r.LoadFromConfigDir(tmp, []string{"minimal_title"}, nil)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if _, ok := r.Get("minimal_title"); !ok {
		t.Error("Get fields ok=false")
	}
	if _, ok := r.GetAssets("minimal_title"); ok {
		t.Error("GetAssets devrait être absent")
	}
	if _, ok := r.GetOutcomes("minimal_title"); ok {
		t.Error("GetOutcomes devrait être absent")
	}
}

func TestRegistry_LoadFromConfigDir_FieldsMissing(t *testing.T) {
	// Si fields.toml manque, le titre n'est pas chargé du tout (erreur agrégée).
	tmp := t.TempDir()
	r := NewRegistry()
	errs := r.LoadFromConfigDir(tmp, []string{"absent_title"}, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for missing fields.toml")
	}
	if _, ok := r.Get("absent_title"); ok {
		t.Error("Get devrait être absent quand fields.toml manque")
	}
}

func TestRegistry_LoadFromConfigDir_InvalidAssets(t *testing.T) {
	// fields.toml valide + assets.toml invalide → fields charge, assets erreur agrégée.
	tmp := t.TempDir()
	mappingsDir := filepath.Join(tmp, "config", "titles", "broken_title", "mappings")
	writeTOML(t, mappingsDir, "fields.toml", fieldsTOMLContent("broken_title"))
	writeTOML(t, mappingsDir, "assets.toml", `
[meta]
title_slug = "broken_title"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked" }
display_order = 10
`) // FR manquant → invalide

	r := NewRegistry()
	errs := r.LoadFromConfigDir(tmp, []string{"broken_title"}, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid assets.toml")
	}
	// fields toujours chargé
	if _, ok := r.Get("broken_title"); !ok {
		t.Error("fields devrait être chargé")
	}
	// assets non chargé
	if _, ok := r.GetAssets("broken_title"); ok {
		t.Error("assets ne devrait pas être chargé")
	}
}

func TestRegistry_NilLogger(t *testing.T) {
	// Vérifie que LoadFromConfigDir accepte logger=nil sans paniquer.
	tmp := t.TempDir()
	r := NewRegistry()
	errs := r.LoadFromConfigDir(tmp, []string{"x"}, nil)
	if len(errs) == 0 {
		t.Error("expected error for missing dir")
	}
}

func TestRegistry_GetAssetsAbsentSlug(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.GetAssets("never_loaded"); ok {
		t.Error("GetAssets sur slug inconnu devrait retourner false")
	}
	if _, ok := r.GetOutcomes("never_loaded"); ok {
		t.Error("GetOutcomes sur slug inconnu devrait retourner false")
	}
}
