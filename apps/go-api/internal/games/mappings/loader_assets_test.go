package mappings

import (
	"strings"
	"testing"
)

func TestLoadAssetsFromBytes_HappyPath(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50

[assets.challenge_tier.heroic]
labels = { en = "Heroic", fr = "Héroïque" }
color_token = "challenge.heroic"
display_order = 20
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", set.TitleSlug())
	}
	if set.SchemaVersion() != 1 {
		t.Errorf("SchemaVersion = %d, want 1", set.SchemaVersion())
	}

	got, ok := set.Get("mode", "ranked")
	if !ok {
		t.Fatal("Get(mode, ranked) introuvable")
	}
	if lbl, _ := got.Label("fr"); lbl != "Classé" {
		t.Errorf("Label fr = %q, want Classé", lbl)
	}
	if got.DisplayOrder != 50 {
		t.Errorf("DisplayOrder = %d, want 50", got.DisplayOrder)
	}

	tier, ok := set.Get("challenge_tier", "heroic")
	if !ok {
		t.Fatal("Get(challenge_tier, heroic) introuvable")
	}
	if tier.ColorToken != "challenge.heroic" {
		t.Errorf("ColorToken = %q, want challenge.heroic", tier.ColorToken)
	}
}

func TestLoadAssetsFromBytes_AllOfKindOrder(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.bbb]
labels = { en = "BBB", fr = "BBB" }
display_order = 30

[assets.mode.aaa]
labels = { en = "AAA", fr = "AAA" }
display_order = 10

[assets.mode.ccc]
labels = { en = "CCC", fr = "CCC" }
display_order = 20
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	got := set.AllOfKind("mode")
	if len(got) != 3 {
		t.Fatalf("AllOfKind len = %d, want 3", len(got))
	}
	want := []string{"aaa", "ccc", "bbb"}
	for i, m := range got {
		if m.ID != want[i] {
			t.Errorf("position %d: id = %q, want %q", i, m.ID, want[i])
		}
	}
}

func TestLoadAssetsFromBytes_LabelFallback(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50
`)
	set, _ := LoadAssetsFromBytes("test.toml", doc)
	a, _ := set.Get("mode", "ranked")
	// Locale inconnue → fallback EN.
	lbl, used := a.Label("de")
	if lbl != "Ranked" || !used {
		t.Errorf("fallback de: got %q used=%v, want Ranked + true", lbl, used)
	}
}

func TestLoadAssetsFromBytes_LabelFallbackToID(t *testing.T) {
	// Cas dégradé : un set construit à la main sans labels.
	a := AssetMapping{Kind: "mode", ID: "unknown"}
	lbl, used := a.Label("fr")
	if lbl != "unknown" || !used {
		t.Errorf("fallback to id: got %q used=%v, want unknown + true", lbl, used)
	}
}

func TestLoadAssetsFromBytes_MissingMeta(t *testing.T) {
	doc := []byte(`
[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for missing meta")
	}
	if !strings.Contains(err.Error(), "title_slug manquant") {
		t.Errorf("error = %v, want title_slug manquant", err)
	}
}

func TestLoadAssetsFromBytes_MissingLabel(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked" }
display_order = 50
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for missing FR label")
	}
	if !strings.Contains(err.Error(), "label FR manquant") {
		t.Errorf("error = %v, want label FR manquant", err)
	}
}

func TestLoadAssetsFromBytes_DisplayOrderCollision(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.aaa]
labels = { en = "AAA", fr = "AAA" }
display_order = 10

[assets.mode.bbb]
labels = { en = "BBB", fr = "BBB" }
display_order = 10
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for display_order collision")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("error = %v, want collision", err)
	}
}

func TestNewAssetMappingSet_NilMap(t *testing.T) {
	set := NewAssetMappingSet("hi", 1, nil)
	if set.AllOfKind("any") != nil {
		t.Error("nil map should yield nil AllOfKind")
	}
	if len(set.Kinds()) != 0 {
		t.Error("nil map should yield 0 kinds")
	}
}

func TestAssetMappingSet_NilSafe(t *testing.T) {
	var set *AssetMappingSet
	if _, ok := set.Get("mode", "ranked"); ok {
		t.Error("nil set Get should return false")
	}
	if set.AllOfKind("mode") != nil {
		t.Error("nil set AllOfKind should return nil")
	}
	if set.Kinds() != nil {
		t.Error("nil set Kinds should return nil")
	}
}
