package games

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

const genericFieldsToml = `
[meta]
title_slug     = "halo_5"
schema_version = 3

[fields.kills]
labels        = { en = "Kills", fr = "Frags" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"
`

func loadGenericFields(t *testing.T) *mappings.FieldMappingSet {
	t.Helper()
	set, err := mappings.LoadFieldsFromBytes("x.toml", []byte(genericFieldsToml))
	if err != nil {
		t.Fatalf("LoadFieldsFromBytes: %v", err)
	}
	return set
}

func TestGenericSemanticAdapter_NilFields(t *testing.T) {
	if a := NewGenericSemanticAdapter("halo_5", nil, nil, nil, nil); a != nil {
		t.Error("fields nil -> adapter nil attendu (signal boot)")
	}
}

func TestGenericSemanticAdapter_SlugParametrized(t *testing.T) {
	// Le MEME code sert plusieurs titres : seul le slug change (intention DRY).
	for _, slug := range []string{"halo_infinite", "halo_5", "synthetic_title_b"} {
		a := NewGenericSemanticAdapter(slug, loadGenericFields(t), nil, nil, nil)
		if a == nil {
			t.Fatalf("adapter nil pour %q", slug)
		}
		if a.TitleSlug() != slug {
			t.Errorf("TitleSlug = %q, want %q", a.TitleSlug(), slug)
		}
	}
}

func TestGenericSemanticAdapter_NilRanks_EmptyCatalogOfSlug(t *testing.T) {
	a := NewGenericSemanticAdapter("halo_5", loadGenericFields(t), nil, nil, nil)
	ranks := a.Ranks()
	if ranks == nil {
		t.Fatal("Ranks() ne doit jamais retourner nil (catalog vide attendu)")
	}
	if ranks.Len() != 0 {
		t.Errorf("Ranks() = %d, want 0 (catalog vide par defaut)", ranks.Len())
	}
}

func TestGenericSemanticAdapter_PassThrough(t *testing.T) {
	fields := loadGenericFields(t)
	a := NewGenericSemanticAdapter("halo_5", fields, nil, nil, nil)
	if a.SchemaVersion() != 3 {
		t.Errorf("SchemaVersion = %d, want 3", a.SchemaVersion())
	}
	if a.Fields() != fields {
		t.Error("Fields() doit pointer sur le set injecte")
	}
	if _, ok := a.Fields().Get(canonical.FieldKills); !ok {
		t.Error("FieldKills introuvable via Fields()")
	}
	// assets/outcomes non injectes -> nil (degradation gracieuse cote consommateur).
	if a.Assets() != nil || a.Outcomes() != nil {
		t.Error("Assets()/Outcomes() doivent etre nil quand non injectes")
	}
}

func TestGenericSemanticAdapter_RanksPassThrough(t *testing.T) {
	custom := mappings.NewRankCatalog("halo_5", []mappings.RankEntry{
		{ID: 1, Title: map[string]string{"en": "Diamond 5", "fr": "Diamant 5"}},
	})
	a := NewGenericSemanticAdapter("halo_5", loadGenericFields(t), custom, nil, nil)
	if a.Ranks() != custom {
		t.Error("Ranks() doit pointer sur le catalog injecte")
	}
}
