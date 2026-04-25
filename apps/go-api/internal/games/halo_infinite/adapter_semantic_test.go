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
