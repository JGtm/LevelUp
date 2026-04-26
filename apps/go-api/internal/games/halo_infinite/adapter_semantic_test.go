package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

const minimalToml = `
[meta]
title_slug     = "halo_infinite"
schema_version = 7

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"
`

func TestSemanticAdapter_TitleSlug(t *testing.T) {
	t.Parallel()
	set, err := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	a := NewSemanticAdapter(set, nil, nil, nil)
	if a == nil {
		t.Fatal("nil adapter")
	}
	if a.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q", a.TitleSlug())
	}
}

func TestSemanticAdapter_SchemaVersion(t *testing.T) {
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	a := NewSemanticAdapter(set, nil, nil, nil)
	if got := a.SchemaVersion(); got != 7 {
		t.Errorf("SchemaVersion = %d, want 7", got)
	}
}

func TestSemanticAdapter_Fields_Exposes(t *testing.T) {
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	a := NewSemanticAdapter(set, nil, nil, nil)
	if _, ok := a.Fields().Get(canonical.FieldKills); !ok {
		t.Errorf("FieldKills introuvable via SemanticAdapter")
	}
}

func TestSemanticAdapter_NilSet(t *testing.T) {
	t.Parallel()
	if a := NewSemanticAdapter(nil, nil, nil, nil); a != nil {
		t.Errorf("NewSemanticAdapter(nil, nil, nil, nil) devrait retourner nil")
	}
}

func TestSemanticAdapter_Ranks_NilDefault(t *testing.T) {
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	a := NewSemanticAdapter(set, nil, nil, nil)
	ranks := a.Ranks()
	if ranks == nil {
		t.Fatal("Ranks() ne doit jamais retourner nil (catalog vide attendu)")
	}
	if ranks.Len() != 0 {
		t.Errorf("Ranks() = %d entrées, want 0 (catalog vide par défaut)", ranks.Len())
	}
}

func TestSemanticAdapter_Ranks_PassThrough(t *testing.T) {
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	custom := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 1, Title: map[string]string{"en": "Bronze 1", "fr": "Bronze 1"}},
	})
	a := NewSemanticAdapter(set, custom, nil, nil)
	if a.Ranks() != custom {
		t.Errorf("Ranks() doit pointer sur le catalog injecté")
	}
}

func TestSemanticAdapter_AssetsAndOutcomes_Nil(t *testing.T) {
	// Quand assets/outcomes ne sont pas injectés, les méthodes retournent nil.
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	a := NewSemanticAdapter(set, nil, nil, nil)
	if a.Assets() != nil {
		t.Errorf("Assets() devrait être nil quand non injecté")
	}
	if a.Outcomes() != nil {
		t.Errorf("Outcomes() devrait être nil quand non injecté")
	}
}

func TestSemanticAdapter_AssetsAndOutcomes_Injected(t *testing.T) {
	// Quand assets/outcomes sont injectés, les méthodes retournent les sets.
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	assetsToml := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 10
`)
	assets, err := mappings.LoadAssetsFromBytes("a.toml", assetsToml)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	outcomesToml := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"
`)
	outcomes, err := mappings.LoadOutcomesFromBytes("o.toml", outcomesToml)
	if err != nil {
		t.Fatalf("LoadOutcomesFromBytes: %v", err)
	}
	a := NewSemanticAdapter(set, nil, assets, outcomes)
	if a.Assets() != assets {
		t.Errorf("Assets() doit pointer sur le set injecté")
	}
	if a.Outcomes() != outcomes {
		t.Errorf("Outcomes() doit pointer sur le set injecté")
	}

	// Vérification cross-méthode : la lookup d'un asset connu doit fonctionner.
	got, ok := a.Assets().Get("mode", "ranked")
	if !ok {
		t.Fatal("Get(mode, ranked) introuvable")
	}
	if lbl, _ := got.Label("fr"); lbl != "Classé" {
		t.Errorf("Label fr = %q", lbl)
	}
}
