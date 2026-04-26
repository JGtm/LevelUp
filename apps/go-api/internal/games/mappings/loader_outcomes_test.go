package mappings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOutcomesFromBytes_HappyPath(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Loss", fr = "Défaite" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"

[outcomes.dnf]
labels = { en = "DNF", fr = "Abandon" }
color_token = "outcome.neutral"
`)
	set, err := LoadOutcomesFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadOutcomesFromBytes: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", set.TitleSlug())
	}
	got, ok := set.Get("win")
	if !ok {
		t.Fatal("Get(win) introuvable")
	}
	if lbl, _ := got.Label("fr"); lbl != "Victoire" {
		t.Errorf("Label fr = %q, want Victoire", lbl)
	}
	if got.ColorToken != "outcome.positive" {
		t.Errorf("ColorToken = %q, want outcome.positive", got.ColorToken)
	}
}

func TestLoadOutcomesFromBytes_AllSorted(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Loss", fr = "Défaite" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"
`)
	set, _ := LoadOutcomesFromBytes("test.toml", doc)
	all := set.All()
	want := []string{"loss", "tie", "win"}
	if len(all) != len(want) {
		t.Fatalf("len = %d, want %d", len(all), len(want))
	}
	for i, o := range all {
		if o.Key != want[i] {
			t.Errorf("position %d: key = %q, want %q", i, o.Key, want[i])
		}
	}
}

func TestLoadOutcomesFromBytes_FallbackEnUsed(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"
`)
	set, _ := LoadOutcomesFromBytes("test.toml", doc)
	o, _ := set.Get("win")
	lbl, used := o.Label("de")
	if lbl != "Win" || !used {
		t.Errorf("fallback locale: got %q used=%v, want Win + true", lbl, used)
	}
	// fallback total à la key
	o2 := OutcomeMapping{Key: "ghost"}
	lbl, used = o2.Label("fr")
	if lbl != "ghost" || !used {
		t.Errorf("fallback to key: got %q used=%v", lbl, used)
	}
}

func TestLoadOutcomesFromBytes_UnknownKey(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.totally_invented]
labels = { en = "?", fr = "?" }
color_token = "outcome.neutral"
`)
	_, err := LoadOutcomesFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for unknown outcome key")
	}
	if !strings.Contains(err.Error(), "totally_invented") {
		t.Errorf("error = %v, want mention of totally_invented", err)
	}
}

func TestLoadOutcomesFromBytes_MissingColorToken(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
`)
	_, err := LoadOutcomesFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for missing color_token")
	}
	if !strings.Contains(err.Error(), "color_token") {
		t.Errorf("error = %v, want color_token", err)
	}
}

func TestLoadOutcomesFromBytes_MissingFRLabel(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win" }
color_token = "outcome.positive"
`)
	_, err := LoadOutcomesFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for missing FR label")
	}
	if !strings.Contains(err.Error(), "label FR manquant") {
		t.Errorf("error = %v, want label FR manquant", err)
	}
}

func TestNewOutcomeMappingSet_NilMap(t *testing.T) {
	set := NewOutcomeMappingSet("hi", 1, nil)
	if len(set.All()) != 0 {
		t.Error("nil map should yield empty All")
	}
	if len(set.Keys()) != 0 {
		t.Error("nil map should yield empty Keys")
	}
}

func TestOutcomeMappingSet_NilSafe(t *testing.T) {
	var set *OutcomeMappingSet
	if _, ok := set.Get("win"); ok {
		t.Error("nil set Get should return false")
	}
	if set.All() != nil {
		t.Error("nil set All should return nil")
	}
	if set.Keys() != nil {
		t.Error("nil set Keys should return nil")
	}
}

func TestLoadOutcomesFromFile_FileNotFound(t *testing.T) {
	_, err := LoadOutcomesFromFile(filepath.Join(t.TempDir(), "absent.toml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("error = %v, want read prefix", err)
	}
}

func TestLoadOutcomesFromFile_RealFile(t *testing.T) {
	tomlPath := filepath.Join(t.TempDir(), "outcomes.toml")
	doc := []byte(`
[meta]
title_slug = "smoke"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"
`)
	if err := os.WriteFile(tomlPath, doc, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	set, err := LoadOutcomesFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadOutcomesFromFile: %v", err)
	}
	if set.TitleSlug() != "smoke" {
		t.Errorf("TitleSlug = %q", set.TitleSlug())
	}
}
