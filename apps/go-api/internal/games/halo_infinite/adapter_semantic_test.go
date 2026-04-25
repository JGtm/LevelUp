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
	a := NewSemanticAdapter(set, nil)
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
	a := NewSemanticAdapter(set, nil)
	if got := a.SchemaVersion(); got != 7 {
		t.Errorf("SchemaVersion = %d, want 7", got)
	}
}

func TestSemanticAdapter_Fields_Exposes(t *testing.T) {
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	a := NewSemanticAdapter(set, nil)
	if _, ok := a.Fields().Get(canonical.FieldKills); !ok {
		t.Errorf("FieldKills introuvable via SemanticAdapter")
	}
}

func TestSemanticAdapter_NilSet(t *testing.T) {
	t.Parallel()
	if a := NewSemanticAdapter(nil, nil); a != nil {
		t.Errorf("NewSemanticAdapter(nil, nil) devrait retourner nil")
	}
}

func TestSemanticAdapter_Ranks_NilDefault(t *testing.T) {
	t.Parallel()
	set, _ := mappings.LoadFieldsFromBytes("x.toml", []byte(minimalToml))
	a := NewSemanticAdapter(set, nil)
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
	a := NewSemanticAdapter(set, custom)
	if a.Ranks() != custom {
		t.Errorf("Ranks() doit pointer sur le catalog injecté")
	}
}
