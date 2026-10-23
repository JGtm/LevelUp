package mappings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMapVariants_Valid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[variants]
suffixes = ["- Ranked", " Heavies "]
`)
	set, err := LoadMapVariantsFromBytes("map_variants.toml", raw)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got, want := set.Suffixes(), []string{"- Ranked", "Heavies"}; !reflect.DeepEqual(got, want) {
		t.Errorf("suffixes: got %v, want %v", got, want)
	}
	if got := set.SchemaVersion(); got != 1 {
		t.Errorf("schema_version: got %d", got)
	}
}

func TestLoadMapVariants_Invalid(t *testing.T) {
	cases := map[string]string{
		"slug manquant": "[meta]\nschema_version = 1\n[variants]\nsuffixes = [\"Heavies\"]\n",
		"schema absent": "[meta]\ntitle_slug = \"halo_infinite\"\n[variants]\nsuffixes = [\"Heavies\"]\n",
		"suffixe vide":  "[meta]\ntitle_slug = \"halo_infinite\"\nschema_version = 1\n[variants]\nsuffixes = [\"  \"]\n",
		"toml invalide": "[meta\n",
	}
	for name, raw := range cases {
		if _, err := LoadMapVariantsFromBytes("map_variants.toml", []byte(raw)); err == nil {
			t.Errorf("%s : une erreur est attendue", name)
		}
	}
}

func TestMapVariantSet_NilDeclaresNothing(t *testing.T) {
	var set *MapVariantSet
	if got := set.Suffixes(); got != nil {
		t.Errorf("set nil : aucun suffixe attendu, reçu %v", got)
	}
	if got := set.SchemaVersion(); got != 0 {
		t.Errorf("set nil : version 0 attendue, reçue %d", got)
	}
}

// Le fichier versionné du titre se charge et déclare les deux suffixes mesurés.
func TestLoadMapVariants_FichierHaloInfinite(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "config", "titles", "halo_infinite", "mappings", "map_variants.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("config absente de l'arbre de test : %v", err)
	}
	set, err := LoadMapVariantsFromFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got, want := set.Suffixes(), []string{"- Ranked", "Heavies"}; !reflect.DeepEqual(got, want) {
		t.Errorf("suffixes: got %v, want %v", got, want)
	}
}
